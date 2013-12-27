/*
	Description 	: A routine to test max connection of server.
	Date 			: 2013.11.16
*/

package main

import (
	"code.google.com/p/go.net/websocket"
	"fmt"
	"runtime"
	"time"
)

var _ = time.Second
var _ = runtime.GOROOT()
var delay = 500 * time.Millisecond

func main() {
	for {
		go newConn()
		time.Sleep(delay)
	}
}

func newConn() {
	conn, err := websocket.Dial("ws://localhost:8001/login", "", "http://localhost")
	if err != nil {
		fmt.Println("Error:", err.Error())
		return
	}
	defer func() {
		err := conn.Close()
		if err != nil {
			fmt.Println("WebSocket close error:", err.Error())
			return
		}
	}()
	for {
		time.Sleep(delay)
	}
}
