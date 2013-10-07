package xserver

import (
	"code.google.com/p/go.net/websocket"
	"database/sql"
	"encoding/json"
	_ "github.com/go-sql-driver/mysql"
	"io"
	"os"
	"strconv"
	"time"
)

type LoginInfo struct {
	Username   string
	Userpasswd string
}

type File struct {
	FileName string
	Body     []byte
}

type Ticket struct {
	FSender   uint64
	FReceiver uint64
	FileName  string
	TimeStamp int64
}

type connection struct {
	uid    uint64          // connection id
	ws     *websocket.Conn // connection socket
	server *Server         // the server was connected
	send   chan *Pack      // message channel
	doneCh chan bool
}

// [later:JSON]
func NewClient(ws *websocket.Conn, server *Server) *connection {
	var login LoginInfo

	logger.Info(" # New Connection # ")
	logger.Info("Client :", ws.Request().RemoteAddr)
	for {
		err := websocket.JSON.Receive(ws, &login) // get uid & Sender
		if err == io.EOF {
			logger.Error("Login:recieve EOF")
			return nil
		} else if err != nil {
			logger.Error("Login receive error :", err.Error())
			return nil
		}
		logger.Tracef("Receive login message : [ Username: %v ]  [ Password: %v ]", login.Username, login.Userpasswd)
		id, err := server.checkLogin(login.Username, login.Userpasswd)
		if err != nil {
			logger.Error(err.Error())
			return nil
		}
		bid := make([]byte, 0) // convert uint64 to []byte
		bid = strconv.AppendUint(bid, id, 10)
		p := server.pack(MasterId, id, bid, time.Now().Unix(), OpLogin, FwSingle)
		websocket.JSON.Send(ws, p)
		if id > 0 {
			return &connection{
				uid:    id,
				ws:     ws,
				server: server,
				send:   make(chan *Pack),
				doneCh: make(chan bool)}
		}
	}
}

