// Copyright 2013 xsuii. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//
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
var _ = fmt.Printf
var _ = runtime.GOOS

const ( // [TODO]something constant here might move to configure file later.
	SeqLength = 5

	Delay          = 0
	FileCleanCycle = 5 * time.Second

	Day          = 24 * time.Hour
	Week         = 7 * Day
	FileDeadline = 10 * time.Minute // [Conf]
)

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
	// CheckSum string // [TODO] md5sum or something
}

type FileSync struct {
	window      int
	convergence int
	downld      bool
}

type FileTask struct {
	taskId      string // it presentate a UUID(e.g. 4b156b8c-1751-4382-a8dd-ba2779b087e0)
	dirPath     string
	rFile       *os.File
	wFile       *os.File
	fileInfo    *FileInfo
	sender      uint64
	receiver    uint64
	sendTime    int64
	window      int // size of stored file pieces // window & convergence use for synchronous
	convergence int // size of downed file pieces
	downld      bool
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

			// [TODO]{store to database} //

			logger.Tracef("Show task list:")
			for _, ft := range fm.fileTasks {
				logger.Tracef("Task list:{ TaskId:%v, FileInfo:%v, Window:%v, Convergence:%v, Downld:%v }",
					ft.taskId,
					ft.fileInfo,
					ft.window,
					ft.convergence,
					ft.downld)
			}

			logger.Debug("Add task done.")
		case dt := <-fm.delTask: // del file store task
			// [TODO] Actually, it should not delete too soon, for
			// re-download. Instead, set some deadline.
			if !dt.downld { // someone is downloading
				logger.Infof("Del file task:{%v}", dt.taskId)
				delete(fm.fileTasks, dt.taskId)
			}
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
					logger.Infof("File {%v} upload done.", ft.fileInfo.FileName)
				}
			} else {
				// send to client
				logger.Warn("You operate on a nil task.")
			}
			logger.Debug("Write pieces done.")
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
		dirPath:     dirPath,
		rFile:       nil,
		wFile:       f,
		fileInfo:    &fInfo,
		sender:      pack.Sender,
		receiver:    pack.Reciever,
		sendTime:    pack.TimeStamp,
		window:      0,
		convergence: 0,
		downld:      false,
	}, pack, nil
}

// Path condition
// 1. Unique (in order not to make conflict)
// 2. Connection with time (set deadline)
// 3. Easy to delete when in deadline
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

