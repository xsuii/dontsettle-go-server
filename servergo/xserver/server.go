package xserver

import (
	"code.google.com/p/go.net/websocket"
	"database/sql"
	_ "github.com/Go-SQL-Driver/MySQL"
	"html/template"
	"net/http"
	"time"
)

// operation code
const (
	OpNull         = 0
	OpMaster       = 1 // this present master's message, include bad-package...
	OpLogin        = 2
	OpRegister     = 3
	OpChat         = 4
	OpFileTransfer = 5

	// system id
	NullId      = 0
	MasterId    = 10000
	BroadCastId = 10001

	// ForwardType
	FwGroup     = 1
	FwSingle    = 2
	FwBroadcast = 3
)

type IdType uint64 // use this way to achieve easy-extension

type Pack struct {
	Sender      uint64 // sender's id
	Receiver    uint64
	Body        []byte // filename when Type=file
	TimeStamp   int64
	OpCode      byte //
	ForwardType byte // could be group, single, broadcast define in const
}

// for one-to-one chat  [later:this can merge with group struct]
type Postman struct {
	sUid  uint64
	dUids []uint64
	pack  *Pack
}

type ServerState struct {
	Online int
}

type Server struct {
	clientPattern  string
	managerPattern string
	history        []Pack
	connections    map[uint64]*connection // Registered connections
	register       chan *connection
	unregister     chan *connection
	broadcast      chan Pack
	postman        chan *Postman
	errCh          chan error
	doneCh         chan bool
	db             *sql.DB
}

