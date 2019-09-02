// Package remiro provides service to manipulate request across several redis instances.
//
// It works by mimicking redis, listening for packets over TCP that complies with
// REdis Serialization Protocol (RESP), and then parsing the Redis command which
// might be modified before being routed against a real Redis server.
package remiro

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.opencensus.io/stats"
	"go.opencensus.io/tag"

	"github.com/gomodule/redigo/redis"
	log "github.com/sirupsen/logrus"
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

// redisHandler is an implementation of Handler
type redisHandler struct {
	sourcePool      *redis.Pool
	destinationPool *redis.Pool
	deleteOnGet     bool
	deleteOnSet     bool
}

var replyTypeBytes = []byte{'+', '-', ':', '$', '*'}

func (r *redisHandler) Handle(conn redcon.Conn, cmd redcon.Command) {
	startTime := time.Now()

	reqCtx, err := tag.New(context.Background())
	if err != nil {
		log.Warnf("Failed to initialize instrumentation: %v", err)
	}

	defer stats.Record(reqCtx, reqLatencyMs.M(sinceInMs(startTime)))

	command := strings.ToUpper(string(cmd.Args[0]))
	switch command {
	case "GET":
		args := make([]interface{}, 0)
		if len(cmd.Args) > 1 {
			args = toInterfaceSlice(cmd.Args[1:])
		}

		dstConn := r.destinationPool.Get()
		defer dstConn.Close()

		reply, err := redis.String(dstConn.Do(command, args...))
		go recordRedisCmd("destination", "GET")
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
		go recordRedisCmd("source", "GET")
		if err != nil {
			if err == redis.ErrNil {
				conn.WriteNull()
			} else {
				conn.WriteError(err.Error())
			}
			break
		}

		val := reply
		key := cmd.Args[1]

		_, err = redis.String(dstConn.Do("SET", key, val))
		go recordRedisCmd("destination", "SET")
		if err != nil {
			log.Error(fmt.Errorf("Error when setting key %s: %v", key, err))
		}

		if r.deleteOnGet && err == nil {
			if err := deleteKey(srcConn, key); err != nil {
				log.Warn(err)
			}
			go recordRedisCmd("source", "DEL")
		}

		conn.WriteBulkString(reply)

	case "SET":
		args := make([]interface{}, 0)
		if len(cmd.Args) > 1 {
			args = toInterfaceSlice(cmd.Args[1:])
		}

		dstConn := r.destinationPool.Get()
		defer dstConn.Close()

		reply, err := redis.String(dstConn.Do(command, args...))
		go recordRedisCmd("destination", "SET")
		if err != nil {
			conn.WriteError(err.Error())
			break
		}

		if r.deleteOnSet {
			key := cmd.Args[1]

			srcConn := r.sourcePool.Get()
			defer srcConn.Close()

			if err := deleteKey(srcConn, key); err != nil {
				log.Warn(err)
			}
			go recordRedisCmd("source", "DEL")
		}

		conn.WriteString(reply)

	case "PING":
		conn.WriteString("PONG")

	default:
		args := make([]interface{}, 0)
		if len(cmd.Args) > 1 {
			args = toInterfaceSlice(cmd.Args[1:])
		}

		dstConn := r.destinationPool.Get()
		defer dstConn.Close()

		reply, err := dstConn.Do(command, args...)
		go recordRedisCmd("destination", command)
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

type duration struct {
	time.Duration
}

func (d *duration) UnmarshalText(text []byte) error {
	var err error
	d.Duration, err = time.ParseDuration(string(text))
	return err
}

// ClientConfig holds the configuration for Redis client
type ClientConfig struct {
	Addr         string
	MaxIdleConns int
	IdleTimeout  duration
}

// RedisConfig holds configuration for initializing redisHandler
type RedisConfig struct {
	DeleteOnGet bool
	DeleteOnSet bool
	Source      ClientConfig
	Destination ClientConfig
}

// NewRedisHandler returns new instance of redisHandler, a connection
// handler that handler redis-like interface
func NewRedisHandler(config RedisConfig) Handler {
	return &redisHandler{
		sourcePool:      newRedisPool(config.Source),
		destinationPool: newRedisPool(config.Destination),
		deleteOnGet:     config.DeleteOnGet,
		deleteOnSet:     config.DeleteOnSet,
	}
}

func newRedisPool(config ClientConfig) *redis.Pool {
	return &redis.Pool{
		MaxIdle:     config.MaxIdleConns,
		IdleTimeout: config.IdleTimeout.Duration,
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", config.Addr)
		},
	}
}

func deleteKey(conn redis.Conn, key []byte) error {
	_, err := redis.Int(conn.Do("DEL", key))
	if err != nil && err != redis.ErrNil {
		return fmt.Errorf("Error when deleting key %s: %v", key, err)
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

func toInterfaceSlice(args [][]byte) []interface{} {
	iArgs := make([]interface{}, len(args))
	for i, v := range args {
		iArgs[i] = v
	}

	return iArgs
}

func isRawReply(reply []byte) bool {
	for _, replyType := range replyTypeBytes {
		if reply[0] == replyType {
			return true
		}
	}

	return false
}
