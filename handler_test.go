package remiro

import (
	"fmt"
	"io"
	"net"
	"testing"

	"github.com/gomodule/redigo/redis"
	"github.com/rafaeljusto/redigomock"
	"github.com/stretchr/testify/assert"
)

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

		signal := make(chan error)
		s := NewServer(":0", handler)
		go func() {
			defer s.Close()

			if err := s.ListenServeAndSignal(signal); err != nil {
				t.Fatal(err)
			}
		}()

		done := make(chan bool)
		go func() {
			defer func() {
				done <- true
			}()

			err := <-signal
			if err != nil {
				t.Fatal(err)
			}

			reply, err := doRequest(s.Addr().String(), rawMessage)
			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, rawValue, reply, "reply should be equal to value")
			assert.True(t, dstGET.Called, "destination redis should be called")
		}()

		<-done
	})

	t.Run(`[Given] a key is not available in "destination"
			 [And] "destination" return non-nil error
		    [When] a GET request for the key is received
		    [Then] return the error from "destination"`, func(t *testing.T) {

		handler, _, dstMock := initHandlerMock()

		dstGET := dstMock.Command("GET", []byte(key)).ExpectError(fmt.Errorf(errorMsg))

		signal := make(chan error)
		s := NewServer(":0", handler)
		go func() {
			defer s.Close()

			if err := s.ListenServeAndSignal(signal); err != nil {
				t.Fatal(err)
			}
		}()

		done := make(chan bool)
		go func() {
			defer func() {
				done <- true
			}()

			err := <-signal
			if err != nil {
				t.Fatal(err)
			}

			reply, err := doRequest(s.Addr().String(), rawMessage)
			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, rawError, reply, "reply should be equal to error message")
			assert.True(t, dstGET.Called, "destination redis should be called")
		}()

		<-done
	})

	t.Run(`[Given] a key is not available in "destination"
			 [And] "source" return non-nil error
		    [When] a GET request for the key is received
		    [Then] return the error from "source"`, func(t *testing.T) {

		handler, srcMock, dstMock := initHandlerMock()

		dstGET := dstMock.Command("GET", []byte(key)).ExpectError(nil)
		srcGET := srcMock.Command("GET", []byte(key)).ExpectError(fmt.Errorf(errorMsg))

		signal := make(chan error)
		s := NewServer(":0", handler)
		go func() {
			defer s.Close()

			if err := s.ListenServeAndSignal(signal); err != nil {
				t.Fatal(err)
			}
		}()

		done := make(chan bool)
		go func() {
			defer func() {
				done <- true
			}()

			err := <-signal
			if err != nil {
				t.Fatal(err)
			}

			reply, err := doRequest(s.Addr().String(), rawMessage)
			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, rawError, reply, "reply should be equal to error message")
			assert.True(t, dstGET.Called, "destination redis should be called")
			assert.True(t, srcGET.Called, "source redis should be called")
		}()

		<-done
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

		signal := make(chan error)
		s := NewServer(":0", handler)
		go func() {
			defer s.Close()

			if err := s.ListenServeAndSignal(signal); err != nil {
				t.Fatal(err)
			}
		}()

		done := make(chan bool)
		go func() {
			defer func() {
				done <- true
			}()

			err := <-signal
			if err != nil {
				t.Fatal(err)
			}

			reply, err := doRequest(s.Addr().String(), rawMessage)
			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, rawValue, reply, "reply should be equal to value")
			assert.True(t, dstGET.Called, "destination redis GET command should be called")
			assert.True(t, dstSET.Called, "destination redis SET command should be called")
			assert.True(t, srcGET.Called, "source redis GET command should be called")
		}()

		<-done
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
		srcDEL := srcMock.Command("DEL", []byte(key)).Expect(1)

		signal := make(chan error)
		s := NewServer(":0", handler)
		go func() {
			defer s.Close()

			if err := s.ListenServeAndSignal(signal); err != nil {
				t.Fatal(err)
			}
		}()

		done := make(chan bool)
		go func() {
			defer func() {
				done <- true
			}()

			err := <-signal
			if err != nil {
				t.Fatal(err)
			}

			reply, err := doRequest(s.Addr().String(), rawMessage)
			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, rawValue, reply, "reply should be equal to value")
			assert.True(t, dstGET.Called, "destination redis GET command should be called")
			assert.True(t, dstSET.Called, "destination redis SET command should be called")
			assert.True(t, srcGET.Called, "source redis GET command should be called")
			assert.True(t, srcDEL.Called, "source redis DEL command should be called")
		}()

		<-done
	})

	t.Run(`[Given] a key is not available in "destination"
			 [And] the key is not available in "source"
		    [When] a GET request for the key is received
		    [Then] return nil rawMessage from "source"`, func(t *testing.T) {

		handler, srcMock, dstMock := initHandlerMock()

		dstGET := dstMock.Command("GET", []byte(key)).Expect(nil)
		srcGET := srcMock.Command("GET", []byte(key)).Expect(nil)

		signal := make(chan error)
		s := NewServer(":0", handler)
		go func() {
			defer s.Close()

			if err := s.ListenServeAndSignal(signal); err != nil {
				t.Fatal(err)
			}
		}()

		done := make(chan bool)
		go func() {
			defer func() {
				done <- true
			}()

			err := <-signal
			if err != nil {
				t.Fatal(err)
			}

			reply, err := doRequest(s.Addr().String(), rawMessage)
			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, rawNil, reply, "reply should be equal to nil")
			assert.True(t, dstGET.Called, "destination redis GET command should be called")
			assert.True(t, srcGET.Called, "source redis GET command should be called")
		}()

		<-done
	})
}

