package chat

import (
	"code.google.com/p/go.net/websocket"
	"database/sql"
	_ "github.com/Go-SQL-Driver/MySQL"
	"io"
	"log"
	"strconv"
	"strings"
)

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
	for {
		select {

		case <-c.doneCh:
			c.server.Unregister(c)
			c.Done()
			log.Println("done from listen read")

		default:
			var msg string
			err := websocket.Message.Receive(c.ws, &msg)
			if err == io.EOF {
				c.Done()
				log.Println("default : done from listen read")
			} else if err != nil {
				c.server.Err(err)
			} else {
				temp := strings.Split(msg, "+") // later : use JSON identify with head&body
				switch temp[0] {
				case "S": // one-to-one chat. // this will get "S + send to + massage body".
					db, err := sql.Open("mysql", "root:mrp520@/game")
					checkError(err)
					log.Println("open database")

					stmt, err := db.Prepare("SELECT uid FROM user WHERE username=?")
					checkError(err)

					rows, err := stmt.Query(temp[1])
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
					m := "[S]:" + c.author + ": " + temp[2]
					s := &pack{dUid: dId, sUid: sId, msg: m, t: "S"}

					log.Println(s)
					c.server.transfer <- s
				case "G": // one-to-many chat.  // this will get "G + send to + massage body".
					db, err := sql.Open("mysql", "root:mrp520@/game")
					checkError(err)
					log.Println("database open")

					stmt, err := db.Prepare("SELECT uid FROM ingroup WHERE gid in(SELECT gid FROM game.group WHERE groupname=?)")

					rows, err := stmt.Query(temp[1])

					m := "[G:" + temp[1] + "]" + c.author + ": " + temp[2]
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
					c.server.BroadCast("[B]:" + c.author + ": " + temp[1])
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
