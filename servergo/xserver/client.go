package xserver

import (
	"code.google.com/p/go.net/websocket"
	"encoding/json"
	_ "github.com/go-sql-driver/mysql"
	"io"
	"os"
	"runtime"
	"strconv"
	"time"
)

var _ = os.Mkdir
var _ = runtime.GOOS

type LoginInfo struct {
	Username   string
	Userpasswd string
}

type File struct {
	FileName string
	Body     []byte
}

type FileSequence struct {
	FileName   string
	FileSize   int
	FileSeq    int
	SeqContent string
	SeqSize    int
}

type Ticket struct {
	FSender   uint64
	FReciever uint64
	FileName  string
	TimeStamp int64
}

type connection struct {
	uid    uint64          // connection id
	ws     *websocket.Conn // connection socket
	server *Server         // the server was connected
	send   chan *Pack      // message channel
	doneCh chan bool
	syncCh chan bool
}

func NewClient(ws *websocket.Conn, server *Server) *connection {
	var lgif LoginInfo

	logger.Infof(" # New Connection : %v # ", ws.Request().RemoteAddr)
	for {
		err := websocket.JSON.Receive(ws, &lgif) // get uid & Sender
		if err == io.EOF {
			logger.Error("Login:recieve EOF")
			return nil
		} else if err != nil {
			logger.Error("Login receive error :", err.Error())
			return nil
		}
		logger.Tracef("Receive login message : { Username:%v, Password:%v }", lgif.Username, lgif.Userpasswd)

		id, err := server.checkLogin(lgif.Username, lgif.Userpasswd)
		if err != nil {
			logger.Error(err.Error())
			return nil
		}

		bid := make([]byte, 0) // convert uint64 to []byte
		bid = strconv.AppendUint(bid, id, 10)
		p := server.NewPack(MasterId, id, time.Now().Unix(), OpLogin, bid)
		websocket.JSON.Send(ws, p) // [fix:it shouldn't be]
		if id > 0 {
			return &connection{
				uid:    id,
				ws:     ws,
				server: server,
				send:   make(chan *Pack),
				doneCh: make(chan bool),
				syncCh: make(chan bool)}
		}
	}
}

func (c *connection) listenRead() { // send to all
	logger.Debug("listening read")
	for {
		//time.Sleep(time.Second)
		var pack Pack
		select {
		case <-c.doneCh:
			c.server.Unregister(c)
			c.Done()
			logger.Debug("done from listen read")
		default:
			err := websocket.JSON.Receive(c.ws, &pack)
			if err == io.EOF {
				c.Done()
				logger.Info("default : done from listen read")
				break
			} else if err != nil {
				logger.Error("Package receive error :", err.Error())
				break
			}
			c.server.showPack("server", "recieve", pack)

			if err := c.server.checkPackValid(pack); err != nil {
				logger.Error(err.Error())
				body := c.server.errorWrapper(ErrBadPackage, "You send the invalid package.")
				c.ResponseB(OpError, body)
				break
			} else {
				logger.Info("Package is valid.")
			}

			switch pack.OpCode {
			case OpChatToOne: // chat
				c.server.toOne <- &pack
			case OpChatToMuti:
				fallthrough
			case OpChatBroadcast:
				c.server.toMuti <- &pack
			case OpFileUpldReq:
				logger.Info("Server receive file upload request.")
				ft, fTk, err := c.server.fileMan.NewFileTaskAndFileTicket(&pack)
				if err != nil {
					logger.Errorf("Make new file task error:%v", err.Error())
					body := c.server.errorWrapper(ErrFileUpReqAck, "You request is invail because of the server side.")
					c.ResponseB(OpError, body)
					break
				}
				c.showFileTask(ft)
				c.server.fileMan.addTask <- ft
				c.server.toOne <- fTk
				c.ResponseS(OpFileUpldReqAckOk, ft.taskId)
			case OpFileDownldReq:
				logger.Info("Recieve download request.")
				// If file not out date, send ACK to client side and start download.
				//go c.downloadFile(&pack)
				c.ResponseB(OpFileDownldReqAckOk, pack.Body)
			case OpFileUpld: // file transfer surport only 1:1 now
				var fs FileSeq
				err := json.Unmarshal(pack.Body, &fs)
				if err != nil {
					logger.Errorf("[File up]:%v", err.Error())
				}
				c.server.fileMan.fileUpLd <- &fs
			case OpFileDownld:
				logger.Info("Start download.")
				c.server.fileMan.fileDownLd <- string(pack.Body) // start download
			default:
				// no such operation, and send back an error message.
				body := c.server.errorWrapper(ErrOperation, "You request the error operation")
				c.ResponseB(OpError, body)
			}
		}
		//time.Sleep(time.Second)
	}
}

