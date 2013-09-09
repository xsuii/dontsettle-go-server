/*
 * Author	: xsuii
 * Usage	: an android game server
 */

package main

import (
	"code.google.com/p/go.net/websocket"
	"flag"
	"fmt"
	"log"
	"net/http"

	mainlog "github.com/cihub/seelog"
	"github.com/xsuii/dontsettle/chat"
	"github.com/xsuii/dontsettle/identify"
)

func loadLogAppComfig() {
	logger, err := mainlog.LoggerFromConfigAsFile("conf/log/color.xml")
	if err != nil {
		fmt.Println(err)
		return
	}
	mainlog.ReplaceLogger(logger)
	chat.UseLogger(logger)
	identify.UseLogger(logger)
}

var addr = flag.String("addr", ":8001", "http service address") // default listening port is 8000

func main() {
	defer chat.FlushLog()
	defer mainlog.Flush()
	loadLogAppComfig()
	flag.Parse()
	log.SetFlags(log.Lshortfile)

	mainlog.Trace("trace")
	mainlog.Debug("debug")
	mainlog.Info("info")
	mainlog.Warn("warn")
	mainlog.Error("error")
	mainlog.Critical("critical")

	server := chat.NewServer("/chat")
	go server.Listen()

	http.Handle("/login", websocket.Handler(identify.Login))
	http.Handle("/register", websocket.Handler(identify.Register))

	http.Handle("/", http.FileServer(http.Dir("www"))) // web root

	mainlog.Info("Listening on port", *addr)
	mainlog.Critical(http.ListenAndServe(*addr, nil)) // run server
}