func NewServer(cPattern string, mPattern string) *Server {
	history := []Pack{}
	connections := make(map[uint64]*connection)
	register := make(chan *connection)
	unregister := make(chan *connection)
	broadcast := make(chan Pack)
	postman := make(chan *Postman)
	errCh := make(chan error)
	doneCh := make(chan bool)
	db := &sql.DB{}

	return &Server{
		cPattern,
		mPattern,
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

func (s *Server) checkLogin(username string, userpasswd string) (uint64, error) {
	var effect int
	var uid uint64

	s.openDatabase("Func_checkLogin():")
	defer func() {
		s.closeDatabase("Func_checkLogin()")
	}()

	stmt, err := s.db.Prepare("select UID, username, userpassword from user where username=? && userpassword=?")
	if err != nil {
		return 0, err
	}

	rows, err := stmt.Query(username, userpasswd) // temp contants username and password which split before
	if err != nil {
		return 0, err
	}

	for rows.Next() {
		var userpassword string
		effect++

		err = rows.Scan(&uid, &username, &userpassword)
		if err != nil {
			return 0, err
		}

		logger.Trace("MySQL : [ UID:", uid, " ]  [ Username:", username, " ]  [ Password:", userpassword, " ]")
	}

	if effect > 0 {
		logger.Trace(uid, "(uid) login success.")
		return uid, nil
	} else {
		logger.Warn("login fail . . .")
		return 0, nil
	}
}

func (s *Server) openDatabase(who string) {
	logger.Trace(who, ":open database")
	var err error
	s.db, err = sql.Open("mysql", "root:mrp520@/game")
	if err != nil {
		logger.Error("Error:", err.Error())
	}
}

func (s *Server) closeDatabase(who string) {
	logger.Trace(who, ":close database")
	err := s.db.Close()
	if err != nil {
		logger.Error("Error:", err.Error())
	}
}

func (s *Server) offlineMsgStore(b *Postman, offId []uint64) {
	logger.Info("store offline message")
	var affect int
	stmt, err := s.db.Prepare("INSERT offlinemessage SET Receiver=?, Sender=?, TimeStamp=?, Body=?, OpCode=?, ForwardType=?")
	if err != nil {
		logger.Error("Error:", err.Error())
		return
	}

	for _, d := range offId {
		_, err := stmt.Exec(d, b.sUid, b.pack.TimeStamp, b.pack.Body, b.pack.OpCode, b.pack.ForwardType)
		if err != nil {
			logger.Error("Error:", err.Error())
			return
		}
		affect++
	}

	logger.Trace("affect : ", affect)
}

func (s *Server) clientHandler() {
	clientConnected := func(ws *websocket.Conn) {
		logger.Info("new client connect")
		defer func() {
			logger.Info("connection close!")
			err := ws.Close()
			if err != nil {
				logger.Error(err.Error())
			}
		}()

		client := NewClient(ws, s)
		if client != nil {
			s.Register(client)
			client.Listen()
		}
	}
	http.Handle(s.clientPattern, websocket.Handler(clientConnected))
}

func (s *Server) managerHandler() {
	managerConnected := func(ws *websocket.Conn) {
		logger.Info("new manager connect")
		defer func() {
			logger.Info("manager connection close")
			err := ws.Close()
			if err != nil {
				logger.Error(err.Error())
			}
		}()

		manager := NewManager(ws, s)
		if manager != nil {
			s.Register(manager)
			manager.Listen()
		}
	}

	http.Handle(s.managerPattern, websocket.Handler(managerConnected))
}

func (s *Server) getState() ServerState {
	return ServerState{Online: len(s.connections)}
}

func (s *Server) serverState() {
	state := func(w http.ResponseWriter, r *http.Request) {
		t := template.New("Server state")
		t, _ = t.Parse(
			`<head>
			<title>Server State</title>
			</head>
			<body>
			<h1>Server State</h1>
			Online: {{.Online}}
			</body>`)
		st := s.getState()
		t.Execute(w, st)
	}
	http.HandleFunc("/state", state)
}

func (s *Server) pack(sender uint64, receiver uint64, body []byte, timestamp int64, opcode byte, forwardtype byte) Pack {
	return Pack{
		Sender:      sender,
		Receiver:    receiver,
		Body:        body,
		TimeStamp:   timestamp,
		OpCode:      opcode,
		ForwardType: forwardtype,
	}
}

func NewPostman(suid uint64, duids []uint64, pack *Pack) Postman {
	return Postman{
		sUid:  suid,
		dUids: duids,
		pack:  pack,
	}
}

// showing a pack
func (s *Server) showPack(who string, act string, p Pack) {
	logger.Tracef("\n%s %s package:"+
		"\n%-20s%-20s%-20s%-15s%-7s%s"+
		"\n%-20v%-20v%-20v%-15v%-7v%v",
		who, act,
		"Sender", "Receiver", "TimeStamp", "ForwardType", "OpCode", "Body",
		p.Sender, p.Receiver, p.TimeStamp, p.ForwardType, p.OpCode, p.Body)
}

// check the validity of package
func (s *Server) validPack(p Pack) bool {
	return p.Receiver != NullId &&
		p.Sender != NullId &&
		p.Body != nil &&
		p.TimeStamp != 0 &&
		p.ForwardType != ' ' &&
		p.OpCode != OpNull
}

// server's feedback message where a client's wrong request or action
func (s *Server) masterPack(c *connection, body []byte) {
	p := &Pack{
		Sender:      MasterId,
		Receiver:    c.uid,
		Body:        body,
		TimeStamp:   s.getTimeStamp(),
		OpCode:      OpMaster,
		ForwardType: FwSingle}
	websocket.JSON.Send(c.ws, p)
}

func (s *Server) Listen() {
	logger.Info("Listening server . . .")

	// create handler
	s.clientHandler()
	s.managerHandler()
	s.serverState()

	for {
		select {
		case c := <-s.register:
			s.connections[c.uid] = c
			logger.Trace("Client Register : ", c.uid)
			logger.Trace("Current connection :", s.connections)
		case c := <-s.unregister:
			logger.Trace("Delete Client : ", c.uid)
			delete(s.connections, c.uid)
			close(c.send)
		case bmsg := <-s.broadcast:
			logger.Trace("broadcast : ", bmsg)
			//s.history = append(s.history, bmsg)
			s.sendAll(bmsg)
		case tr := <-s.postman: // Responsible for distributing information(include one-to-one、one-to-many)
			logger.Trace("postman :", tr)
			s.openDatabase("Postman")
			var off []uint64
			logger.Trace("postman check connect:", s.connections)

			for _, g := range tr.dUids {
				c := s.connections[g]
				if c == nil {
					logger.Trace(g, "offline . . .")
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
			logger.Error(err.Error())
		case <-s.doneCh: // when server close
			logger.Info("done")
			return
		}
	}
}

func (s *Server) getTimeStamp() int64 {
	return time.Now().Unix()
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
