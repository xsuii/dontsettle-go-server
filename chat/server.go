package chat

import (
	"code.google.com/p/go.net/websocket"
	"log"
	"net/http"
)

// for one-to-one chat
type single struct {
	toid string
	msg  string
}

type group struct {
	members []string
	msg     string
}

type Server struct {
	pattern     string
	history     []string
	connections map[string]*connection // Registered connections
	register    chan *connection
	unregister  chan *connection
	broadcast   chan string
	biunique    chan *single
	togroup     chan *group
	errCh       chan error
	doneCh      chan bool
}

func NewServer(pattern string) *Server {
	history := []string{}
	connections := make(map[string]*connection)
	register := make(chan *connection)
	unregister := make(chan *connection)
	broadcast := make(chan string)
	biunique := make(chan *single)
	togroup := make(chan *group)
	errCh := make(chan error)
	doneCh := make(chan bool)

	return &Server{
		pattern,
		history,
		connections,
		register,
		unregister,
		broadcast,
		biunique,
		togroup,
		errCh,
		doneCh,
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

func (s *Server) Biunique(b *single) {
	s.biunique <- b
}

func (s *Server) Done() {
	s.doneCh <- true
}

func (s *Server) Err(err error) {
	s.errCh <- err
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
		case gp := <-s.togroup: // one to many
			log.Println("chat to group : ")
			log.Println(gp.members)
			for _, g := range gp.members {
				c := s.connections[g]
				if c == nil {
					log.Println(g, "offline")
					continue
				}
				select {
				case c.send <- gp.msg:
				}
			}
		case sin := <-s.biunique: // one to one
			log.Println("single chat send to : ", sin.toid)
			c := s.connections[sin.toid]
			if c == nil {
				log.Println("offline . . .")
				break
			}
			c.send <- sin.msg
		case err := <-s.errCh:
			log.Println(err.Error())
		case <-s.doneCh: // when server close
			log.Println("done")
			return
		}
	}
}
