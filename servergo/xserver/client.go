package xserver

import (
	"code.google.com/p/go.net/websocket"
	"database/sql"
	"encoding/json"
	_ "github.com/Go-SQL-Driver/MySQL"
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
	Body     string
}

type connection struct {
	uid    uint // connection id
	Sender uint
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
		logger.Trace(login)
		logger.Trace("Receive login message : [ Username:", login.Username, " ]  [ Password:", login.Userpasswd, " ]")
		id, err := server.checkLogin(login.Username, login.Userpasswd)
		if err != nil {
			logger.Error(err.Error())
			return nil
		}
		b := make([]byte, 0)
		b = strconv.AppendUint(b, id, 10)
		p := &Pack{
			Sender:      0,
			Receiver:    0,
			Body:        b,
			DateTime:    time.Now().String(),
			OpCode:      OpLogin,
			ForwardType: "",
		}
		websocket.JSON.Send(ws, p)
		if id > 0 {
			return &connection{
				Sender: login.Username,
				uid:    sId,
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
				logger.Error(err.Error())
				return
			} else {
				// this will deal one-to-one、one-to-many、broadcast、file
				c.server.showPack("server", "recieve", pack)
				logger.Debug("check pack's validable")
				if !c.server.validPack(pack) {
					logger.Warn("Bad package!")
					c.server.masterPack(c, "Back package!")
					break
				}
				dst := c.GetUids(pack.ForwardType, pack.Receiver)
				if pack.OpCode == OpFileTransfer {
					if c.uid == dst[0] {
						// 如果收件人和发件人相同，意味着发件人使用邮件领取单领取邮件（这里是下载已上传的文件）
						// [+] if file upload not done yet
						logger.Trace(c.Sender, "download file:", pack.Body)
						sUid := c.GetUids(pack.ForwardType, pack.Sender)
						path = sUid[0] + "/" + c.uid // get download file's path
						err := c.DownloadFile(path, pack)
						if err != nil {
							logger.Error(err.Error())
						}
						break
					} else {
						path = c.uid + "/" + dst[0]
						logger.Trace(c.Sender, "upload file:", pack.Body)
						err := c.StoreFile(path, pack.Body)
						if err != nil {
							logger.Error(err)
						}
					}
				}
				logger.Debug("Pack package!")
				m := &Pack{
					Sender:      c.Sender,
					Receiver:    pack.Receiver,
					Body:        pack.Body,
					DateTime:    pack.DateTime,
					OpCode:      pack.OpCode,
					ForwardType: pack.ForwardType}
				p := &Postman{
					sUid: c.uid,
					dUid: dst,
					pack: m}
				logger.Trace(p)
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
		suid  string
		msg   string
		time  string
		t     int
		dt    string
		count int
	)
	logger.Debug("offline message push")
	c.server.openDatabase("offlinepusher")
	defer func() {
		c.server.closeDatabase("offlinepusher")
		logger.Trace("offline message push to %s over", c.uid)
	}()

	stmt, err := c.server.db.Prepare("SELECT sUID, time, message, packtype, dsttype FROM offlinemessage WHERE dUID=?")
	if err != nil {
		logger.Error(err.Error())
	}

	rows, err := stmt.Query(c.uid)
	if err != nil {
		logger.Error(err.Error())
	}

	for rows.Next() {
		count++
		err = rows.Scan(&suid, &time, &msg, &t, &dt)
		if err != nil {
			logger.Error(err.Error())
		}
		at := c.GetName(suid)
		ad := c.GetName(c.uid)
		m := &Pack{
			Sender:      at,
			Receiver:    ad,
			Body:        msg,
			DateTime:    time,
			OpCode:      t,
			ForwardType: dt,
		}
		p := &Postman{
			sUid: suid,
			dUid: []string{c.uid},
			pack: m,
		}
		c.server.Post(p)
	}
	logger.Tracef("Push %v message.", count)
	c.server.openDatabase("delete")
	stmt, err = c.server.db.Prepare("DELETE FROM offlinemessage WHERE dUID=?")
	if err != nil {
		logger.Error(err.Error())
	}

	_, err = stmt.Exec(c.uid)
	if err != nil {
		logger.Error(err.Error())
	}
}

func (c *connection) GetName(id string) string {
	var name string
	logger.Debug("get name with uid")
	c.server.openDatabase("GetName()")
	defer func() {
		c.server.closeDatabase("GetName()")
	}()

	stmt, err := c.server.db.Prepare("SELECT username FROM user WHERE uid=?")
	if err != nil {
		logger.Error(err.Error())
	}

	rows, err := stmt.Query(id)
	if err != nil {
		logger.Error(err.Error())
	}

	for rows.Next() {
		err = rows.Scan(&name)
		if err != nil {
			logger.Error(err.Error())
		}
	}
	logger.Trace("get:", name)
	return name
}

func (c *connection) GetUids(t string, whom uint) []string {
	logger.Trace("get", whom, "'s uid")
	var (
		uid  string
		dUid []string
		stmt *sql.Stmt
		rows *sql.Rows
		err  error
	)
	c.server.openDatabase("GetUids()")
	defer func() {
		c.server.closeDatabase("GetUids()")
	}()

	if t == "S" {
		stmt, err = c.server.db.Prepare("SELECT uid FROM user WHERE username=?")
		if err != nil {
			logger.Error("Error:", err.Error())
		}
		rows, err = stmt.Query(whom)
		if err != nil {
			logger.Error("Error:", err.Error())
		}
	} else if t == "G" {
		stmt, err = c.server.db.Prepare("SELECT uid FROM ingroup WHERE gid in(SELECT gid FROM game.group WHERE groupname=?)")
		if err != nil {
			logger.Error("Error:", err.Error())
		}
		rows, err = stmt.Query(whom)
		if err != nil {
			logger.Error("Error:", err.Error())
		}
	} else if t == "B" {
		rows, err = c.server.db.Query("SELECT uid FROM user")
		if err != nil {
			logger.Error("Error:", err.Error())
		}
	} else {
		logger.Warn("error destination type")
		return nil
	}

	for rows.Next() {
		err = rows.Scan(&uid)
		if err != nil {
			logger.Error("Error:", err.Error())
		}
		dUid = append(dUid, uid)
	}
	logger.Trace(dUid)

	return dUid
}

func (c *connection) StoreFile(path string, filename []byte) error {
	// store file in server side
	var data []byte
	logger.Trace("begin to store file:", path, filename)
	err := websocket.Message.Receive(c.ws, &data)
	if err != nil {
		return err
	}

	// file content
	if len(data) > 50 {
		logger.Debug("Receive file data :", string(data[:50]))
	} else {
		logger.Debug("Receive file data :", string(data))
	}

	err = os.MkdirAll("./repertory/"+path, 0777)
	if err != nil {
		return err
	}

	f, err := os.Create("./repertory/" + path + "/" + string(filename)) // file name. it should be deleted if exist or add datetime as filename
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
func (c *connection) DownloadFile(path string, pack Pack) error {
	logger.Trace("begin to download file:", path, pack.Body)

	f, err := os.Open("./repertory/" + path + "/" + pack.Body)
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
		logger.Trace(string(data[:50]))
	} else {
		logger.Trace(string(data))
	}

	fi := &File{
		FileName: pack.Body,
		Body:     string(data),
	}

	file, err := json.Marshal(fi)
	if err != nil {
		return err
	}

	p := &Pack{
		Sender:      "MASTER",
		Receiver:    c.Sender,
		Body:        string(file),
		DateTime:    pack.DateTime,
		OpCode:      pack.OpCode,
		ForwardType: pack.ForwardType,
	}
	websocket.JSON.Send(c.ws, p)
	return nil
}
