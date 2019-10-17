// Package handler defines the mechanics to manipulate request across several redis instances.
//
// It works by mimicking redis, listening for packets over TCP that complies with
// REdis Serialization Protocol (RESP), and then parsing the Redis command which
// might be modified before being routed against a real Redis server.
package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"contrib.go.opencensus.io/exporter/prometheus"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"

	"github.com/gomodule/redigo/redis"
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/redcon"
)

// Handler provide set of methods to handle incoming connection.
type Handler interface {
	Handle(conn redcon.Conn, cmd redcon.Command)
	Accept(conn redcon.Conn) bool
	Closed(conn redcon.Conn, err error)

	HealthCheck(w http.ResponseWriter, req *http.Request)
}

// Run creates a new Listener with specified address on TCP network.
// The Listener will then use a Handler to process incoming connection.
func Run(addr string, handler Handler) error {
	return redcon.ListenAndServe(addr, handler.Handle, handler.Accept, handler.Closed)
}

// NewServer returns a new instance of *redcon.Server, using Handler as its
// connection processor. The difference with Run() function is that the server instance
// is not listening to any address yet, making it useful when you want configure the server
// before running it.
func NewServer(addr string, handler Handler) *redcon.Server {
	return redcon.NewServer(addr, handler.Handle, handler.Accept, handler.Closed)
}

// RunInstrumentation creates and run a HTTP server which provides a couple of endpoints:
// - /health to check server health
// - /metrics to provide instrumentation metrics
func RunInstrumentation(addr string, handler Handler, errSignal chan error) error {
	if err := view.Register(views...); err != nil {
		return err
	}

	pe, err := prometheus.NewExporter(prometheus.Options{
		Namespace: "remiro",
	})
	if err != nil {
		return err
	}

	view.RegisterExporter(pe)
	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", pe)
		mux.Handle("/health", http.HandlerFunc(handler.HealthCheck))
		if err := http.ListenAndServe(addr, mux); err != nil {
			errSignal <- err
		}
	}()

	return nil
}

// redisHandler is an implementation of Handler
type redisHandler struct {
	sourcePool        *redis.Pool
	destinationPool   *redis.Pool
	deleteOnGet       bool
	deleteOnSet       bool
	deletedKey        map[string]bool
	authenticatedAddr map[string]bool
	password          string
	sync.Mutex
}

var (
	replyTypeBytes = []byte{'+', '-', ':', '$', '*'}
	noAuthCmd      = []string{"AUTH", "QUIT"}
	errAuthMsg     = "NOAUTH Authentication required."
)

func (r *redisHandler) Handle(conn redcon.Conn, cmd redcon.Command) {
	startTime := time.Now()
	reqCtx, err := tag.New(context.Background())
	if err != nil {
		log.Warnf("Failed to initialize instrumentation: %v", err)
	}
	defer stats.Record(reqCtx, reqLatencyMs.M(sinceInMs(startTime)))

	log.Trace(logCmd(cmd.Args))

	command := strings.ToUpper(string(cmd.Args[0]))
	if !r.authorizedConn(conn, command) {
		conn.WriteError(errAuthMsg)
		return
	}

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

		if key := cmd.Args[1]; r.deleteOnSet && !r.deletedKey[string(key)] {
			srcConn := r.sourcePool.Get()
			defer srcConn.Close()

			if err := deleteKey(srcConn, key); err != nil {
				log.Warn(err)
			} else {
				r.Lock()
				r.deletedKey[string(key)] = true
				r.Unlock()
			}
			go recordRedisCmd("source", "DEL")
		}

		conn.WriteString(reply)

	case "PING":
		conn.WriteString("PONG")

	case "QUIT":
		conn.WriteString("OK")
		conn.Close()

	case "AUTH":
		if len(cmd.Args) != 2 {
			conn.WriteError("ERR wrong number of arguments for 'auth' command")
			return
		}

		if r.password == "" {
			conn.WriteError("ERR Client sent AUTH, but no password is set")
			return
		}

		var authenticated bool
		pass := string(cmd.Args[1])
		if pass == r.password {
			authenticated = true
			conn.WriteString("OK")
		} else {
			conn.WriteError("ERR invalid password")
		}

		r.Lock()
		r.authenticatedAddr[conn.RemoteAddr()] = authenticated
		r.Unlock()

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
	log.Tracef("Accepting connection from %s", conn.RemoteAddr())
	return true
}

func (r *redisHandler) Closed(conn redcon.Conn, err error) {
	log.Tracef("Connection from %s has been closed", conn.RemoteAddr())

	r.Lock()
	r.authenticatedAddr[conn.RemoteAddr()] = false
	r.Unlock()
}

func (r *redisHandler) HealthCheck(w http.ResponseWriter, req *http.Request) {
	srcConn := r.sourcePool.Get()
	_, srcErr := redis.String(srcConn.Do("PING"))

	dstConn := r.destinationPool.Get()
	_, dstErr := redis.String(dstConn.Do("PING"))

	var status int
	if srcErr != nil || dstErr != nil {
		status = http.StatusInternalServerError
	} else {
		status = http.StatusOK
	}

	buildRedisReport := func(err error) map[string]string {
		report := make(map[string]string)
		if err != nil {
			report["status"] = "Error"
			report["error"] = err.Error()
		} else {
			report["status"] = "OK"
		}

		return report
	}

	body := map[string]map[string]string{
		"sourceRedis":      buildRedisReport(srcErr),
		"destinationRedis": buildRedisReport(dstErr),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		log.Error(err)
	}
}

func (r *redisHandler) authorizedConn(conn redcon.Conn, cmd string) bool {
	if r.password == "" {
		return true
	}
	for _, allowedCmd := range noAuthCmd {
		if cmd == allowedCmd {
			return true
		}
	}
	return r.authenticatedAddr[conn.RemoteAddr()]
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
	Password     string
	MaxIdleConns int
	IdleTimeout  duration
}

// RedisConfig holds configuration for initializing redisHandler
type RedisConfig struct {
	Password    string
	DeleteOnGet bool
	DeleteOnSet bool
	Source      ClientConfig
	Destination ClientConfig
}

// NewRedisHandler returns new instance of redisHandler, a connection
// handler that handler redis-like interface
func NewRedisHandler(config RedisConfig) Handler {
	return &redisHandler{
		sourcePool:        newRedisPool(config.Source),
		destinationPool:   newRedisPool(config.Destination),
		deleteOnGet:       config.DeleteOnGet,
		deleteOnSet:       config.DeleteOnSet,
		deletedKey:        make(map[string]bool),
		authenticatedAddr: make(map[string]bool),
		password:          config.Password,
	}
}

func newRedisPool(config ClientConfig) *redis.Pool {
	options := make([]redis.DialOption, 0)
	if config.Password != "" {
		options = append(options, redis.DialPassword(config.Password))
	}

	return &redis.Pool{
		MaxIdle:     config.MaxIdleConns,
		IdleTimeout: config.IdleTimeout.Duration,
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", config.Addr, options...)
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

func logCmd(cmdArgs [][]byte) []string {
	cmd := make([]string, len(cmdArgs))
	for i, arg := range cmdArgs {
		cmd[i] = string(arg)
	}
	return cmd
}
