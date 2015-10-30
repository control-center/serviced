// +build unit

package rpcutils

import (
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
	"sync"
	"testing"
	"time"

	"github.com/control-center/serviced/commons/pool"
	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type MySuite struct{}

var _ = Suite(&MySuite{})

type RPCTestType int

func (rtt *RPCTestType) Sleep(seconds *int, reply *int) error {
	time.Sleep(time.Duration(*seconds) * time.Second)
	*reply = *seconds
	return nil
}

func (s *MySuite) SetUpSuite(c *C) {
	rtt := new(RPCTestType)
	rpc.Register(rtt)
	rpc.HandleHTTP()
	listener, err := net.Listen("tcp", ":32111")
	if err != nil {
		c.Errorf("listen error: %s", err)
	}
	go func() {
		for {
			conn, err := listener.Accept()
			defer conn.Close()
			if err != nil {
				c.Errorf("Error accepting connections: %s", err)
			}
			go rpc.ServeCodec(jsonrpc.NewServerCodec(conn))
		}
	}()
}

func (s *MySuite) TestConcurrentTimeout(c *C) {

	client, err := newClient("localhost:32111", 1, connectRPC)
	c.Assert(err, IsNil)

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		var reply, seconds int
		wg.Done()
		// Sleep for one second, timeout after two. Shouldn't error.
		seconds = 1
		err := client.Call("RPCTestType.Sleep", &seconds, &reply, 2*time.Second)
		c.Assert(err, IsNil)
		c.Assert(reply, Equals, seconds)
	}()
	wg.Wait()
	var reply, seconds int
	seconds = 3
	// should time out wating for client
	err = client.Call("RPCTestType.Sleep", &seconds, &reply, 50*time.Millisecond)
	c.Assert(err, Equals, pool.ErrItemUnavailable)
}

func (s *MySuite) TestTimeout(c *C) {

	client, err := newClient("localhost:32111", 2, connectRPC)

	var reply, seconds int

	// Sleep for one second, timeout after two. Shouldn't error.
	seconds = 1
	err = client.Call("RPCTestType.Sleep", &seconds, &reply, 2*time.Second)
	c.Assert(err, IsNil)
	c.Assert(reply, Equals, seconds)

	// Sleep for one second, never timeout. Shouldn't error.
	seconds = 2
	err = client.Call("RPCTestType.Sleep", &seconds, &reply, -1)
	c.Assert(err, IsNil)
	c.Assert(reply, Equals, seconds)

	// Sleep for two seconds, timeout after one. Should error.
	seconds = 2
	err = client.Call("RPCTestType.Sleep", &seconds, &reply, 1*time.Second)
	c.Assert(err, NotNil)
	c.Assert(err, ErrorMatches, "RPC call to RPCTestType.Sleep timed out after .+")
}
