package remiro

import (
	"bufio"
	"log"
	"net"
	"strings"

	"github.com/secmask/go-redisproto"
)

// Handler provide set of methods to handle incoming connection
type Handler interface {
	Serve(c net.Conn)
}

type redisHandler struct {
}

func (r *redisHandler) Serve(c net.Conn) {
	defer c.Close()

	parser := redisproto.NewParser(c)
	writer := redisproto.NewWriter(bufio.NewWriter(c))

	var ew error
	for {
		command, err := parser.ReadCommand()
		if err != nil {
			if _, ok := err.(*redisproto.ProtocolError); ok {
				writer.WriteError(err.Error())
			} else {
				log.Println(err, " closed connection to ", c.RemoteAddr())
				break
			}
		} else {
			cmd := strings.ToUpper(string(command.Get(0)))
			switch cmd {
			case "GET":
				ew = writer.WriteBulkString("dummy")
			case "SET":
				ew = writer.WriteBulkString("OK")
			default:
				ew = writer.WriteError("Command not supported")
			}
		}

		if command.IsLast() {
			writer.Flush()
		}
		if ew != nil {
			log.Println("Connection closed", ew)
			break
		}
	}
}

// NewRedisHandler returns new instance of redisHandler, a connection
// handler that handler redis-like interface
func NewRedisHandler() Handler {
	return &redisHandler{}
}