func (c *connection) listenRead() { // send to all
	logger.Debug("listening read")
	var path string
	for {
		var pack Pack
		logger.Debug("pack:", pack)
		select {
		case <-c.doneCh:
			c.server.Unregister(c)
			c.Done()
			logger.Debug("done from listen read")
		default:
			/*var test string
			err := websocket.Message.Receive(c.ws, &test)
			log.Println(test)*/
			err := websocket.JSON.Receive(c.ws, &pack)
			if err == io.EOF {
				c.Done()
				logger.Info("default : done from listen read")
			} else if err != nil {
				logger.Error("Package receive error :", err.Error())
				break
			} else {
				// this will deal one-to-one、one-to-many、broadcast、file
				c.server.showPack("server", "recieve", pack)
				logger.Debug("check pack's validable")
				if !c.server.validPack(pack) {
					logger.Warn("Bad package!")
					c.server.masterPack(c, []byte("Back package!"))
					break
				}
				err, recv := c.GetUids(pack.ForwardType, pack.Receiver)
				if err != nil {
					logger.Error(err.Error())
					return
				}
				switch pack.OpCode {
				case OpFileUp: // file transfer surport only 1:1 now
					logger.Trace(c.uid, " upload file:", pack.Body)
					path = getFilePath(pack.Sender, pack.Receiver, pack.TimeStamp)
					err := c.StoreFile(path, string(pack.Body))
					if err != nil {
						logger.Error(err)
					}
				case OpFileDown:
					var tk Ticket
					logger.Trace(c.uid, " download file:", string(pack.Body))
					err = json.Unmarshal(pack.Body, &tk)
					if err != nil {
						logger.Error("Unmarshal error :", err.Error())
					}
					logger.Tracef("Ticket : %v", tk)
					path = getFilePath(tk.FSender, tk.FReceiver, tk.TimeStamp) // get download file's path]
					err := c.DownloadFile(path, tk)
					if err != nil {
						logger.Error(err.Error())
					}
					continue
				}
				p := &Postman{
					sUid:  c.uid,
					dUids: recv,
					pack:  &pack}
				c.server.Post(p)
			}
		}
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

func (c *connection) OfflinePush() {
	var (
		sender    uint64
		body      []byte
		timestamp int64
		opcode    byte
		fwt       byte
		count     int
	)
	logger.Debug("offline message push")
	c.server.openDatabase("OfflinePush()")
	defer func() {
		c.server.closeDatabase("OfflinePush()")
		logger.Tracef("offline message push to %v(uid) over", c.uid)
	}()

	stmt, err := c.server.db.Prepare("SELECT offMsgSender, offMsgTimeStamp, offMsgBody, offMsgOpCode, offMsgForwardType FROM offline_message WHERE offMsgReceiver=?")
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
		err = rows.Scan(&sender, &timestamp, &body, &opcode, &fwt)
		if err != nil {
			logger.Error(err.Error())
			return
		}
		m := c.server.pack(sender, c.uid, body, timestamp, opcode, fwt)
		p := &Postman{
			sUid:  sender,
			dUids: []uint64{c.uid},
			pack:  &m,
		}
		c.server.Post(p)
	}
	logger.Tracef("Push %v message.", count)
	c.server.openDatabase("Clear OffMsg")
	stmt, err = c.server.db.Prepare("DELETE FROM offline_message WHERE offMsgReceiver=?")
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

/*
func (c *connection) GetName(id string) (error, string) {
	var name string
	logger.Debug("get name with uid")
	c.server.openDatabase("GetName()")
	defer func() {
		c.server.closeDatabase("GetName()")
	}()

	stmt, err := c.server.db.Prepare("SELECT username FROM user WHERE uid=?")
	if err != nil {
		return err, ""
	}

	rows, err := stmt.Query(id)
	if err != nil {
		return err, ""
	}

	for rows.Next() {
		err = rows.Scan(&name)
		if err != nil {
			return err, ""
		}
	}
	logger.Trace("get:", name)
	return nil, name
}*/

func (c *connection) GetUids(fwt byte, rid uint64) (error, []uint64) {
	var (
		uid   uint64
		dUids []uint64
		stmt  *sql.Stmt
		rows  *sql.Rows
		err   error
	)
	c.server.openDatabase("GetUids()")
	defer func() {
		c.server.closeDatabase("GetUids()")
	}()

	switch fwt {
	case FwSingle:
		logger.Trace("Get ", rid, "'s uid")
		stmt, err = c.server.db.Prepare("SELECT userId FROM user WHERE userId=?")
		if err != nil {
			return err, nil
		}
		rows, err = stmt.Query(rid)
		if err != nil {
			return err, nil
		}
	case FwGroup:
		logger.Trace("Get group ", rid, "'s uid")
		stmt, err = c.server.db.Prepare("SELECT userId FROM in_circle WHERE circleId in(SELECT circleId FROM game.circle WHERE circleId=?)")
		if err != nil {
			return err, nil
		}
		rows, err = stmt.Query(rid)
		if err != nil {
			return err, nil
		}
	case FwBroadcast:
		logger.Trace("Get all uid")
		rows, err = c.server.db.Query("SELECT userId FROM user")
		if err != nil {
			return err, nil
		}
	default:
		logger.Warn("error destination type")
		return nil, nil
	}

	for rows.Next() {
		err = rows.Scan(&uid)
		if err != nil {
			return err, nil
		}
		dUids = append(dUids, uid)
	}
	logger.Trace("Get forwarding ids:", dUids)

	return nil, dUids
}

func getFilePath(id1 uint64, id2 uint64, ts int64) string {
	return idToString(id1) + "/" + idToString(id2) + "/" + int64ToString(ts)
}

func (c *connection) StoreFile(path string, filename string) error {
	// store file in server side
	var data []byte
	logger.Tracef("begin to store file: %v/%v", path, filename)
	err := websocket.Message.Receive(c.ws, &data)
	if err != nil {
		return err
	}

	// file content
	if len(data) > 50 {
		logger.Debug("Receive file data :", data[:50])
	} else {
		logger.Debug("Receive file data :", data)
	}

	err = os.MkdirAll("./repertory/"+path, 0777)
	if err != nil {
		return err
	}

	f, err := os.Create("./repertory/" + path + "/" + filename) // file name. it should be deleted if exist or add TimeStamp as filename
	if err != nil {
		return err
	}
	defer func() {
		err := f.Close()
		if err != nil {
			logger.Error(err.Error())
		}
		logger.Info("finish storing file")
	}()

	d := make([]byte, 4096)
	l := len(data)
	var p int
	if l < 4096 {
		d = data[0:]
		_, err := f.Write(d)
		if err != nil {
			return err
		}
	} else {
		for p < l/4096 {
			d = data[p*4096 : (p+1)*4096]
			_, err := f.Write(d)
			if err != nil {
				return err
			}
			p++
		}
		if l%4096 != 0 { // tail of file
			d = data[p*4096:]
			_, err := f.Write(d)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// this should work by pieces.
func (c *connection) DownloadFile(path string, tk Ticket) error {
	logger.Trace("begin to download file:", path, tk.FileName)

	f, err := os.Open("./repertory/" + path + "/" + string(tk.FileName)) // get file
	if err != nil {
		return err
	}
	defer func() {
		err := f.Close()
		if err != nil {
			logger.Error(err.Error())
		}
		logger.Info("download file done")
	}()

	buf := make([]byte, 1024)
	var data []byte
	for {
		n, err := f.Read(buf)
		if err != nil && err != io.EOF {
			if err != nil {
				return err
			}
		}
		if n != 0 {
			if n < 1024 {
				data = append(data, buf[0:n]...)
			} else {
				data = append(data, buf...)
			}
		} else {
			break
		}
	}

	// observe data encode
	if len(data) > 50 {
		logger.Trace(data[:50])
	} else {
		logger.Trace(data)
	}

	fi := &File{
		FileName: tk.FileName,
		Body:     data,
	}

	file, err := json.Marshal(fi)
	if err != nil {
		return err
	}

	p := c.server.pack(MasterId, c.uid, file, tk.TimeStamp, OpFileDown, FwSingle)
	//websocket.JSON.Send(c.ws, p)
	pm := NewPostman(c.uid, []uint64{p.Receiver}, &p)
	c.server.Post(&pm)
	return nil
}
