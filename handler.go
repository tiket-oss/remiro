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
		args := toInterfaceSlice(cmd.Args[1:])

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
		if err != nil {
			log.Error(fmt.Errorf("Error when setting key %s: %v", key, err))
		}

		if r.deleteOnGet && err == nil {
			if err := deleteKey(srcConn, key); err != nil {
				log.Error(err)
			}
		}

		conn.WriteBulkString(reply)

	case "SET":
		args := toInterfaceSlice(cmd.Args[1:])

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
		args := toInterfaceSlice(cmd.Args[1:])

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
