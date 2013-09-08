/*
 * Author	: xsuii
 * Usage	: an android game server
 */

package main

import (
	"code.google.com/p/go.net/websocket"
	"flag"
	"fmt"
	"net/http"

	log "github.com/cihub/seelog"
	"github.com/xsuii/dontsettle/chat"
	"github.com/xsuii/dontsettle/identify"
)

func loadLogAppComfig() {
	logConfig := `
<seelog type="sync">
	<outputs formatid="dontsettle">
		<console />
		<file path="log/log.log" />
	</outputs>
	<formats>
		<format id="dontsettle" format="donsettle: [%LEV] %Msg%n" />
	</formats>
</seelog>
`
	logger, err := log.LoggerFromConfigAsBytes([]byte(logConfig))
	if err != nil {
		fmt.Println(err)
		return
	}
	log.ReplaceLogger(logger)
	chat.UseLogger(logger)
	identify.UseLogger(logger)
}

var addr = flag.String("addr", ":8001", "http service address") // default listening port is 8000

func main() {
	defer chat.FlushLog()
	defer log.Flush()
	loadLogAppComfig()
	flag.Parse()
	//log.SetFlags(log.Lshortfile) // log begin with file and line number

	server := chat.NewServer("/chat")
	go server.Listen()

	http.Handle("/login", websocket.Handler(identify.Login))
	http.Handle("/register", websocket.Handler(identify.Register))

	http.Handle("/", http.FileServer(http.Dir("www"))) // web root

	log.Info("Listening on port", *addr)
	log.Critical(http.ListenAndServe(*addr, nil)) // run server
}
