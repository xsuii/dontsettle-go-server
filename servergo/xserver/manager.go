/*
	handle manager's operation
*/

package xserver

import (
	"code.google.com/p/go.net/websocket"
)

func NewManager(ws *websocket.Conn, s *Server) *connection {
	logger.Info(" # New Connection # ")
	logger.Info("Manager :", ws.Request().RemoteAddr)
	return nil
}
