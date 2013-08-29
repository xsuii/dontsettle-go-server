package chat

import (
	"code.google.com/p/go.net/websocket"
	"database/sql"
	_ "github.com/Go-SQL-Driver/MySQL"
	"log"
	"net/http"
)

// for one-to-one chat  [later:this can merge with group struct]
type pack struct {
	sUid string
	dUid []string
	msg  string
	t    string
}

type Server struct {
	pattern     string
	history     []string
	connections map[string]*connection // Registered connections
	register    chan *connection
	unregister  chan *connection
	broadcast   chan string
	transfer    chan *pack
	errCh       chan error
	doneCh      chan bool
	db          *sql.DB
}

func NewServer(pattern string) *Server {
	history := []string{}
	connections := make(map[string]*connection)
	register := make(chan *connection)
	unregister := make(chan *connection)
	broadcast := make(chan string)
	transfer := make(chan *pack)
	errCh := make(chan error)
	doneCh := make(chan bool)
	db := &sql.DB{}

	return &Server{
		pattern,
		history,
		connections,
		register,
		unregister,
		broadcast,
		transfer,
		errCh,
		doneCh,
		db,
	}
}

// add client
func (s *Server) Register(c *connection) {
	s.register <- c
}

// delete client
func (s *Server) Unregister(c *connection) {
	s.unregister <- c
}

func (s *Server) BroadCast(msg string) {
	s.broadcast <- msg
}

func (s *Server) Transfer(b *pack) {
	s.transfer <- b
}

func (s *Server) Done() {
	s.doneCh <- true
}

func (s *Server) Err(err error) {
	s.errCh <- err
}

func (s *Server) checkError(err error) {
	if err != nil {
		log.Println(err.Error())
	}
}

func (s *Server) sendPastMessages(c *connection) {
	for _, msg := range s.history {
		c.Write(msg)
	}
}

func (s *Server) sendAll(msg string) {
	for _, c := range s.connections {
		c.Write(msg)
	}
}

func (s *Server) openDatabase() {
	log.Println("open database")
	var err error
	s.db, err = sql.Open("mysql", "root:mrp520@/game")
	s.checkError(err)
}

func (s *Server) closeDatabase() {
	err := s.db.Close()
	s.checkError(err)
}

func (s *Server) offlineMsgStore(b *pack, offId []string) {
	log.Println("store offline message")
	var affect int
	stmt, err := s.db.Prepare("INSERT offlinemessage SET duid=?, suid=?, message=?, type=?")
	s.checkError(err)

	for _, d := range offId {
		_, err := stmt.Exec(d, b.sUid, b.msg, b.t)
		s.checkError(err)
		affect++
	}

	log.Println("affect : ", affect)
}

func (s *Server) Listen() {
	log.Println("Listening server . . .")

	onConnected := func(ws *websocket.Conn) {
		log.Println("new connect . . .")
		defer func() {
			err := ws.Close()
			if err != nil {
				s.errCh <- err
			}
		}()

		client := NewClient(ws, s)
		s.Register(client)
		client.Listen()
	}
	http.Handle(s.pattern, websocket.Handler(onConnected))
	log.Println("Created handler")

	for {
		select {
		case c := <-s.register:
			s.connections[c.uid] = c
			log.Println("New Client : ", c.uid, s.connections)
		case c := <-s.unregister:
			log.Println("Delete Client : ", c.uid)
			delete(s.connections, c.uid)
			close(c.send)
		case bmsg := <-s.broadcast:
			log.Println("broadcast : ", bmsg)
			s.history = append(s.history, bmsg)
			s.sendAll(bmsg)
		case tr := <-s.transfer: // Responsible for distributing information(include one-to-one、one-to-many)
			log.Println(tr)
			s.openDatabase()
			var off []string
			for _, g := range tr.dUid {
				c := s.connections[g]
				if c == nil {
					log.Println(g, "offline . . .")
					off = append(off, g)
					continue
				}
				select {
				case c.send <- tr.msg:
				}
			}
			s.offlineMsgStore(tr, off)
			s.closeDatabase()
		case err := <-s.errCh: // [bug] this dosen's work well
			log.Println(err.Error())
		case <-s.doneCh: // when server close
			log.Println("done")
			return
		}
	}
}
