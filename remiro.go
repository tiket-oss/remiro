// Package remiro provides service to manipulate request across several redis instances.
//
// It works by mimicking redis, listening for packets over TCP that complies with
// REdis Serialization Protocol (RESP), and then parsing the Redis command which
// might be modified before being routed against a real Redis server.
package remiro

import (
	"github.com/tidwall/redcon"
)

// Run creates a new Listener with specified address on TCP network.
func Run(addr string, handler Handler) error {
	return redcon.ListenAndServe(addr, handler.Handle, handler.Accept, handler.Closed)
}

// NewServer returns a new instance of *redcon.Server, useful when you want
// to fine tune the server before running
func NewServer(addr string, handler Handler) *redcon.Server {
	return redcon.NewServer(addr, handler.Handle, handler.Accept, handler.Closed)
}

// Handler provide set of methods to handle incoming connection
type Handler interface {
	Handle(conn redcon.Conn, cmd redcon.Command)
	Accept(conn redcon.Conn) bool
	Closed(conn redcon.Conn, err error)
}
