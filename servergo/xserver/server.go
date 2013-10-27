package xserver

import (
	"code.google.com/p/go.net/websocket"
	"database/sql"
	"encoding/json"
	_ "github.com/go-sql-driver/mysql"
	"html/template"
	"net/http"
	"time"
)

var _ = time.Second

const (
	// operation code
	OpNull               = 0
	OpMaster             = 1 // this present master's message, include bad-package...
	OpLogin              = 2
	OpRegister           = 3
	OpChat               = 4
	OpFileTransfer       = 5
	OpFileUpld           = 6
	OpFileDownld         = 7
	OpFileUpldReq        = 8
	OpFileDownldReq      = 9
	OpChatToOne          = 10
	OpChatToMuti         = 11
	OpChatBroadcast      = 12
	OpFileUpldReqAckOk   = 13
	OpError              = 14
	OpFileTicket         = 15
	OpFileDownldReqAckOk = 16

	// error code
	ErrFileUpReqAck = 1
	ErrOperation    = 2
	ErrBadPackage   = 3

	// system id
	NullId      = 0
	MasterId    = 10000
	BroadCastId = 10001 //
)

var Operation = map[byte]string{
	0:  "OpNull",
	1:  "OpMaster",
	2:  "OpLogin",
	3:  "OpRegister",
	4:  "OpChat",
	5:  "OpFileTransfer",
	6:  "OpFileUpld",
	7:  "OpFileDownld",
	8:  "OpFileUpldReq",
	9:  "OpFileDownldReq",
	10: "OpChatToOne",
	11: "OpChatToMulti",
	12: "OpChatBroadcast",
	13: "OpFileUpldReqAckOk",
	14: "OpError",
	15: "OpFileTicket",
	16: "OpFileDownldReqAckOk",
}

type IdType uint64 // use this way to achieve easy-extension

type ResponseError struct {
	Code    int    // error code
	Message string //error message
}

type ServerError struct {
	Message string
}

func (e ServerError) Error() string {
	return e.Message
}

// [TODO] Jesus, The 'Reciever' should be 'Receiver'
type Pack struct {
	Sender    uint64 // sender's id
	Reciever  uint64
	Body      []byte
	TimeStamp int64
	OpCode    byte
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
	toMuti         chan *Pack
	toOne          chan *Pack
	errCh          chan error
	doneCh         chan bool
	db             *sql.DB
	fileMan        *FileManager
}

func NewServer(cPattern string, mPattern string) *Server {
	history := []Pack{}
	connections := make(map[uint64]*connection)
	register := make(chan *connection)
	unregister := make(chan *connection)
	toMuti := make(chan *Pack)
	toOne := make(chan *Pack)
	errCh := make(chan error)
	doneCh := make(chan bool)
	db := &sql.DB{}
	sv := &Server{
		cPattern,
		mPattern,
		history,
		connections,
		register,
		unregister,
		toMuti,
		toOne,
		errCh,
		doneCh,
		db,
		&FileManager{},
	}
	sv.fileMan = NewFileManager(sv)
	return sv
}