func Test_redisHandler_HandleSET(t *testing.T) {

	var (
		key, value = "mykey", "hello"
		rawMessage = fmt.Sprintf("*3\r\n$3\r\nSET\r\n$%d\r\n%s\r\n$%d\r\n%s\r\n", len(key), key, len(value), value)
		rawOK      = "+OK\r\n"
	)

	t.Run(`[Given] deleteOnSet set to false
			[When] a SET request for a key is received
			[Then] SET the key with the value to "destination"`, func(t *testing.T) {

		handler, _, dstMock := initHandlerMock()

		dstSET := dstMock.Command("SET", []byte(key), []byte(value)).Expect("OK")

		signal := make(chan error)
		s := NewServer(":0", handler)
		go func() {
			defer s.Close()

			if err := s.ListenServeAndSignal(signal); err != nil {
				t.Fatal(err)
			}
		}()

		done := make(chan bool)
		go func() {
			defer func() {
				done <- true
			}()

			err := <-signal
			if err != nil {
				t.Fatal(err)
			}

			reply, err := doRequest(s.Addr().String(), rawMessage)
			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, rawOK, reply, "reply should be \"OK\"")
			assert.True(t, dstSET.Called, "destination redis should be called")
		}()

		<-done
	})

	t.Run(`[Given] deleteOnSet set to true
			[When] a SET request for a key is received
			[Then] SET the key with the value to "destination"
			 [And] DELETE the key from "source"`, func(t *testing.T) {

	})
}

func Test_redisHandler_HandlePING(t *testing.T) {

	t.Run(`[When] a PING request is received
		   [Then] return PONG`, func(t *testing.T) {

	})
}

func Test_redisHandler_HandleDefault(t *testing.T) {

	t.Run(`[When] any request except GET, SET, and PING is received
		   [Then] forward the request to "destination"
		    [And] return the response`, func(t *testing.T) {

	})
}

func initHandlerMock() (handler *redisHandler, srcMock, dstMock *redigomock.Conn) {
	srcMock = redigomock.NewConn()
	dstMock = redigomock.NewConn()
	handler = &redisHandler{
		sourcePool: &redis.Pool{
			Dial: func() (redis.Conn, error) {
				return srcMock, nil
			},
		},
		destinationPool: &redis.Pool{
			Dial: func() (redis.Conn, error) {
				return dstMock, nil
			},
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
