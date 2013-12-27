// Copyright 2013 xsuii. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//
package xserver

import (
	"code.google.com/p/go.net/websocket"
)

func NewManager(ws *websocket.Conn, s *Server) *connection {
	logger.Info(" # New Connection # ")
	logger.Info("Manager :", ws.Request().RemoteAddr)
	return nil
}