func (c *connection) listenWrite() {
	logger.Debug("listening write")
	for {
		select {
		case message := <-c.send:
			c.server.showPack("server", "send", *message)
			websocket.JSON.Send(c.ws, message)
		case <-c.doneCh:
			c.server.Unregister(c)
			c.Done()
			logger.Info("done from listen write")
			return
		}
	}
}

func (c *connection) Listen() {
	logger.Debug("client listening")
	go c.listenWrite()

	//{ user data push or update }//

	c.OfflinePush()
	c.listenRead()
}

// Response to client whom request
func (c *connection) ResponseS(opcode int, body string) {
	rp := &Pack{
		Sender:    MasterId,
		Reciever:  c.uid,
		TimeStamp: time.Now().Unix(),
		OpCode:    byte(opcode),
		Body:      []byte(body),
	}
	c.server.toOne <- rp
}

func (c *connection) ResponseB(opcode int, body []byte) {
	rp := &Pack{
		Sender:    MasterId,
		Reciever:  c.uid,
		TimeStamp: time.Now().Unix(),
		OpCode:    byte(opcode),
		Body:      body,
	}
	c.server.toOne <- rp
}

func (c *connection) showFileTask(ft *FileTask) {
	logger.Tracef("Show new file task : { TaskId:%v, wFile:%v, rFile:%v, FileName:%v, FileSize:%v, Window:%v, Convergence:%v }",
		ft.taskId,
		ft.wFile,
		ft.rFile,
		ft.fileInfo.FileName,
		ft.fileInfo.FileSize,
		ft.window,
		ft.convergence)
}

func (c *connection) Conn() *websocket.Conn { // get client's connection
	return c.ws
}

func (c *connection) Write(pack *Pack) {
	select {
	case c.send <- pack:
	default:
		c.server.Unregister(c)
		logger.Debug("client %s is disconnected.", c.uid)
	}
}

func (c *connection) Done() {
	c.doneCh <- true
}

// do sync job assciate with Sync()
func (c *connection) DoneSync() {
	c.syncCh <- true
}

// do sync job assciate with DoneSync()
func (c *connection) Sync() {
	<-c.syncCh
}

// [TODO] Just push the latest n message at the login, push more when the client ask.
func (c *connection) OfflinePush() {
	logger.Debug("Start offline message push.")
	var (
		sender    uint64
		body      []byte
		timestamp int64
		opcode    byte
		count     int
	)
	c.server.openDatabase("[Fund:OfflinePush]")
	defer func() {
		c.server.closeDatabase("[Fund:OfflinePush]")
		logger.Tracef("offline message push to %v(uid) over", c.uid)
	}()

	stmt, err := c.server.db.Prepare("SELECT offMsgSender, offMsgTimeStamp, offMsgBody, offMsgOpCode FROM offline_message WHERE offMsgReciever=?")
	if err != nil {
		logger.Error(err.Error())
		return
	}

	rows, err := stmt.Query(c.uid)
	if err != nil {
		logger.Error(err.Error())
		return
	}

	for rows.Next() {
		count++
		err = rows.Scan(&sender, &timestamp, &body, &opcode)
		if err != nil {
			logger.Error(err.Error())
			return
		}
		p := c.server.NewPack(sender, c.uid, timestamp, opcode, body)
		c.server.toOne <- p
	}
	logger.Tracef("Push total %v message.", count)
	c.server.openDatabase("[OP:Clear OffMsg]")
	stmt, err = c.server.db.Prepare("DELETE FROM offline_message WHERE offMsgReciever=?")
	if err != nil {
		logger.Error(err.Error())
		return
	}

	_, err = stmt.Exec(c.uid)
	if err != nil {
		logger.Error(err.Error())
		return
	}
}
