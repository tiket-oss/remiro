// Package remiro provides service to manipulate request across several redis instances.
//
// It works by mimicking redis, listening for packets over TCP that complies with
// REdis Serialization Protocol (RESP), and then parsing the Redis command which
// might be modified before being routed against a real Redis server.
package remiro

import (
	"fmt"
	"log"

	"github.com/tidwall/redcon"
)

// Config holds configuration for instantiating a net.Listener.
type Config struct {
	Host string
	Port string
}

// Run creates a new Listener with specified host and port on TCP network.
func Run(cfg Config, handler Handler) {
	addr := fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)
	err := redcon.ListenAndServe(addr, handler.Handle, handler.Accept, handler.Closed)

	if err != nil {
		log.Fatal(err)
	}
}

// Handler provide set of methods to handle incoming connection
type Handler interface {
	Handle(conn redcon.Conn, cmd redcon.Command)
	Accept(conn redcon.Conn) bool
	Closed(conn redcon.Conn, err error)
}
