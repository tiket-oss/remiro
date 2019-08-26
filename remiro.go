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

// RunWithSignal creates a new listener with specified address on TCP network.
// It also passes nil or error to signal
func RunWithSignal(addr string, handler Handler, signal chan error) error {
	s := redcon.NewServer(addr, handler.Handle, handler.Accept, handler.Closed)
	return s.ListenServeAndSignal(signal)
}

// Handler provide set of methods to handle incoming connection
type Handler interface {
	Handle(conn redcon.Conn, cmd redcon.Command)
	Accept(conn redcon.Conn) bool
	Closed(conn redcon.Conn, err error)
}
