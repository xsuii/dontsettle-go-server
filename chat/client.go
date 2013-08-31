package chat

import (
	"code.google.com/p/go.net/websocket"
	"database/sql"
	_ "github.com/Go-SQL-Driver/MySQL"
	"io"
	"log"
	"os"
	"strings"
)

type Talks struct {
	Type   string
	ToName string
	Body   string
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
	log.Println(" - uid : ", temp[0], " - author : ", temp[1], " - ")

	return &connection{
		uid:    temp[0],
		author: temp[1],
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
	var uid string
	var dUid []string
	var stmt *sql.Stmt
	var err error
	c.server.openDatabase()

	if t == "S" {
		stmt, err = c.server.db.Prepare("SELECT uid FROM user WHERE username=?")
		checkError(err)
	} else if t == "G" {
		stmt, err = c.server.db.Prepare("SELECT uid FROM ingroup WHERE gid in(SELECT gid FROM game.group WHERE groupname=?)")
		checkError(err)
	}

	rows, err := stmt.Query(whom)
	checkError(err)

	for rows.Next() {
		err = rows.Scan(&uid)
		checkError(err)
		dUid = append(dUid, uid)
	}
	log.Println(dUid)

	c.server.closeDatabase()
	return dUid
}

func (c *connection) StoreFile(filename string) {
	// store file in server side
	var data []byte
	log.Println("begin to store file")
	websocket.Message.Receive(c.ws, &data)
	f, err := os.Create("./repertory/" + filename) // file name
	checkError(err)
	d := make([]byte, 4096)
	l := len(data)
	var p int
	if l < 4096 {
		d = data[0:]
		_, err := f.Write(d)
		checkError(err)
	} else {
		for p < l/4096 {
			d = data[p*4096 : (p+1)*4096]
			_, err := f.Write(d)
			checkError(err)
			p++
		}
		if l%4096 != 0 { // tail of file
			d = data[p*4096:]
			_, err := f.Write(d)
			checkError(err)
		}
	}
	log.Println("finish storing file")
}

func (c *connection) listenRead() { // send to all
	log.Println("read listen . . .")
	var talks Talks
	for {
		select {

		case <-c.doneCh:
			c.server.Unregister(c)
			c.Done()
			log.Println("done from listen read")

		default:
			err := websocket.JSON.Receive(c.ws, &talks)
			if err == io.EOF {
				c.Done()
				log.Println("default : done from listen read")
			} else if err != nil {
				c.server.Err(err)
			} else {
				log.Println(talks)
				switch talks.Type {
				case "S":
					// one-to-one chat.
					// this will get "S + send to + massage body".
					dst := c.GetUids("S", talks.ToName)
					//m := "[S]:" + c.author + ": " + talks.Body
					m := Pack{
						Author:    c.author,
						Addressee: talks.ToName,
						Message:   talks.Body,
						DateTime:  "",
						Type:      "MSG",
						DstT:      "S"}
					p := &Postman{
						sUid: c.uid,
						dUid: dst,
						pack: m,
						t:    "S"}

					log.Println(p)
					c.server.postman <- p
				case "G": // one-to-many chat.  // this will get "G + send to + massage body".
					dst := c.GetUids("G", talks.ToName)
					m := Pack{
						Author:    c.author,
						Addressee: talks.ToName,
						Message:   talks.Body,
						DateTime:  "",
						Type:      "MSG",
						DstT:      "G"}
					p := &Postman{
						sUid: c.uid,
						dUid: dst,
						pack: m,
						t:    "G"}
					log.Println(p)

					c.server.postman <- p
				case "B":
					m := Pack{
						Author:    c.author,
						Addressee: talks.ToName,
						Message:   talks.Body,
						DateTime:  "",
						Type:      "MSG",
						DstT:      "B"}
					c.server.BroadCast(m)
				case "F":
					// recieve file the client upload
					// [later] should have a limit size(client side)
					// talks struct( "body" field : file name , "ToName" field : to whom )
					log.Println("get file:", talks.Body)

					c.StoreFile(talks.Body)

					// forwording file to target user
					d := c.GetUids("S", talks.ToName)
					m := Pack{
						Author:    c.author,
						Addressee: talks.ToName,
						Message:   talks.Body,
						DateTime:  "",
						Type:      "FILE",
						DstT:      "S"}
					p := &Postman{
						sUid: c.uid,
						dUid: d,
						pack: m,
						t:    "S"}
					c.server.postman <- p
				}
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

func checkError(err error) {
	if err != nil {
		log.Println(err.Error())
	}
}
