package handler

import (
	"fmt"
	"io"
	"net"
	"testing"

	"github.com/gomodule/redigo/redis"
	"github.com/rafaeljusto/redigomock"
	"github.com/stretchr/testify/assert"
)

func Test_redisHandler(t *testing.T) {
	t.Run(`[Given] a password is set in the configuration
			[When] a non-AUTH command is received
			 [And] the connection bearing the command is not authenticated
			[Then] returns error stating the connection requires authentication`, func(t *testing.T) {

		handler, _, _ := initHandlerMock()
		handler.password = "justapass"

		rawCmd := "*1\r\n$4\r\nPING\r\n"
		rawErr := "-NOAUTH Authentication required.\r\n"

		fatal := make(chan error)
		signal := make(chan error)
		s := NewServer(":0", handler)
		go func() {
			defer s.Close()

			if err := s.ListenServeAndSignal(signal); err != nil {
				fatal <- err
			}
		}()

		done := make(chan bool)
		go func() {
			defer func() {
				done <- true
			}()

			err := <-signal
			if err != nil {
				fatal <- err
			}

			reply, err := doRequest(s.Addr().String(), rawCmd)
			if err != nil {
				fatal <- err
			}

			assert.Equal(t, rawErr, reply, "reply should be a \"No Auth\" error")
		}()

		waitForComplete(t, done, fatal)
	})
}

