// Package server provides implementation for creating a TCP server which will
// listen to a specific port.
package server

import (
	"fmt"
	"log"
	"net"
)

const (
	connType = "tcp"
)

// Config holds configuration for instantiating a net.Listener
type Config struct {
	Host string
	Port string
}

// Run creates a new Listener with specified host and port on TCP network
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
