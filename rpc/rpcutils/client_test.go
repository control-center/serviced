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

var (
	_             = Suite(&MySuite{})
	rtt           *RPCTestType
	rpcClient     Client
	bareRpcClient *rpc.Client
)

type RPCTestType int

func (rtt *RPCTestType) Sleep(sleep time.Duration, reply *time.Duration) error {
	time.Sleep(sleep)
	*reply = sleep
	return nil
}

type TestArgs struct {
	A string
	B int
	C bool
	D []string
}

func (rtt *RPCTestType) NilReply(arg string, _ *struct{}) error {
	return nil
}

func (rtt *RPCTestType) Echo(arg string, reply *string) error {
	*reply = arg
	return nil
}

func (rtt *RPCTestType) StructCall(arg TestArgs, reply *TestArgs) error {
	*reply = arg
	return nil
}

func (s *MySuite) SetUpSuite(c *C) {
	rtt = new(RPCTestType)
	RegisterLocal("RPCTestType", rtt)
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

	rpcClient, _ = newClient("localhost:32111", 1, DiscardClientTimeout, connectRPC)
	bareRpcClient, _ = connectRPC("localhost:32111")
}

func (s *MySuite) TestConcurrentTimeout(c *C) {

	sleepTime := 100 * time.Millisecond
	client, err := newClient("localhost:32111", 1, DiscardClientTimeout, connectRPC)
	c.Assert(err, IsNil)

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		var reply time.Duration
		wg.Done()
		// Sleep, timeout after two. Shouldn't error.
		err := client.Call("RPCTestType.Sleep", sleepTime, &reply, 2*sleepTime)
		c.Assert(err, IsNil)
		c.Assert(reply, Equals, sleepTime)
	}()
	wg.Wait()
	var reply time.Duration
	// should time out wating for client
	err = client.Call("RPCTestType.Sleep", sleepTime, &reply, sleepTime/2)
	c.Assert(err, Equals, pool.ErrItemUnavailable)
}

func (s *MySuite) TestTimeout(c *C) {

	client, err := newClient("localhost:32111", 1, 10*time.Millisecond, connectRPC)

	sleepTime := 100 * time.Millisecond

	var reply time.Duration

	// Sleep for one second, timeout after two. Shouldn't error.
	err = client.Call("RPCTestType.Sleep", sleepTime, &reply, 2*time.Second)
	c.Assert(err, IsNil)
	c.Assert(reply, Equals, sleepTime)

	// Sleep, never timeout. Shouldn't error.
	sleepTime = sleepTime * 2
	err = client.Call("RPCTestType.Sleep", sleepTime, &reply, 0)
	c.Assert(err, IsNil)
	c.Assert(reply, Equals, sleepTime)

	// Sleep and timeout after half sleep. Should error.
	err = client.Call("RPCTestType.Sleep", &sleepTime, &reply, sleepTime/2)
	c.Assert(err, NotNil)
	c.Assert(err, ErrorMatches, "RPC call to RPCTestType.Sleep timed out after .+")
}

func (s *MySuite) TestLongCall(c *C) {
	client, err := newClient("localhost:32111", 1, 250*time.Millisecond, connectRPC)
	c.Assert(err, IsNil)

	startWg := sync.WaitGroup{}

	wg := sync.WaitGroup{}
	wg.Add(1)
	startWg.Add(1)
	go func() {
		var reply time.Duration
		startWg.Done()
		// Sleep for time , timeout after twice as much. Shouldn't error but underlying client will be invalidated
		sleepTime := 750 * time.Millisecond
		err := client.Call("RPCTestType.Sleep", sleepTime, &reply, 2*sleepTime)
		c.Assert(err, IsNil)
		c.Assert(reply, Equals, sleepTime)
		wg.Done()

	}()
	startWg.Wait()
	//after 250ms the previous call should have caused the the client to go stale
	time.Sleep(500 * time.Millisecond)
	var reply time.Duration
	sleepTime := 10 * time.Millisecond
	// should not time out wating for client
	err = client.Call("RPCTestType.Sleep", sleepTime, &reply, 20*time.Millisecond)
	c.Assert(err, IsNil)
	c.Assert(reply, Equals, sleepTime)
	wg.Wait() //wait for go routine to run asserts
}

func (s *MySuite) TestInvalidAddress(c *C) {
	orig := dialTimeoutSecs
	dialTimeoutSecs = 1
	defer func() {
		dialTimeoutSecs = orig
	}()
	defer func() {
		c.Assert(recover(), IsNil)
	}()
	client, _ := newClient("1.2.3.4:1234", 1, 1, connectRPC)
	// CC-1570: Client is lazy, so have to make a call to cause a panic
	client.Call("whatever", nil, nil, 1)
}
