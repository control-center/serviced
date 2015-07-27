package rpcutils

import (
	"fmt"
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
	"testing"
	"time"
)

type RPCTestType int

func (rtt *RPCTestType) Sleep(seconds *int, reply *int) error {
	time.Sleep(time.Duration(*seconds) * time.Second)
	return nil
}

func TestTimeout(t *testing.T) {
	rtt := new(RPCTestType)
	rpc.Register(rtt)
	rpc.HandleHTTP()
	listener, err := net.Listen("tcp", ":32111")
	if err != nil {
		t.Errorf("listen error: %s", err)
	}
	go func() {
		for {
			conn, err := listener.Accept()
			defer conn.Close()
			if err != nil {
				t.Errorf("Error accepting connections: %s", err)
			}
			go rpc.ServeCodec(jsonrpc.NewServerCodec(conn))
		}
	}()

	client, err := NewReconnectingClient("localhost:32111")

	var reply, seconds int

	// Sleep for one second, timeout after two. Shouldn't error.
	seconds = 1
	err = client.Call("RPCTestType.Sleep", &seconds, &reply, 2)
	if err != nil {
		t.Errorf("RPCTestType.Sleep error: %s", err)
	}

	// Sleep for one second, never timeout. Shouldn't error.
	seconds = 1
	err = client.Call("RPCTestType.Sleep", &seconds, &reply, 0)
	if err != nil {
		t.Errorf("RPCTestType.Sleep error: %s", err)
	}

	// Sleep for two seconds, timeout after one. Should error.
	seconds = 2
	err = client.Call("RPCTestType.Sleep", &seconds, &reply, 1)
	if fmt.Sprintf("%s", err) != "RPC call to RPCTestType.Sleep timed out after 1 seconds." {
		t.Error("Should have timed out, but didn't.")
	}
}
