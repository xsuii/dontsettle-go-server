package xserver

import (
	"code.google.com/p/go-uuid/uuid"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"time"
)

const (
	SeqLength = 10
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
	taskId      string // it presentate a UUID
	path        string
	rFile       *os.File
	wFile       *os.File
	fileInfo    FileInfo
	window      int // size of stored file pieces
	convergence int // size of downed file pieces
}

type FileManager struct {
	addTask   chan *FileTask
	delTask   chan *FileTask
	fileTasks map[string]*FileTask // each task has a task id(FileId field in FileSeq Struct)
	taskCount int
	fileSeq   chan *FileSeq
	syncCh    chan bool
}

func NewFileManager() *FileManager {
	logger.Tracef("Create New File Manager.")
	addTask := make(chan *FileTask)
	delTask := make(chan *FileTask)
	fileTasks := make(map[string]*FileTask)
	taskCount := 0
	fileSeq := make(chan *FileSeq)
	syncCh := make(chan bool)
	return &FileManager{
		addTask,
		delTask,
		fileTasks,
		taskCount,
		fileSeq,
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
			logger.Debug("Del file task.")
			err := dt.wFile.Close()
			if err != nil {
				logger.Error("Close write-file error:", err.Error())
			}
			err = dt.rFile.Close()
			if err != nil {
				logger.Error("Close read-file error:", err.Error())
			}
			delete(fm.fileTasks, dt.taskId)
		case fs := <-fm.fileSeq:
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
				}
			} else {
				logger.Warn("You operate on a nil task.")
			}
			logger.Debug("Write pieces done.")
		}
	}
}

// create file task and wrap ticket
func (fm *FileManager) NewFileTaskAndFileTicket(pack *Pack) (*FileTask, error) {
	var fInfo FileInfo
	err := json.Unmarshal(pack.Body, &fInfo)
	if err != nil {
		return nil, err
	}

	dirPath := getFilePath(pack.Sender, pack.Reciever, pack.TimeStamp) // store path = ./repository/sender/Reciever/timestamp

	f, err := fm.CreateFile(dirPath, fInfo) // create file
	if err != nil {
		return nil, err
	}

	ti := uuid.New() // UUID
	ftk := &FileTicket{
		TaskId:   ti,
		FileInfo: fInfo,
		//ReqTimeStamp: pack.TimeStamp,
	}
	bd, err := json.Marshal(&ftk)
	if err != nil {
		return nil, err
	}
	pack.Body = bd // change the body, which add file task id in it

	return &FileTask{
		taskId:      ti,
		path:        dirPath + fInfo.FileName,
		rFile:       nil,
		wFile:       f,
		fileInfo:    fInfo,
		window:      0,
		convergence: 0,
	}, nil
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
