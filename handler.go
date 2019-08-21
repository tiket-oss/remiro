package remiro

import (
	"strings"

	"github.com/tidwall/redcon"
)

type redisHandler struct {
}

func (r *redisHandler) Handle(conn redcon.Conn, cmd redcon.Command) {
	switch strings.ToUpper(string(cmd.Args[0])) {
	case "GET":
		resp := r.handleGET(cmd)
		conn.WriteRaw(resp)
	case "SET":
		resp := r.handleSET(cmd)
		conn.WriteRaw(resp)
	case "PING":
		conn.WriteString("PONG")
	case "QUIT":
		conn.WriteString("OK")
		conn.Close()
	}
}

func (r *redisHandler) Accept(conn redcon.Conn) bool {
	return true
}

func (r *redisHandler) Closed(conn redcon.Conn, err error) {

}

func (r *redisHandler) handleGET(cmd redcon.Command) []byte {
	return []byte("+Handling GET\r\n")
}

func (r *redisHandler) handleSET(cmd redcon.Command) []byte {
	return []byte("+Handling SET\r\n")
}

// NewRedisHandler returns new instance of redisHandler, a connection
// handler that handler redis-like interface
func NewRedisHandler() Handler {
	return &redisHandler{}
}
