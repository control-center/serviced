package proxy

import (
	"github.com/zenoss/glog"

	"net"
	"testing"
	"time"
)


func stringAcceptor(listener net.Listener) (chan string) {
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
	proxy, err := NewProxy("foo", 0, false, local)
	if err != nil {
		t.Fatalf("Could not create a proxy: %s", err)
	}
	addresses := []string{remote.Addr().String()}
	proxy.SetNewAddresses(addresses)
	stringChan := stringAcceptor(remote)
	conn, err := net.Dial("tcp4", local.Addr().String())
	if err != nil {
		t.Fatalf("Could not create a connection to the proxyport: %s", err)
	}
	msg := "foo"
	n, err := conn.Write([]byte(msg))
	if err != nil {
		t.Fatalf("Failed to write msg to proxy: %s", err)
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
