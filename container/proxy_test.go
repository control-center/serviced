// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package container

import (
	"github.com/zenoss/glog"

	"net"
	"testing"
	"time"
)

func stringAcceptor(listener net.Listener) chan string {
	stringChan := make(chan string)
	go func() {
		defer close(stringChan)
		for {
			conn, err := listener.Accept()
			if err != nil {
				glog.Errorf("unexpected error: %s", err)
				return
			}
			buffer := make([]byte, 1000)
			n, err := conn.Read(buffer)
			if err != nil {
				glog.Errorf("problem reading from socket: %s", err)
			}
			stringChan <- string(buffer[:n])
		}
	}()
	return stringChan
}

func TestNoMux(t *testing.T) {

	remote, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Could not bind to a port for test")
	}
	local, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Could not bind to a port for test")
	}
	prxy, err := newProxy("foo", 0, false, local)
	if err != nil {
		t.Fatalf("Could not create a prxy: %s", err)
	}
	addresses := []string{remote.Addr().String()}
	prxy.SetNewAddresses(addresses)
	stringChan := stringAcceptor(remote)
	conn, err := net.Dial("tcp4", local.Addr().String())
	if err != nil {
		t.Fatalf("Could not create a connection to the prxyport: %s", err)
	}
	msg := "foo"
	n, err := conn.Write([]byte(msg))
	if err != nil {
		t.Fatalf("Failed to write msg to prxy: %s", err)
	}
	if n != len(msg) {
		t.Fatalf("Expected %d bytes to be written, only %d were written", len(msg), n)
	}
	timeout := make(chan bool, 1)
	go func() {
		time.Sleep(1 * time.Second)
		timeout <- true
	}()

	select {
	case stringResponse := <-stringChan:
		if stringResponse != msg {
			t.Fatalf("response did not equal msg: '%v' != '%v'", stringResponse, msg)
		}
	case <-timeout:
		t.Fatalf("Timed out reading response from test port")
	}
}
