package remiro

import (
	"net"
)

// Handler provide set of methods to handle redis command
type Handler interface {
	Serve(c net.Conn)
}

type redisHandler struct {
}

func (r *redisHandler) Serve(c net.Conn) {
	defer c.Close()
}

// NewRedisHandler returns new instance of redisHandler
func NewRedisHandler() Handler {
	return &redisHandler{}
}
