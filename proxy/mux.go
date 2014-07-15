// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package proxy

import (
	"github.com/zenoss/glog"

	"bufio"
	"fmt"
	"io"
	"net"
	"strings"
	"time"
)

// TCPMux is an implementation of tcp muxing RFC 1078.
type TCPMux struct {
	listener    net.Listener    // the connection this mux listens on
	connections chan net.Conn   // stream of accepted connections
	closing     chan chan error // shutdown noticiation
}

// NewTCPMux creates a new tcp mux with the given listener. If it succees, it
// is expected that this object is the owner of the listener and will close it
// when Close() is called on the TCPMux.
func NewTCPMux(listener net.Listener) (mux *TCPMux, err error) {
	if listener == nil {
		return nil, fmt.Errorf("listener can not be nil")
	}
	mux = &TCPMux{
		listener:    listener,
		connections: make(chan net.Conn),
		closing:     make(chan chan error),
	}
	go mux.loop()
	return mux, nil
}

func (mux *TCPMux) Close() {
	glog.V(5).Info("Close Called")
	close(mux.closing)
}

func (mux *TCPMux) acceptor(listener net.Listener, closing chan chan struct{}) {
	defer func() {
		close(mux.connections)
	}()
	for {
		conn, err := mux.listener.Accept()
		if err != nil {
			if strings.Contains(err.Error(), "too many open files") {
				glog.Warningf("error accepting connections, retrying in 50 ms: %s", err)
				select {
				case <-closing:
					glog.V(5).Info("shutting down acceptor")
					return
				case <-time.After(time.Millisecond * 50):
					continue
				}
			}
			glog.Errorf("shutting down acceptor: %s", err)
			return
		}
		glog.V(5).Infof("accepted connection: %s", conn)
		select {
		case <-closing:
			glog.V(5).Info("shutting down acceptor")
			conn.Close()
			return
		case mux.connections <- conn:
		}
	}
}

func (mux *TCPMux) loop() {
	glog.V(5).Infof("entering TPCMux loop")
	closeAcceptor := make(chan chan struct{})
	go mux.acceptor(mux.listener, closeAcceptor)
	for {
		select {
		case errc := <-mux.closing:
			glog.V(5).Info("Closing mux")
			closeAcceptorAck := make(chan struct{})
			mux.listener.Close()
			closeAcceptor <- closeAcceptorAck
			errc <- nil
			return
		case conn, ok := <-mux.connections:
			if !ok {
				mux.connections = nil
				glog.V(6).Info("got nil conn, channel is closed")
				continue
			}
			glog.V(5).Info("handing mux connection")
			go mux.muxConnection(conn)
		}
	}
}

// muxConnection takes an inbound connection reads a line from it and
// then attempts to set up a connection to the service specified by the
// line. The service is specified in the form "IP:PORT\n". If the connection
// to the service is sucessful, all traffic continues to be proxied between
// two connections.
func (mux *TCPMux) muxConnection(conn net.Conn) {
	// make sure that we don't block indefinitely
	conn.SetReadDeadline(time.Now().Add(time.Second * 5))
	reader := bufio.NewReader(conn)
	line, err := reader.ReadString('\n')
	if err != nil {
		glog.Errorf("could not read mux line: %s", err)
		conn.Close()
		return
	}
	// restore deadline
	conn.SetReadDeadline(time.Time{})
	line = strings.TrimSpace(line)

	svc, err := net.Dial("tcp4", line)
	if err != nil {
		glog.Errorf("got %s => %s, could not dial to '%s' : %s", conn.LocalAddr(), conn.RemoteAddr(), line, err)
		conn.Close()
		return
	}
	// write any pending buffered data that wasn't part of the service spec
	if reader.Buffered() > 0 {
		bufferedBytes, err := reader.Peek(reader.Buffered())
		if err != nil {
			glog.Errorf("error peaking at buffered bytes: %s", err)
		}
		n, err := conn.Write(bufferedBytes)
		if err != nil {
			glog.Errorf("error writting buffered bytes: %s", err)
		}
		if n != len(bufferedBytes) {
			glog.Errorf("exepected to write %d bytes but wrote %s", len(bufferedBytes), n)
		}
	}

	go func() {
		io.Copy(conn, svc)
		conn.Close()
		svc.Close()
	}()
	go func() {
		io.Copy(svc, conn)
		conn.Close()
		svc.Close()
	}()
}