func Test_redisHandler_HandleGET(t *testing.T) {

	var (
		key, value = "mykey", "hello"
		errorMsg   = "Unexpected error"
		rawMessage = fmt.Sprintf("*2\r\n$3\r\nGET\r\n$%d\r\n%s\r\n", len(key), key)
		rawValue   = fmt.Sprintf("$%d\r\n%s\r\n", len(value), value)
		rawNil     = "$-1\r\n"
		rawError   = fmt.Sprintf("-%s\r\n", errorMsg)
	)

	t.Run(`[Given] a key is available in "destination" 
		    [When] a GET request for the key is received
		    [Then] GET and return the key value from "destination"`, func(t *testing.T) {

		handler, _, dstMock := initHandlerMock()

		dstGET := dstMock.Command("GET", []byte(key)).Expect(value)

		fatal := make(chan error)
		signal := make(chan error)
		s := NewServer(":0", handler)
		go func() {
			defer s.Close()

			if err := s.ListenServeAndSignal(signal); err != nil {
				fatal <- err
			}
		}()

		done := make(chan bool)
		go func() {
			defer func() {
				done <- true
			}()

			err := <-signal
			if err != nil {
				fatal <- err
			}

			reply, err := doRequest(s.Addr().String(), rawMessage)
			if err != nil {
				fatal <- err
			}

			assert.Equal(t, rawValue, reply, "reply should be equal to value")
			assert.True(t, dstGET.Called, "destination redis should be called")
		}()

		waitForComplete(t, done, fatal)
	})

	t.Run(`[Given] a key is not available in "destination"
			 [And] "destination" return non-nil error
		    [When] a GET request for the key is received
		    [Then] return the error from "destination"`, func(t *testing.T) {

		handler, _, dstMock := initHandlerMock()

		dstGET := dstMock.Command("GET", []byte(key)).ExpectError(fmt.Errorf(errorMsg))

		fatal := make(chan error)
		signal := make(chan error)
		s := NewServer(":0", handler)
		go func() {
			defer s.Close()

			if err := s.ListenServeAndSignal(signal); err != nil {
				fatal <- err
			}
		}()

		done := make(chan bool)
		go func() {
			defer func() {
				done <- true
			}()

			err := <-signal
			if err != nil {
				fatal <- err
			}

			reply, err := doRequest(s.Addr().String(), rawMessage)
			if err != nil {
				fatal <- err
			}

			assert.Equal(t, rawError, reply, "reply should be equal to error message")
			assert.True(t, dstGET.Called, "destination redis should be called")
		}()

		waitForComplete(t, done, fatal)
	})

	t.Run(`[Given] a key is not available in "destination"
			 [And] "source" return non-nil error
		    [When] a GET request for the key is received
		    [Then] return the error from "source"`, func(t *testing.T) {

		handler, srcMock, dstMock := initHandlerMock()

		dstGET := dstMock.Command("GET", []byte(key)).ExpectError(nil)
		srcGET := srcMock.Command("GET", []byte(key)).ExpectError(fmt.Errorf(errorMsg))

		fatal := make(chan error)
		signal := make(chan error)
		s := NewServer(":0", handler)
		go func() {
			defer s.Close()

			if err := s.ListenServeAndSignal(signal); err != nil {
				fatal <- err
			}
		}()

		done := make(chan bool)
		go func() {
			defer func() {
				done <- true
			}()

			err := <-signal
			if err != nil {
				fatal <- err
			}

			reply, err := doRequest(s.Addr().String(), rawMessage)
			if err != nil {
				fatal <- err
			}

			assert.Equal(t, rawError, reply, "reply should be equal to error message")
			assert.True(t, dstGET.Called, "destination redis should be called")
			assert.True(t, srcGET.Called, "source redis should be called")
		}()

		waitForComplete(t, done, fatal)
	})

	t.Run(`[Given] a key is not available in "destination"
			 [And] the key is available in "source"
			 [And] deleteOnGet set to false
		    [When] a GET request for the key is received
		    [Then] GET and return the key value from "source"
		     [And] SET the value with the key to "destination"`, func(t *testing.T) {

		handler, srcMock, dstMock := initHandlerMock()
		handler.deleteOnGet = false

		dstGET := dstMock.Command("GET", []byte(key)).Expect(nil)
		dstSET := dstMock.Command("SET", []byte(key), value).Expect("OK")
		srcGET := srcMock.Command("GET", []byte(key)).Expect(value)

		fatal := make(chan error)
		signal := make(chan error)
		s := NewServer(":0", handler)
		go func() {
			defer s.Close()

			if err := s.ListenServeAndSignal(signal); err != nil {
				fatal <- err
			}
		}()

		done := make(chan bool)
		go func() {
			defer func() {
				done <- true
			}()

			err := <-signal
			if err != nil {
				fatal <- err
			}

			reply, err := doRequest(s.Addr().String(), rawMessage)
			if err != nil {
				fatal <- err
			}

			assert.Equal(t, rawValue, reply, "reply should be equal to value")
			assert.True(t, dstGET.Called, "destination redis GET command should be called")
			assert.True(t, dstSET.Called, "destination redis SET command should be called")
			assert.True(t, srcGET.Called, "source redis GET command should be called")
		}()

		waitForComplete(t, done, fatal)
	})

	t.Run(`[Given] a key is not available in "destination"
			 [And] the key is available in "source"
			 [And] deleteOnGet set to true
		    [When] a GET request for the key is received
		    [Then] GET and return the key value from "source"
			 [And] SET the value with the key to "destination"
			 [And] DELETE the key from "source"`, func(t *testing.T) {

		handler, srcMock, dstMock := initHandlerMock()
		handler.deleteOnGet = true

		dstGET := dstMock.Command("GET", []byte(key)).Expect(nil)
		dstSET := dstMock.Command("SET", []byte(key), value).Expect("OK")
		srcGET := srcMock.Command("GET", []byte(key)).Expect(value)
		srcDEL := srcMock.Command("DEL", []byte(key)).Expect(int64(1))

		fatal := make(chan error)
		signal := make(chan error)
		s := NewServer(":0", handler)
		go func() {
			defer s.Close()

			if err := s.ListenServeAndSignal(signal); err != nil {
				fatal <- err
			}
		}()

		done := make(chan bool)
		go func() {
			defer func() {
				done <- true
			}()

			err := <-signal
			if err != nil {
				fatal <- err
			}

			reply, err := doRequest(s.Addr().String(), rawMessage)
			if err != nil {
				fatal <- err
			}

			assert.Equal(t, rawValue, reply, "reply should be equal to value")
			assert.True(t, dstGET.Called, "destination redis GET command should be called")
			assert.True(t, dstSET.Called, "destination redis SET command should be called")
			assert.True(t, srcGET.Called, "source redis GET command should be called")
			assert.True(t, srcDEL.Called, "source redis DEL command should be called")
		}()

		waitForComplete(t, done, fatal)
	})

	t.Run(`[Given] a key is not available in "destination"
			 [And] the key is not available in "source"
		    [When] a GET request for the key is received
		    [Then] return nil rawMessage from "source"`, func(t *testing.T) {

		handler, srcMock, dstMock := initHandlerMock()

		dstGET := dstMock.Command("GET", []byte(key)).Expect(nil)
		srcGET := srcMock.Command("GET", []byte(key)).Expect(nil)

		fatal := make(chan error)
		signal := make(chan error)
		s := NewServer(":0", handler)
		go func() {
			defer s.Close()

			if err := s.ListenServeAndSignal(signal); err != nil {
				fatal <- err
			}
		}()

		done := make(chan bool)
		go func() {
			defer func() {
				done <- true
			}()

			err := <-signal
			if err != nil {
				fatal <- err
			}

			reply, err := doRequest(s.Addr().String(), rawMessage)
			if err != nil {
				fatal <- err
			}

			assert.Equal(t, rawNil, reply, "reply should be equal to nil")
			assert.True(t, dstGET.Called, "destination redis GET command should be called")
			assert.True(t, srcGET.Called, "source redis GET command should be called")
		}()

		waitForComplete(t, done, fatal)
	})
}

