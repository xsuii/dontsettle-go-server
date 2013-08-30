package chat

import (
	"code.google.com/p/go.net/websocket"
	"database/sql"
	_ "github.com/Go-SQL-Driver/MySQL"
	"io"
	"log"
	"os"
	"strconv"
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
	send   chan string     // message channel
	doneCh chan bool
}

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
		send:   make(chan string),
		doneCh: make(chan bool)}
}

func (c *connection) Conn() *websocket.Conn { // get client's connection
	return c.ws
}

func (c *connection) Write(msg string) {
	select {
	case c.send <- msg:
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

func (c *connection) listenRead() { // send to all
	log.Println("read listen . . .")
	var uid int
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
				case "S": // one-to-one chat. // this will get "S + send to + massage body".
					db, err := sql.Open("mysql", "root:mrp520@/game")
					checkError(err)
					log.Println("open database")

					stmt, err := db.Prepare("SELECT uid FROM user WHERE username=?")
					checkError(err)

					rows, err := stmt.Query(talks.ToName)
					checkError(err)

					for rows.Next() {
						err = rows.Scan(&uid)
						checkError(err)
					}
					log.Println(uid)

					db.Close()
					log.Println("database close")

					var dId []string
					dId = append(dId, strconv.Itoa(uid))
					sId := c.uid
					m := "[S]:" + c.author + ": " + talks.Body
					s := &pack{dUid: dId, sUid: sId, msg: m, t: "S"}

					log.Println(s)
					c.server.transfer <- s
				case "G": // one-to-many chat.  // this will get "G + send to + massage body".
					db, err := sql.Open("mysql", "root:mrp520@/game")
					checkError(err)
					log.Println("database open")

					stmt, err := db.Prepare("SELECT uid FROM ingroup WHERE gid in(SELECT gid FROM game.group WHERE groupname=?)")

					rows, err := stmt.Query(talks.ToName)

					m := "[G:" + talks.ToName + "]" + c.author + ": " + talks.Body
					var mem []string
					for rows.Next() {
						var uid string
						err = rows.Scan(&uid)
						checkError(err)
						mem = append(mem, uid)
						log.Println(uid)
					}
					g := &pack{sUid: c.uid, dUid: mem, msg: m, t: "G"}

					log.Println(g)
					c.server.transfer <- g
				case "B":
					c.server.BroadCast("[B]:" + c.author + ": " + talks.Body)
				case "F": // should have a limit size
					log.Println("get file:", talks.Body)

					// store file in server side
					var data []byte
					websocket.Message.Receive(c.ws, &data)
					f, err := os.Create("./repertory/" + talks.Body) // file name
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

					// forwording file to target user
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
			websocket.Message.Send(c.ws, message)

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
