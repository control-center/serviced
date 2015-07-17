// Copyright 2014 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// +build unit

package container

import (
	"strings"

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
	t.Skip("Not having a mux isn't currently supported")

	remote, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Could not bind to a port for test")
	}
	local, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Could not bind to a port for test")
	}
	prxy, err := newProxy("foo", "endpointfoo", 0, false, local, false)
	if err != nil {
		t.Fatalf("Could not create a prxy: %s", err)
	}
	host := strings.Split(remote.Addr().String(), ":")[0]
	addresses := []addressTuple{addressTuple{host, remote.Addr().String()}}
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