func Test_redisHandler_HandleSET(t *testing.T) {

	var (
		key, value = "mykey", "hello"
		errorMsg   = "Unexpected error"
		rawMessage = fmt.Sprintf("*3\r\n$3\r\nSET\r\n$%d\r\n%s\r\n$%d\r\n%s\r\n", len(key), key, len(value), value)
		rawOK      = "+OK\r\n"
		rawError   = fmt.Sprintf("-%s\r\n", errorMsg)
	)

	t.Run(`[Given] deleteOnSet set to false
			[When] a SET request for a key is received
			[Then] SET the key with the value to "destination"`, func(t *testing.T) {

		handler, _, dstMock := initHandlerMock()
		handler.deleteOnSet = false

		dstSET := dstMock.Command("SET", []byte(key), []byte(value)).Expect("OK")

		fatal := make(chan error)
		signal := make(chan error)
		s := NewServer(":0", handler)
		go func() {
			defer s.Close()

			if err := s.ListenServeAndSignal(signal); err != nil {
				fatal <- err
			}
		}()

		done := make(chan bool)
		go func() {
			defer func() {
				done <- true
			}()

			err := <-signal
			if err != nil {
				fatal <- err
			}

			reply, err := doRequest(s.Addr().String(), rawMessage)
			if err != nil {
				fatal <- err
			}

			assert.Equal(t, rawOK, reply, "reply should be \"OK\"")
			assert.True(t, dstSET.Called, "destination redis should be called")
		}()

		waitForComplete(t, done, fatal)
	})

	t.Run(`[Given] deleteOnSet set to false
			 [And] "destination" returns error
			[When] a SET request for a key is received
			[Then] returns the error message`, func(t *testing.T) {

		handler, _, dstMock := initHandlerMock()
		handler.deleteOnSet = false

		dstSET := dstMock.Command("SET", []byte(key), []byte(value)).ExpectError(fmt.Errorf(errorMsg))

		fatal := make(chan error)
		signal := make(chan error)
		s := NewServer(":0", handler)
		go func() {
			defer s.Close()

			if err := s.ListenServeAndSignal(signal); err != nil {
				fatal <- err
			}
		}()

		done := make(chan bool)
		go func() {
			defer func() {
				done <- true
			}()

			err := <-signal
			if err != nil {
				fatal <- err
			}

			reply, err := doRequest(s.Addr().String(), rawMessage)
			if err != nil {
				fatal <- err
			}

			assert.Equal(t, rawError, reply, "reply should be equal to error message")
			assert.True(t, dstSET.Called, "destination redis should be called")
		}()

		waitForComplete(t, done, fatal)
	})

	t.Run(`[Given] deleteOnSet set to true
			[When] a SET request for a key is received
			[Then] SET the key with the value to "destination"
			 [And] DELETE the key from "source"`, func(t *testing.T) {

		handler, srcMock, dstMock := initHandlerMock()
		handler.deleteOnSet = true

		dstSET := dstMock.Command("SET", []byte(key), []byte(value)).Expect("OK")
		srcDEL := srcMock.Command("DEL", []byte(key)).Expect(int64(1))

		fatal := make(chan error)
		signal := make(chan error)
		s := NewServer(":0", handler)
		go func() {
			defer s.Close()

			if err := s.ListenServeAndSignal(signal); err != nil {
				fatal <- err
			}
		}()

		done := make(chan bool)
		go func() {
			defer func() {
				done <- true
			}()

			err := <-signal
			if err != nil {
				fatal <- err
			}

			reply, err := doRequest(s.Addr().String(), rawMessage)
			if err != nil {
				fatal <- err
			}

			assert.Equal(t, rawOK, reply, "reply should be \"OK\"")
			assert.True(t, dstSET.Called, "destination redis should be called")
			assert.True(t, srcDEL.Called, "source redis DEL command should be called")
		}()

		waitForComplete(t, done, fatal)
	})

	t.Run(`[Given] deleteOnSet set to true
			 [And] the key has been deleted
			[When] a SET request for a key is received
			[Then] SET the key with the value to "destination"
			 [And] Don't DELETE the key from "source"`, func(t *testing.T) {

		handler, srcMock, dstMock := initHandlerMock()
		handler.deleteOnSet = true
		handler.deletedKey[key] = true

		dstSET := dstMock.Command("SET", []byte(key), []byte(value)).Expect("OK")
		srcDEL := srcMock.Command("DEL", []byte(key)).Expect(int64(1))

		fatal := make(chan error)
		signal := make(chan error)
		s := NewServer(":0", handler)
		go func() {
			defer s.Close()

			if err := s.ListenServeAndSignal(signal); err != nil {
				fatal <- err
			}
		}()

		done := make(chan bool)
		go func() {
			defer func() {
				done <- true
			}()

			err := <-signal
			if err != nil {
				fatal <- err
			}

			reply, err := doRequest(s.Addr().String(), rawMessage)
			if err != nil {
				fatal <- err
			}

			assert.Equal(t, rawOK, reply, "reply should be \"OK\"")
			assert.True(t, dstSET.Called, "destination redis should be called")
			assert.False(t, srcDEL.Called, "source redis DEL command should not be called")
		}()

		waitForComplete(t, done, fatal)
	})
}

