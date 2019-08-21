package remiro

import "github.com/tidwall/redcon"

type redisHandler struct {
}

func (r *redisHandler) Handle(conn redcon.Conn, cmd redcon.Command) {

}

func (r *redisHandler) Accept(conn redcon.Conn) bool {
	return true
}

func (r *redisHandler) Closed(conn redcon.Conn, err error) {

}

// NewRedisHandler returns new instance of redisHandler, a connection
// handler that handler redis-like interface
func NewRedisHandler() Handler {
	return &redisHandler{}
}