func (s *Server) checkLogin(username string, userpasswd string) (uint64, error) {
	var effect int
	var uid uint64

	s.openDatabase("[Func:checkLogin]")
	defer func() {
		s.closeDatabase("[Func:checkLogin]")
	}()

	stmt, err := s.db.Prepare("select userId, userName, userPassword from user where userName=? && userPassword=?")
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

		logger.Tracef("MySQL excute result: { UID:%v, Username:%v, Password:%v }", uid, username, userpassword)
	}

	if effect > 0 {
		logger.Tracef("%v(uid) login success.", uid)
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

func (s *Server) offlineMsgStore(p *Pack, offId []uint64) {
	logger.Info("store offline message")
	s.openDatabase("[OP:Offline messsage store]")
	defer func() {
		s.closeDatabase("[OP:Offline messsage store]")
	}()
	var affect int
	stmt, err := s.db.Prepare("INSERT offline_message SET offMsgReciever=?, offMsgSender=?, offMsgTimeStamp=?, offMsgBody=?, offMsgOpCode=?")
	if err != nil {
		logger.Error("MySQL request error:", err.Error())
		return
	}

	for _, d := range offId {
		_, err := stmt.Exec(d, p.Sender, p.TimeStamp, p.Body, p.OpCode)
		if err != nil {
			logger.Error("MySQL excute error:", err.Error())
			return
		}
		affect++
	}

	logger.Tracef("Store total %v msg.", affect)
}

func (s *Server) NewPack(sender uint64, Reciever uint64, timestamp int64, opcode byte, body []byte) *Pack {
	return &Pack{
		Sender:    sender,
		Reciever:  Reciever,
		TimeStamp: timestamp,
		OpCode:    opcode,
		Body:      body,
	}
}

// showing a pack
func (s *Server) showPack(who string, act string, p Pack) {
	var prefix, suffix string
	switch act { // color print
	case "receive":
		prefix = "\x1b[38;5;167m"
		suffix = "\x1b[0m"
	case "send":
		prefix = "\x1b[38;5;178m"
		suffix = "\x1b[0m"
	}

	logger.Tracef("%v "+prefix+"%v"+suffix+" { Sender:%v, Reciever:%v, TimeStamp:%v, "+prefix+"OpCode:%v, Body:%v"+suffix+" }",
		who,
		act,
		p.Sender,
		p.Reciever,
		p.TimeStamp,
		Operation[p.OpCode],
		string(p.Body))
}

// check the validity of package
func (s *Server) checkPackValid(p Pack) error {
	logger.Info("Check package's valid.")
	if !(p.Reciever != NullId &&
		p.Sender != NullId &&
		p.Body != nil &&
		p.TimeStamp != 0 &&
		p.OpCode != OpNull) {
		return ServerError{
			"Receive bad package",
		}
	} else {
		return nil
	}
}

func (s *Server) errorWrapper(code int, errMsg string) []byte {
	wrap := &ResponseError{
		Code:    code,
		Message: errMsg,
	}
	bWrap, err := json.Marshal(wrap)
	if err != nil {
		logger.Error(err.Error())
	}
	return bWrap
}

// server's feedback message where a client's wrong request or action
func (s *Server) masterPack(c *connection, body []byte) {
	p := &Pack{
		Sender:    MasterId,
		Reciever:  c.uid,
		Body:      body,
		TimeStamp: getTimeStamp(),
		OpCode:    OpMaster}
	websocket.JSON.Send(c.ws, p)
}

func (s *Server) clientHandler() {
	clientConnected := func(ws *websocket.Conn) {
		defer func() {
			logger.Info("connection close!")
			err := ws.Close()
			if err != nil {
				logger.Error(err.Error())
			}
		}()

		client := NewClient(ws, s)
		if client != nil {
			//s.Register(client)
			s.register <- client
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

func (s *Server) Listen() {
	logger.Info("Server Listening.")

	go s.fileMan.FileRoute()

	// create server handler
	s.clientHandler()
	s.managerHandler()
	s.serverState()

	for {
		select {
		case c := <-s.register:
			s.connections[c.uid] = c
			logger.Tracef("Client %v(uid) Register.", c.uid)
			s.showConnections()
			//c.DoneSync()
		case c := <-s.unregister:
			logger.Tracef("Delete Client %v(uid).", c.uid)
			delete(s.connections, c.uid)
			close(c.send)
		case sin := <-s.toOne:
			c := s.connections[sin.Reciever]
			if c != nil {
				c.send <- sin
			} else {
				s.offlineMsgStore(sin, []uint64{sin.Reciever})
			}
		case mult := <-s.toMuti: // Responsible for distributing information(include one-to-one、one-to-many)
			var off []uint64
			var offCount = 0
			err, recvs := s.GetUids(mult)
			if err != nil {
				logger.Error("Get uid error : ", err.Error())
				return
			}
			for _, r := range recvs { // forwarding message
				c := s.connections[r]
				if c == nil { // offline user
					offCount++
					off = append(off, r)
					continue
				}
				select {
				case c.send <- mult:
				}
			}
			logger.Tracef("%v user offline.", offCount)
			if len(off) > 0 {
				s.offlineMsgStore(mult, off)
			}
		case err := <-s.errCh: // [bug] this dosen's work well
			logger.Error(err.Error())
		case <-s.doneCh: // when server close
			logger.Info("done")
			return
		}
	}
}

func (s *Server) showConnections() {
	var ids []uint64
	for i, _ := range s.connections {
		ids = append(ids, i)
	}
	logger.Tracef("Current connections : %v", ids)
}

func (s *Server) GetUids(p *Pack) (error, []uint64) {
	var (
		uid   uint64
		dUids []uint64
		stmt  *sql.Stmt
		rows  *sql.Rows
		err   error
	)
	s.openDatabase("[Func:GetUids]")
	defer func() {
		s.closeDatabase("[Func:GetUids]")
	}()

	switch p.OpCode {
	case OpChatToMuti:
		logger.Trace("Get group ", p.Reciever, "'s uid")
		stmt, err = s.db.Prepare("SELECT userId FROM in_circle WHERE circleId in(SELECT circleId FROM game.circle WHERE circleId=?)")
		if err != nil {
			return err, nil
		}
		rows, err = stmt.Query(p.Reciever)
		if err != nil {
			return err, nil
		}
	case OpChatBroadcast:
		logger.Info("Broadcast message.")
		rows, err = s.db.Query("SELECT userId From user")
		if err != nil {
			return err, nil
		}
	}

	logger.Debug("Scan mysql excute result.")
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

// add client
func (s *Server) Register(c *connection) {
	s.register <- c
}

// delete client
func (s *Server) Unregister(c *connection) {
	s.unregister <- c
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

func (s *Server) sendAll(pack *Pack) {
	for _, c := range s.connections {
		c.Write(pack)
	}
}
