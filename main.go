/*
 * Author	: xsuii
 * Usage	: an android game server
 */

package main

import (
	"code.google.com/p/go.net/websocket"
	"flag"
	"log"
	"net/http"

	"github.com/xsuii/dontsettle/chat"
	"github.com/xsuii/dontsettle/identify"
)

var addr = flag.String("addr", ":8001", "http service address") // default listening port is 8000

func main() {
	flag.Parse()
	log.SetFlags(log.Lshortfile) // log begin with file and line number

	server := chat.NewServer("/chat")
	go server.Listen()

	http.Handle("/login", websocket.Handler(identify.Login))
	http.Handle("/register", websocket.Handler(identify.Register))

	http.Handle("/", http.FileServer(http.Dir("www"))) // web root

	log.Println("Listening on port", *addr)
	log.Fatal(http.ListenAndServe(*addr, nil)) // run server
}
