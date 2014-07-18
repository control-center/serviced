// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package proxy

import (
	"fmt"
	"io"
	"net"
	"strings"
	"time"
	"testing"
)

type echoListener struct {
	t        *testing.T
	listener net.Listener
	closing  chan chan error
}

func newEchoListener(t *testing.T) *echoListener {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("could not listen: %s", err)
	}

	e := &echoListener{
		t:        t,
		listener: listener,
		closing:  make(chan chan error),
	}
	go e.loop()
	return e
}

func listenerToPort(listener net.Listener) string {
	parts := strings.Split(listener.Addr().String(), ":")
	return parts[len(parts)-1]
}

func connectToListener(listener net.Listener) (net.Conn, error) {
	port := listenerToPort(listener)
	return net.Dial("tcp", fmt.Sprintf("127.0.0.1:%s", port))
}

func (e *echoListener) Connect() (net.Conn) {
	conn, err := connectToListener(e.listener)
	if err != nil {
		e.t.Fatalf("could not connect to echo server: %s", err)
	}
	return conn
}

func (e *echoListener) Close() error {
	e.listener.Close()
	errc := make(chan error)
	e.closing <- errc
	return <-errc
}

func (e *echoListener) loop() {
	for {
		conn, err := e.listener.Accept()
		select {
		case errc := <-e.closing:
			errc <- err
		default:
			if err != nil {
				if strings.Contains(err.Error(), "use of closed network connection") {
					time.Sleep(time.Millisecond * 100)
					continue
				}
				e.t.Logf("err on connection: %s", err)
				if conn != nil {
					conn.Close()
				}
				continue
			}
			// echo handler
			go func(c net.Conn) {
				io.Copy(c, c)
				c.Close()
				c.Close()
			}(conn)
		}
	}
}

// testConnect returns a connection to the mux for unit tests
func (mux *TCPMux) testConnect(t *testing.T) net.Conn {
	conn, err := connectToListener(mux.listener)
	if err != nil {
		t.Fatalf("Could not connect to mux: %s", err)
	}
	return conn
}

func TestTCPMux(t *testing.T) {
	target := newEchoListener(t)
	defer target.Close()

	muxEndpoint, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("could not create tcpmux endpoint: %s", err)
	}
	mux, err := NewTCPMux(muxEndpoint)
	if err != nil {
		t.Fatalf("did not expect failure creating TCPMux: %s", err)
	}

	testMsg := "\nhello\n"

	conn := mux.testConnect(t)
	header := fmt.Sprintf("127.0.0.1:%s\n", listenerToPort(target.listener))
	conn.Write([]byte(header))
	conn.Write([]byte(testMsg))
	buffer := make([]byte, 4096)
	n, err := conn.Read(buffer)
	t.Logf("got %d bytes back", n)
	if n <= 0 {
		t.Fatalf("expected something")
	}
	returnedValue := string(buffer[0:n])
	if returnedValue != testMsg {
		t.Fatalf("got back %+v expected %+v", returnedValue, testMsg)
	}
	conn.Close()

}
