/*
 * Author	: xsuii
 * Usage	: an android game server
 */

package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"runtime"

	mainlog "github.com/cihub/seelog"
	"github.com/xsuii/dontsettle/servergo/xserver"
)

// get program's operation system target
var platform = runtime.GOOS

func loadLogComfig() {
	logger, err := mainlog.LoggerFromConfigAsFile("conf/log/" + platform + "/color.xml") //{ should change according to platform }//
	if err != nil {
		fmt.Println(err)
		return
	}
	mainlog.ReplaceLogger(logger)
	xserver.UseLogger(logger)
}

var addr = flag.String("addr", ":8001", "http service address") // default listening port is 8000

func main() {
	defer xserver.FlushLog()
	defer mainlog.Flush()
	loadLogComfig()
	flag.Parse()
	log.SetFlags(log.Lshortfile)

	// seelog debug
	mainlog.Info("--------------- color test ---------------")
	mainlog.Trace("trace")
	mainlog.Debug("debug")
	mainlog.Info("info")
	mainlog.Warn("warn")
	mainlog.Error("error")
	mainlog.Critical("critical")
	mainlog.Info("------------------------------------------")

	server := xserver.NewServer("/login", "/mlogin") // "/login" pattern for client, and "/mlogin" pattern for manager
	go server.Listen()

	// www route
	//http.Handle("/login", websocket.Handler(identify.Login))
	//http.Handle("/register", websocket.Handler(identify.Register))

	// server root
	http.Handle("/", http.FileServer(http.Dir("www"))) // web root

	mainlog.Info("Listening on port", *addr)
	mainlog.Critical(http.ListenAndServe(*addr, nil)) // run server
}
