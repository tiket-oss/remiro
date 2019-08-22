package remiro

import (
	"fmt"
	"strings"
	"time"

	"github.com/gomodule/redigo/redis"
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/redcon"
)

// redisHandler is an implementation of Handler
type redisHandler struct {
	sourcePool      *redis.Pool
	destinationPool *redis.Pool
	deleteOnGet     bool
	deleteOnSet     bool
}

var replyTypeBytes = []byte{'+', '-', ':', '$', '*'}

func (r *redisHandler) Handle(conn redcon.Conn, cmd redcon.Command) {
	command := strings.ToUpper(string(cmd.Args[0]))
	switch command {
	case "GET":
		// NOTE: this is necessary as []interface{}, that are used in
		// Do() method, have different memory representation, see:
		// https://golang.org/doc/faq#convert_slice_of_interface
		args := make([]interface{}, len(cmd.Args)-1)
		for i, v := range cmd.Args[1:] {
			args[i] = v
		}

		dstConn := r.destinationPool.Get()
		defer dstConn.Close()

		reply, err := redis.String(dstConn.Do(command, args...))
		if err == nil {
			conn.WriteBulkString(reply)
			break
		}

		if err != redis.ErrNil {
			conn.WriteError(err.Error())
			break
		}

		srcConn := r.sourcePool.Get()
		defer srcConn.Close()

		reply, err = redis.String(srcConn.Do(command, args...))
		if err == nil {
			val := reply
			key := cmd.Args[1]

			_, err = redis.String(dstConn.Do("SET", key, val))
			if err != nil {
				log.Error(fmt.Errorf("Error when setting key %s: %v", key, err))
			}

			if r.deleteOnGet && err == nil {
				if err := deleteKey(srcConn, key); err != nil {
					log.Error(err)
				}
			}
		}
		conn.WriteBulkString(reply)

	case "SET":
		args := make([]interface{}, len(cmd.Args)-1)
		for i, v := range cmd.Args[1:] {
			args[i] = v
		}

		dstConn := r.destinationPool.Get()
		defer dstConn.Close()

		reply, err := redis.String(dstConn.Do(command, args...))
		if err != nil {
			conn.WriteError(err.Error())
			break
		}

		if r.deleteOnSet && err == nil {
			key := cmd.Args[1]

			srcConn := r.sourcePool.Get()
			defer srcConn.Close()

			if err := deleteKey(srcConn, key); err != nil {
				log.Error(err)
			}
		}

		conn.WriteString(reply)

	case "PING":
		conn.WriteString("PONG")

	default:
		args := make([]interface{}, len(cmd.Args)-1)
		for i, v := range cmd.Args[1:] {
			args[i] = v
		}

		dstConn := r.destinationPool.Get()
		defer dstConn.Close()

		reply, err := dstConn.Do(command, args...)
		if err != nil {
			if _, ok := err.(redis.Error); !ok {
				log.Error(fmt.Errorf("Error when executing command %s: %v", command, err))
			}
		}

		writeResponse(conn, reply)
	}
}

func (r *redisHandler) Accept(conn redcon.Conn) bool {
	return true
}

func (r *redisHandler) Closed(conn redcon.Conn, err error) {

}

// NewRedisHandler returns new instance of redisHandler, a connection
// handler that handler redis-like interface
func NewRedisHandler(srcURL, dstURL string) Handler {
	return &redisHandler{
		sourcePool:      newRedisPool(srcURL),
		destinationPool: newRedisPool(dstURL),
		deleteOnGet:     true,
		deleteOnSet:     true,
	}
}

func newRedisPool(addr string) *redis.Pool {
	return &redis.Pool{
		MaxIdle:     100,
		IdleTimeout: 30 * time.Second,
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", addr)
		},
	}
}

func deleteKey(conn redis.Conn, key []byte) error {
	nDel, err := redis.Int(conn.Do("DEL", key))
	if nDel == 0 || err != nil {
		return fmt.Errorf("Error when deleting key %s: %v (%d deleted)", key, err, nDel)
	}

	return nil
}

func writeResponse(conn redcon.Conn, reply interface{}) {
	switch resp := reply.(type) {
	case nil:
		conn.WriteNull()
	case error:
		conn.WriteError(resp.Error())
	case string:
		conn.WriteString(resp)
	case []byte:
		if isRawReply(resp) {
			conn.WriteRaw(resp)
		} else {
			conn.WriteBulk(resp)
		}
	case int:
		conn.WriteInt(resp)
	case int64:
		conn.WriteInt64(resp)
	case []interface{}:
		conn.WriteArray(len(resp))
		for _, res := range resp {
			writeResponse(conn, res)
		}
	default:
		msg := fmt.Sprintf("Unrecognized reply: %v", resp)
		conn.WriteError(msg)
	}
}

func isRawReply(reply []byte) bool {
	for _, replyType := range replyTypeBytes {
		if reply[0] == replyType {
			return true
		}
	}

	return false
}
