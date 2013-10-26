package xserver

import (
	"code.google.com/p/go-uuid/uuid"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"time"
)

var _ = time.Second

const (
	SeqLength = 5
)

var _ = time.Second
var _ = fmt.Printf
var _ = runtime.GOOS

type FileInfo struct {
	FileName string
	FileSize int
}

type FileTicket struct {
	TaskId   string
	FileInfo FileInfo
	//ReqTimeStamp int64 // the time when request send
}

type FileSeq struct {
	TaskId     string
	SeqNum     int
	SeqContent string
	SeqSize    int
}

type FileTask struct {
	taskId      string // it presentate a UUID(e.g. 4b156b8c-1751-4382-a8dd-ba2779b087e0)
	path        string
	rFile       *os.File
	wFile       *os.File
	fileInfo    FileInfo
	window      int // size of stored file pieces // window & convergence use for synchronous
	convergence int // size of downed file pieces
	receiver    uint64
	// [TODO] addTime int
}

// [TODO] The task list should store into data base, in order
// to recover when reboot the server.
type FileManager struct {
	server     *Server
	addTask    chan *FileTask
	delTask    chan *FileTask
	fileTasks  map[string]*FileTask // each task has a task id(FileId field in FileSeq Struct)
	taskCount  int
	fileUpLd   chan *FileSeq
	fileDownLd chan string
	syncCh     chan bool
}

func NewFileManager(s *Server) *FileManager {
	logger.Tracef("Create New File Manager.")
	server := s
	addTask := make(chan *FileTask)
	delTask := make(chan *FileTask)
	fileTasks := make(map[string]*FileTask)
	taskCount := 0
	fileUpLd := make(chan *FileSeq)
	fileDownLd := make(chan string)
	syncCh := make(chan bool)
	return &FileManager{
		server,
		addTask,
		delTask,
		fileTasks,
		taskCount,
		fileUpLd,
		fileDownLd,
		syncCh,
	}
}

func (fm *FileManager) FileRoute() {
	logger.Debug("File manager routing.")
	for {
		select {
		case at := <-fm.addTask: // add file store task
			logger.Debug("Add new file task.")
			fm.fileTasks[at.taskId] = at

			// {store to database} //

			logger.Tracef("Show task list:")
			for _, ft := range fm.fileTasks {
				logger.Tracef("Task list:{ TaskId:%v, FileInfo:%v, Window:%v, Convergence:%v }",
					ft.taskId,
					ft.fileInfo,
					ft.window,
					ft.convergence)
			}

			logger.Debug("Add task done.")
		case dt := <-fm.delTask: // del file store task
			// [TODO] Actually, it should not delete too soon, for
			// re-download. Instead, set some deadline.
			logger.Infof("Del file task:{%v}", dt.taskId)
			err := dt.rFile.Close()
			if err != nil {
				logger.Error("Close read-file error:", err.Error())
			}
			delete(fm.fileTasks, dt.taskId)
		case fs := <-fm.fileUpLd:
			ft := fm.fileTasks[fs.TaskId]
			if ft != nil {
				if ft.window < ft.fileInfo.FileSize {
					logger.Tracef("Write pieces to:%v", ft.wFile.Name())
					_, err := ft.wFile.Write([]byte(fs.SeqContent))
					if err != nil {
						logger.Error(err.Error())
						break
					}
					ft.window += fs.SeqSize
				} else {
					err := ft.wFile.Close() // window == fileSize
					if err != nil {
						logger.Error("Close write-file error:", err.Error())
					}
				}
			} else {
				// send to client
				logger.Warn("You operate on a nil task.")
			}
			logger.Debug("Write pieces done.")
			time.Sleep(1 * time.Second)
		case fd := <-fm.fileDownLd:
			go fm.downloadFile(fd)
		}
	}
}

// create file task and wrap ticket
func (fm *FileManager) NewFileTaskAndFileTicket(pack *Pack) (*FileTask, *Pack, error) {
	var fInfo FileInfo
	err := json.Unmarshal(pack.Body, &fInfo)
	if err != nil {
		return nil, nil, err
	}

	dirPath := getFilePath(pack.Sender, pack.Reciever, pack.TimeStamp) // store path = ./repository/sender/Reciever/timestamp

	f, err := fm.CreateFile(dirPath, fInfo) // create file
	if err != nil {
		return nil, nil, err
	}

	ti := uuid.New() // UUID
	ftk := &FileTicket{
		TaskId:   ti,
		FileInfo: fInfo,
		//ReqTimeStamp: pack.TimeStamp,
	}
	bd, err := json.Marshal(&ftk)
	if err != nil {
		return nil, nil, err
	}
	pack.Body = bd // change the body, which add file task id in it
	pack.OpCode = OpFileTicket

	return &FileTask{
		taskId:      ti,
		path:        dirPath + fInfo.FileName,
		rFile:       nil,
		wFile:       f,
		fileInfo:    fInfo,
		window:      0,
		convergence: 0,
		receiver:    pack.Reciever,
	}, pack, nil
}

func getFilePath(id1 uint64, id2 uint64, ts int64) string {
	return "./repository/" + idToString(id1) + "/" + idToString(id2) + "/" + int64ToString(ts) + "/"
}

func (fm *FileManager) CreateFile(dirPath string, fi FileInfo) (*os.File, error) {
	// create file
	err := os.MkdirAll(dirPath, 0777)
	if err != nil {
		return nil, err
	}

	f, err := os.Create(dirPath + fi.FileName) // file name. it should be deleted if exist or add TimeStamp as filename
	if err != nil {
		return nil, err
	}
	return f, err
}

func (fm *FileManager) downloadFile(tId string) {
	var err error
	logger.Info("Begin downloading file.")

	ft := fm.fileTasks[tId]
	if ft == nil {
		logger.Error("No such file task.")
		return
	}
	logger.Tracef("Show task:%v", ft)
	ft.rFile, err = os.Open(ft.path)
	if err != nil {
		logger.Error(err.Error())
	}

	var num int
	b := make([]byte, SeqLength)
	for {
		if ft.convergence < ft.window {
			n, err := ft.rFile.Read(b)
			if err != nil {
				logger.Error(err.Error())
			}
			logger.Tracef("Read %v bytes:{%v}", n, string(b[:n]))
			ft.convergence += n

			fs := &FileSeq{
				TaskId:     tId,
				SeqNum:     num,
				SeqSize:    n,
				SeqContent: string(b[0:n]),
			}
			logger.Tracef("Show download sequence:%v", fs)
			bd, err := json.Marshal(fs)
			if err != nil {
				logger.Error(err.Error())
			}
			p := fm.server.NewPack(MasterId, ft.receiver, getTimeStamp(), OpFileDownld, bd)
			logger.Tracef("Show package:%v", string(p.Body))
			fm.server.toOne <- p

			num++
		} else if ft.window < ft.fileInfo.FileSize {
			runtime.Gosched()
		} else {
			logger.Infof("Download {%v} complete.", ft.fileInfo.FileName)
			fm.delTask <- ft
			return
		}
	}
	logger.Info("Download end.")
}