// Getting file.
func (fm *FileManager) downloadFile(tId string) { // [TODO:Naming]{getFile}
	var err error
	var ft *FileTask
	b := make([]byte, SeqLength)
	logger.Info("Begin downloading file.")

	ft = fm.fileTasks[tId]
	if ft == nil {
		var (
			effect   int
			dirPath  string
			sender   uint64
			receiver uint64
			fileName string
			fileSize int
		)

		fm.server.openDatabase("[Donwload File]")
		stmt, err := fm.server.db.Prepare("SELECT dirPath, sender, receiver, fileName, fileSize FROM file_list WHERE taskId=?")
		if err != nil {
			logger.Error(err.Error())
		}

		rows, err := stmt.Query(tId)
		if err != nil {
			logger.Error(err.Error())
		}

		for rows.Next() {
			effect++
			err := rows.Scan(&dirPath, &sender, &receiver, &fileName, &fileSize)
			if err != nil {
				logger.Error(err.Error())
			}
		}

		fInfo := &FileInfo{
			FileName: fileName,
			FileSize: fileSize,
		}
		if effect > 0 {
			ft = &FileTask{
				taskId:      tId,
				dirPath:     dirPath,
				rFile:       nil,
				wFile:       nil,
				fileInfo:    fInfo,
				sender:      sender,
				receiver:    receiver,
				window:      0,
				convergence: 0,
				downld:      true,
			}

			ft.rFile, err = os.Open(ft.dirPath + ft.fileInfo.FileName)
			if err != nil {
				logger.Error(err.Error())
			}
			fm.addTask <- ft

		} else {
			logger.Info("No such file task.")

			// response

			return
		}
		fm.server.closeDatabase("[Download file]")
		for i := 0; ft.window < ft.fileInfo.FileSize; i++ {
			n, err := ft.rFile.Read(b)
			if err != nil {
				logger.Error(err.Error())
				// do something
			}
			ft.window += n

			fs := &FileSeq{
				TaskId:     tId,
				SeqNum:     i,
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
		}
		ft.downld = false
		err = ft.rFile.Close() // window == fileSize
		if err != nil {
			logger.Error("Close write-file error:", err.Error())
		}
		fm.delTask <- ft
	} else {
		logger.Tracef("Show task:%v", ft)
		ft.rFile, err = os.Open(ft.dirPath)
		if err != nil {
			logger.Error(err.Error())
		}
		ft.downld = true

		for i := 0; ; i++ {
			if ft.convergence < ft.window {
				n, err := ft.rFile.Read(b)
				if err != nil {
					logger.Error(err.Error())
					// do something
				}
				logger.Tracef("Read %v bytes:{%v}", n, string(b[:n]))
				ft.convergence += n

				fs := &FileSeq{
					TaskId:     tId,
					SeqNum:     i,
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
			} else if ft.window < ft.fileInfo.FileSize {
				runtime.Gosched()
			} else {
				logger.Infof("Download {%v} complete.", ft.fileInfo.FileName)
				err := ft.rFile.Close() // window == fileSize
				if err != nil {
					logger.Error("Close write-file error:", err.Error())
				}
				fm.delTask <- ft
				return
			}
		}
	}
	logger.Info("Download end.")
}

//
func (fm *FileManager) StoreTask(ft *FileTask) {
	fm.server.openDatabase("[Store Task]")
	defer func() {
		fm.server.closeDatabase("[Store Task]")
	}()

	stmt, err := fm.server.db.Prepare("INSERT file_list SET sendTime=?, taskId=?, dirPath=?, sender=?, receiver=?, fileName=?, fileSize=?")
	if err != nil {
		logger.Error(err.Error())
		return
	}

	_, err = stmt.Exec(ft.sendTime, ft.taskId, ft.dirPath, ft.sender, ft.receiver, ft.fileInfo.FileName, ft.fileInfo.FileSize)
	if err != nil {
		logger.Error(err.Error())
	}
}

// Deadline of the file storing. This present as a file cleaner which user upload,
// it checks table 'file_list' in database every cleaning-cycle-time that been set.
func (fm *FileManager) Deadline() { // [TODO:Naming]{'FileCleaner'}
	logger.Info("Deadline counting.")

	for {
		var (
			taskId   string
			dirPath  string
			sendTime int
		)
		fm.server.openDatabase("[Deadline]")
		var line = time.Now().Unix() - int64(FileDeadline/time.Second)
		//logger.Tracef("Line : %v", line)

		stmt, err := fm.server.db.Prepare("SELECT taskId, dirPath, sendTime FROM file_list WHERE sendTime<?")
		if err != nil {
			logger.Error(err.Error())
		}

		rows, err := stmt.Query(line)
		if err != nil {
			logger.Error(err.Error())
		}

		for rows.Next() {
			err = rows.Scan(&taskId, &dirPath, &sendTime)
			if err != nil {
				logger.Error(err.Error())
			}
			err = os.RemoveAll(dirPath)
			if err != nil {
				logger.Error(err.Error())
			}

			logger.Debugf("Remove %v.", dirPath)

			stmt, err = fm.server.db.Prepare("DELETE FROM file_list WHERE taskId=?")
			if err != nil {
				logger.Error(err.Error())
			}

			_, err = stmt.Exec(taskId)
			if err != nil {
				logger.Error(err.Error())
			}
		}

		fm.server.closeDatabase("[Deadline]")
		time.Sleep(FileCleanCycle)
	}
}
