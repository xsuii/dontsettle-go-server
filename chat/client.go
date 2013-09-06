package chat

import (
	"code.google.com/p/go.net/websocket"
	"database/sql"
	"encoding/json"
	_ "github.com/Go-SQL-Driver/MySQL"
	"io"
	"log"
	"os"
	"strings"
)

type File struct {
	FileName string
	Body     string
}

type connection struct {
	uid    string // connection id
	author string
	ws     *websocket.Conn // connection socket
	server *Server         // the server was connected
	send   chan Pack       // message channel
	doneCh chan bool
}

// [later:JSON]
func NewClient(ws *websocket.Conn, server *Server) *connection {
	log.Println("new client . . .")
	var msg string
	websocket.Message.Receive(ws, &msg) // get uid & author
	temp := strings.Split(msg, "+")
	log.Println(" - author : ", temp[0], " - uid : ", temp[1], " - ")

	return &connection{
		author: temp[0],
		uid:    temp[1],
		ws:     ws,
		server: server,
		send:   make(chan Pack),
		doneCh: make(chan bool)}
}

func (c *connection) Conn() *websocket.Conn { // get client's connection
	return c.ws
}

func (c *connection) Write(pack Pack) {
	select {
	case c.send <- pack:
	default:
		c.server.Unregister(c)
		log.Println("client %s is disconnected.", c.uid)
	}
}

func (c *connection) Done() {
	c.doneCh <- true
}

func (c *connection) Listen() {
	log.Println("client listening")
	go c.listenWrite()
	c.listenRead()
}

func (c *connection) GetUids(t string, whom string) []string {
	log.Println("get", whom, "'s uid")
	var uid string
	var dUid []string
	var stmt *sql.Stmt
	var rows *sql.Rows
	var err error
	c.server.openDatabase()

	if t == "S" {
		stmt, err = c.server.db.Prepare("SELECT uid FROM user WHERE username=?")
		if err != nil {
			log.Println("Error:", err.Error())
		}
		rows, err = stmt.Query(whom)
		if err != nil {
			log.Println("Error:", err.Error())
		}
	} else if t == "G" {
		stmt, err = c.server.db.Prepare("SELECT uid FROM ingroup WHERE gid in(SELECT gid FROM game.group WHERE groupname=?)")
		if err != nil {
			log.Println("Error:", err.Error())
		}
		rows, err = stmt.Query(whom)
		if err != nil {
			log.Println("Error:", err.Error())
		}
	} else if t == "B" {
		rows, err = c.server.db.Query("SELECT uid FROM user")
		if err != nil {
			log.Println("Error:", err.Error())
		}
	} else {
		log.Println("error destination type")
		return nil
	}

	for rows.Next() {
		err = rows.Scan(&uid)
		if err != nil {
			log.Println("Error:", err.Error())
		}
		dUid = append(dUid, uid)
	}
	log.Println(dUid)

	c.server.closeDatabase()
	return dUid
}

func (c *connection) StoreFile(path string, filename string) {
	// store file in server side
	var data []byte
	log.Println("begin to store file:", path, filename)
	err := websocket.Message.Receive(c.ws, &data)
	if err != nil {
		log.Println("Error:", err.Error())
	}

	if len(data) > 50 {
		log.Println("DEBUG : Receive file data :", string(data[:50]))
	} else {
		log.Println("DEBUG : Receive file data :", string(data))
	}

	err = os.MkdirAll("./repertory/"+path, 0777)
	if err != nil {
		log.Println("Error:", err.Error())
	}

	f, err := os.Create("./repertory/" + path + "/" + filename) // file name. it should be deleted if exist or add datetime as filename
	if err != nil {
		log.Println("Error:", err.Error())
	}
	defer func() {
		err := f.Close()
		if err != nil {
			log.Println("Error:", err.Error())
		}
	}()

	d := make([]byte, 4096)
	l := len(data)
	var p int
	if l < 4096 {
		d = data[0:]
		_, err := f.Write(d)
		if err != nil {
			log.Println("Error:", err.Error())
		}
	} else {
		for p < l/4096 {
			d = data[p*4096 : (p+1)*4096]
			_, err := f.Write(d)
			if err != nil {
				log.Println("Error:", err.Error())
			}
			p++
		}
		if l%4096 != 0 { // tail of file
			d = data[p*4096:]
			_, err := f.Write(d)
			if err != nil {
				log.Println("Error:", err.Error())
			}
		}
	}
	log.Println("finish storing file")
}

// this should work by pieces.
func (c *connection) DownloadFile(path string, pack Pack) {
	log.Println("begin to download file:", path, pack.Message)

	f, err := os.Open("./repertory/" + path + "/" + pack.Message)
	if err != nil {
		log.Println("Error:", err.Error())
	}
	defer func() {
		err := f.Close()
		if err != nil {
			log.Println("Error:", err.Error())
		}
		log.Println("download file done")
	}()

	buf := make([]byte, 1024)
	var data []byte
	for {
		n, err := f.Read(buf)
		if err != nil && err != io.EOF {
			if err != nil {
				log.Println("Error:", err.Error())
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
		log.Println(string(data[:50]))
	} else {
		log.Println(string(data))
	}

	fi := &File{
		FileName: pack.Message,
		Body:     string(data),
	}

	file, err := json.Marshal(fi)
	if err != nil {
		log.Println("Error:", err.Error())
	}

	p := &Pack{
		Author:    "MASTER",
		Addressee: c.author,
		Message:   string(file),
		DateTime:  pack.DateTime,
		Type:      pack.Type,
		DstT:      pack.DstT,
	}
	websocket.JSON.Send(c.ws, p)
}

func (c *connection) listenRead() { // send to all
	log.Println("read listen . . .")
	var pack Pack
	var path string
	for {
		select {

		case <-c.doneCh:
			c.server.Unregister(c)
			c.Done()
			log.Println("done from listen read")
		default:
			err := websocket.JSON.Receive(c.ws, &pack)
			log.Println(pack)
			if err == io.EOF {
				c.Done()
				log.Println("default : done from listen read")
			} else if err != nil {
				c.server.Err(err)
			} else {
				// this will deal one-to-one、one-to-many、broadcast、file
				log.Println(pack)
				dst := c.GetUids(pack.DstT, pack.Addressee)
				if pack.Type == "FILE" {
					if c.uid == dst[0] {
						// 如果收件人和发件人相同，意味着发件人使用邮件领取单领取邮件（这里是下载已上传的文件）
						// [+] if file upload not done yet
						log.Println(c.author, "download file:", pack.Message)
						sUid := c.GetUids(pack.DstT, pack.Author)
						path = sUid[0] + "/" + c.uid // get download file's path
						c.DownloadFile(path, pack)
						break
					}
					path = c.uid + "/" + dst[0]
					log.Println(c.author, "upload file:", pack.Message)
					c.StoreFile(path, pack.Message)
				}
				m := Pack{
					Author:    c.author,
					Addressee: pack.Addressee,
					Message:   pack.Message,
					DateTime:  pack.DateTime,
					Type:      pack.Type,
					DstT:      pack.DstT}
				p := &Postman{
					sUid: c.uid,
					dUid: dst,
					pack: m}
				log.Println(p)
				c.server.Post(p)
			}
		}
	}
}

func (c *connection) listenWrite() {
	log.Println("write listen . . .")
	for {
		select {
		case message := <-c.send:
			log.Println(c.author, "send : ", message)
			websocket.JSON.Send(c.ws, message)

		case <-c.doneCh:
			c.server.Unregister(c)
			c.Done()
			log.Println("done from listen write")
			return
		}
	}
}
