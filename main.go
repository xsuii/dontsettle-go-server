/*
 * Author	: xsuii
 * Usage	: an android game server
 */

package main

import (
	"code.google.com/p/go.net/websocket"
	"flag"
	"github.com/xsuii/dontsettle/identify"
	"log"
	"net/http"
)

var addr = flag.String("addr", ":8001", "http service address") // default listening port is 8000

func main() {
	flag.Parse()

	log.SetFlags(log.Lshortfile) // log begin with file and line number

	http.Handle("/login", websocket.Handler(identify.Login))
	http.Handle("/register", websocket.Handler(identify.Register))

	http.Handle("/", http.FileServer(http.Dir("www"))) // web root

	log.Println("Listening on port", *addr)
	log.Fatal(http.ListenAndServe(*addr, nil)) // run server
}
