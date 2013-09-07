package chat

import (
	"code.google.com/p/go.net/websocket"
	"database/sql"
	_ "github.com/Go-SQL-Driver/MySQL"
	"log"
	"net/http"
)

type Pack struct {
	Author    string
	Addressee string
	Message   string // filename when Type=file
	DateTime  string
	Type      string // could be [file|meg]
	DstT      string // could be [G|S|B](group, single, broadcast)
}

// for one-to-one chat  [later:this can merge with group struct]
type Postman struct {
	sUid string
	dUid []string
	pack *Pack
}

type Server struct {
	pattern     string
	history     []Pack
	connections map[string]*connection // Registered connections
	register    chan *connection
	unregister  chan *connection
	broadcast   chan Pack
	postman     chan *Postman
	errCh       chan error
	doneCh      chan bool
	db          *sql.DB
}

func NewServer(pattern string) *Server {
	history := []Pack{}
	connections := make(map[string]*connection)
	register := make(chan *connection)
	unregister := make(chan *connection)
	broadcast := make(chan Pack)
	postman := make(chan *Postman)
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
		postman,
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

func (s *Server) BroadCast(pack Pack) {
	s.broadcast <- pack
}

func (s *Server) Post(p *Postman) {
	s.postman <- p
}

func (s *Server) Done() {
	s.doneCh <- true
}

func (s *Server) Err(err error) {
	s.errCh <- err
}

func (s *Server) sendPastMessages(c *connection) {
	for _, pack := range s.history {
		c.Write(&pack)
	}
}

func (s *Server) sendAll(pack Pack) {
	for _, c := range s.connections {
		c.Write(&pack)
	}
}

func (s *Server) openDatabase(who string) {
	log.Println(who, "open database")
	var err error
	s.db, err = sql.Open("mysql", "root:mrp520@/game")
	if err != nil {
		log.Println("Error:", err.Error())
	}
}

func (s *Server) closeDatabase(who string) {
	log.Println(who, "close database")
	err := s.db.Close()
	if err != nil {
		log.Println("Error:", err.Error())
	}
}

func (s *Server) offlineMsgStore(b *Postman, offId []string) {
	log.Println("store offline message")
	var affect int
	stmt, err := s.db.Prepare("INSERT offlinemessage SET duid=?, suid=?, time=?, message=?, packtype=?, dsttype=?")
	if err != nil {
		log.Println("Error:", err.Error())
	}

	for _, d := range offId {
		_, err := stmt.Exec(d, b.sUid, b.pack.DateTime, b.pack.Message, b.pack.Type, b.pack.DstT)
		if err != nil {
			log.Println("Error:", err.Error())
		}
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
			log.Println("Client Register : ", c.uid)
			log.Println("Current connection :", s.connections)
		case c := <-s.unregister:
			log.Println("Delete Client : ", c.uid)
			delete(s.connections, c.uid)
			close(c.send)
		case bmsg := <-s.broadcast:
			log.Println("broadcast : ", bmsg)
			//s.history = append(s.history, bmsg)
			s.sendAll(bmsg)
		case tr := <-s.postman: // Responsible for distributing information(include one-to-one、one-to-many)
			log.Println("postman :", tr)
			s.openDatabase("Postman")
			var off []string
			log.Println("postman check connect:", s.connections)

			for _, g := range tr.dUid {
				c := s.connections[g]
				if c == nil {
					log.Println(g, "offline . . .")
					off = append(off, g)
					continue
				}
				select {
				case c.send <- tr.pack:
				}
			}
			if len(off) > 0 {
				s.offlineMsgStore(tr, off)
			}
			s.closeDatabase("Postman")
		case err := <-s.errCh: // [bug] this dosen's work well
			log.Println(err.Error())
		case <-s.doneCh: // when server close
			log.Println("done")
			return
		}
	}
}