func Test_redisHandler_HandlePING(t *testing.T) {

	var (
		rawPing = "*1\r\n$4\r\nPING\r\n"
		rawPong = "+PONG\r\n"
	)

	t.Run(`[When] a PING request is received
		   [Then] return PONG`, func(t *testing.T) {

		handler, _, _ := initHandlerMock()

		fatal := make(chan error)
		signal := make(chan error)
		s := NewServer(":0", handler)
		go func() {
			defer s.Close()

			if err := s.ListenServeAndSignal(signal); err != nil {
				fatal <- err
			}
		}()

		done := make(chan bool)
		go func() {
			defer func() {
				done <- true
			}()

			err := <-signal
			if err != nil {
				fatal <- err
			}

			reply, err := doRequest(s.Addr().String(), rawPing)
			if err != nil {
				fatal <- err
			}

			assert.Equal(t, rawPong, reply, "reply should be \"PONG\"")
		}()

		waitForComplete(t, done, fatal)
	})
}

func Test_redisHandler_HandleAUTH(t *testing.T) {
	handlerPass := "justapass"

	t.Run(`[Given] a Password is set in configuration
			[When] an AUTH command is received
			 [And] the password argument matches with the one set in config
			[Then] returns OK
			 [And] authenticate the connection`, func(t *testing.T) {

		handler, _, _ := initHandlerMock()
		handler.password = handlerPass

		var (
			rawAuth = fmt.Sprintf("*2\r\n$4\r\nAUTH\r\n$%d\r\n%s\r\n", len(handler.password), handler.password)
			rawOK   = "+OK\r\n"
		)

		fatal := make(chan error)
		signal := make(chan error)
		s := NewServer(":0", handler)
		go func() {
			defer s.Close()

			if err := s.ListenServeAndSignal(signal); err != nil {
				fatal <- err
			}
		}()

		done := make(chan bool)
		go func() {
			defer func() {
				done <- true
			}()

			err := <-signal
			if err != nil {
				fatal <- err
			}

			reply, err := doRequest(s.Addr().String(), rawAuth)
			if err != nil {
				fatal <- err
			}

			assert.Equal(t, rawOK, reply, "reply should be \"OK\"")
		}()

		waitForComplete(t, done, fatal)
	})

	t.Run(`[Given] a Password is set in configuration
			[When] an AUTH command is received
			 [And] the password argument doesn't match with the one set in config
			[Then] returns error stating invalid password`, func(t *testing.T) {

		handler, _, _ := initHandlerMock()
		handler.password = handlerPass
		passArgs := "wrongpass"

		var (
			rawAuth = fmt.Sprintf("*2\r\n$4\r\nAUTH\r\n$%d\r\n%s\r\n", len(passArgs), passArgs)
			rawErr  = "-ERR invalid password\r\n"
		)

		fatal := make(chan error)
		signal := make(chan error)
		s := NewServer(":0", handler)
		go func() {
			defer s.Close()

			if err := s.ListenServeAndSignal(signal); err != nil {
				fatal <- err
			}
		}()

		done := make(chan bool)
		go func() {
			defer func() {
				done <- true
			}()

			err := <-signal
			if err != nil {
				fatal <- err
			}

			reply, err := doRequest(s.Addr().String(), rawAuth)
			if err != nil {
				fatal <- err
			}

			assert.Equal(t, rawErr, reply, "reply should be an \"Invalid password\" error")
		}()

		waitForComplete(t, done, fatal)
	})

	t.Run(`[Given] a password is not set in the configuration
			[When] an AUTH command is received
			[Then] returns error stating that password is not set`, func(t *testing.T) {

		handler, _, _ := initHandlerMock()
		handler.password = ""

		passArgs := "nonexistent"
		rawAuth := fmt.Sprintf("*2\r\n$4\r\nAUTH\r\n$%d\r\n%s\r\n", len(passArgs), passArgs)
		rawErr := "-ERR Client sent AUTH, but no password is set\r\n"

		fatal := make(chan error)
		signal := make(chan error)
		s := NewServer(":0", handler)
		go func() {
			defer s.Close()

			if err := s.ListenServeAndSignal(signal); err != nil {
				fatal <- err
			}
		}()

		done := make(chan bool)
		go func() {
			defer func() {
				done <- true
			}()

			err := <-signal
			if err != nil {
				fatal <- err
			}

			reply, err := doRequest(s.Addr().String(), rawAuth)
			if err != nil {
				fatal <- err
			}

			assert.Equal(t, rawErr, reply, "reply should be an \"No password set\" error")
		}()

		waitForComplete(t, done, fatal)
	})

	t.Run(`[When] an incorrect AUTH command is received (wrong number of args)
		   [Then] returns error stating that the number of args is wrong`, func(t *testing.T) {

		handler, _, _ := initHandlerMock()
		handler.password = handlerPass

		rawAuth := fmt.Sprintf("*1\r\n$4\r\nAUTH\r\n")
		rawErr := "-ERR wrong number of arguments for 'auth' command\r\n"

		fatal := make(chan error)
		signal := make(chan error)
		s := NewServer(":0", handler)
		go func() {
			defer s.Close()

			if err := s.ListenServeAndSignal(signal); err != nil {
				fatal <- err
			}
		}()

		done := make(chan bool)
		go func() {
			defer func() {
				done <- true
			}()

			err := <-signal
			if err != nil {
				fatal <- err
			}

			reply, err := doRequest(s.Addr().String(), rawAuth)
			if err != nil {
				fatal <- err
			}

			assert.Equal(t, rawErr, reply, "reply should be an \"Wrong number of args\" error")
		}()

		waitForComplete(t, done, fatal)
	})
}

