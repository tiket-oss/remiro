// Package remiro provides service to manipulate request across several redis instances.
//
// It works by mimicking redis, listening for packets over TCP that complies with
// REdis Serialization Protocol (RESP), and then parsing the Redis command which
// might be modified before being routed against a real Redis server.
package remiro

import (
	"fmt"
	"log"
	"net"
)

const (
	connType = "tcp"
)

// Config holds configuration for instantiating a net.Listener.
type Config struct {
	Host string
	Port string
}

// Run creates a new Listener with specified host and port on TCP network.
// It will also need a handler function to serve incoming connection.
func Run(cfg Config, handler func(c net.Conn)) {
	l, err := net.Listen(connType, fmt.Sprintf("%s:%s", cfg.Host, cfg.Port))
	if err != nil {
		log.Fatal(fmt.Errorf("Error listening: %v", err))
	}
	defer l.Close()

	for {
		c, err := l.Accept()
		if err != nil {
			log.Print(fmt.Errorf("Error accepting connection: %v", err))
		}

		go handler(c)
	}
}