func Test_redisHandler_HandleDefault(t *testing.T) {

	var tc = []struct {
		rawMsg   string
		cmd      string
		args     [][]byte
		reply    interface{}
		replyRaw string
	}{
		{"*2\r\n$4\r\nECHO\r\n$5\r\nHello\r\n", "ECHO", [][]byte{[]byte("Hello")}, "Hello", "+Hello\r\n"},
		{"*3\r\n$4\r\nHGET\r\n$6\r\nmyhash\r\n$5\r\nfield\r\n", "HGET", [][]byte{[]byte("myhash"), []byte("field")}, nil, "$-1\r\n"},
		{"*1\r\n$4\r\nHSET\r\n", "HSET", [][]byte{}, fmt.Errorf("Wrong number of args"), "-Wrong number of args\r\n"},
		{"*2\r\n$3\r\nTTL\r\n$5\r\nmykey\r\n", "TTL", [][]byte{[]byte("mykey")}, 10, ":10\r\n"},
		{"*1\r\n$7\r\nCOMMAND\r\n", "COMMAND", [][]byte{}, []interface{}{[]byte("GET"), []byte("SET")}, "*2\r\n$3\r\nGET\r\n$3\r\nSET\r\n"},
	}

	t.Run(`[When] any request except GET, SET, and PING is received
		   [Then] forward the request to "destination"
		    [And] return the response`, func(t *testing.T) {

		for _, tt := range tc {
			handler, _, dstMock := initHandlerMock()
			dstCMD := dstMock.Command(tt.cmd, toInterfaceSlice(tt.args)...).Expect(tt.reply)

			fatal := make(chan error)
			signal := make(chan error)
			s := NewServer(":0", handler)
			go func() {
				defer s.Close()

				if err := s.ListenServeAndSignal(signal); err != nil {
					fatal <- err
				}
			}()

			done := make(chan bool)
			go func() {
				defer func() {
					done <- true
				}()

				err := <-signal
				if err != nil {
					fatal <- err
				}

				reply, err := doRequest(s.Addr().String(), tt.rawMsg)
				if err != nil {
					fatal <- err
				}

				assert.Equal(t, tt.replyRaw, reply, "reply should be equal to expectation")
				assert.True(t, dstCMD.Called, "destination redis should be called")
			}()

			waitForComplete(t, done, fatal)
		}
	})
}

func initHandlerMock() (handler *redisHandler, srcMock, dstMock *redigomock.Conn) {
	srcMock = redigomock.NewConn()
	dstMock = redigomock.NewConn()

	var config = RedisConfig{}
	handler = NewRedisHandler(config).(*redisHandler)

	handler.sourcePool = &redis.Pool{
		Dial: func() (redis.Conn, error) {
			return srcMock, nil
		},
	}
	handler.destinationPool = &redis.Pool{
		Dial: func() (redis.Conn, error) {
			return dstMock, nil
		},
	}

	return
}

func doRequest(addr, msg string) (reply string, err error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return
	}
	defer conn.Close()

	_, err = io.WriteString(conn, msg)
	if err != nil {
		return
	}

	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		return
	}

	reply = string(buf[:n])
	return
}

func waitForComplete(t *testing.T, done chan bool, fatal chan error) {
	select {
	case <-done:
		return
	case err := <-fatal:
		t.Fatal(err)
	}
}
