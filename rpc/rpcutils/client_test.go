// +build unit

package rpcutils

import (
	"fmt"
	"net"
	"net/rpc"
	"sync"
	"testing"
	"time"

	"github.com/control-center/serviced/auth"
	"github.com/control-center/serviced/commons/pool"
	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type MySuite struct{}

var (
	_                   = Suite(&MySuite{})
	rtt                 *RPCTestType
	rpcClient           Client
	bareRpcClient       *rpc.Client
	serveCodec          = rpc.ServeCodec
	serveCodecMutex     sync.RWMutex
	unlockingSleepMutex sync.Mutex
)

type RPCTestType int

func (rtt *RPCTestType) Sleep(sleep time.Duration, reply *time.Duration) error {
	time.Sleep(sleep)
	*reply = sleep
	return nil
}

// Unlocks the unlockingSleepMutex at the start of the method, be sure you have
//  Locked it before calling this one
func (rtt *RPCTestType) UnlockingSleep(sleep time.Duration, reply *time.Duration) error {
	unlockingSleepMutex.Unlock()
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

func (rtt *RPCTestType) NonAuthenticatingCall(arg string, reply *string) error {
	*reply = arg
	return nil
}

func (s *MySuite) SetUpSuite(c *C) {
	NonAuthenticatingCalls = []string{
		"RPCTestType.NonAuthenticatingCall",
	}
	NonAdminRequiredCalls = map[string]struct{}{
		"RPCTestType.NonAdminRequiredCall": struct{}{},
	}
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
			serveCodecMutex.RLock()
			go serveCodec(NewDefaultAuthServerCodec(conn))
			serveCodecMutex.RUnlock()
		}
	}()

	rpcClient, _ = newClient("localhost:32111", 1, DiscardClientTimeout, connectRPC)
	bareRpcClient, _ = connectRPC("localhost:32111")
}

func (s *MySuite) SetUpTest(c *C) {
	auth.ClearKeys()
	auth.ClearToken()
	codectest.Reset()
	// Load master keys so we can authenticate:
	tmpDir := c.MkDir()
	masterKeyFile := fmt.Sprintf("%s/master", tmpDir)
	if err := auth.CreateOrLoadMasterKeys(masterKeyFile); err != nil {
		c.Errorf("Error getting master keys: %s", err)
	}
	serveCodecMutex.Lock()
	defer serveCodecMutex.Unlock()
	serveCodec = rpc.ServeCodec
}

// Test Methods Start Here

func (s *MySuite) TestConcurrentTimeout(c *C) {

	sleepTime := 500 * time.Millisecond
	client, err := newClient("localhost:32111", 1, DiscardClientTimeout, connectRPC)
	c.Assert(err, IsNil)

	unlockingSleepMutex.Lock()
	go func() {
		var reply time.Duration
		// Sleep, timeout after two. Shouldn't error.
		err := client.Call("RPCTestType.UnlockingSleep", sleepTime, &reply, 2*sleepTime)
		c.Assert(err, IsNil)
		c.Assert(reply, Equals, sleepTime)
	}()
	// Wait until previous RPC call has started
	unlockingSleepMutex.Lock()

	var reply time.Duration
	// should time out wating for client
	err = client.Call("RPCTestType.UnlockingSleep", sleepTime, &reply, sleepTime/2)
	c.Assert(err, Equals, pool.ErrItemUnavailable)
}

func (s *MySuite) TestTimeout(c *C) {

	client, err := newClient("localhost:32111", 1, 10*time.Millisecond, connectRPC)

	sleepTime := 1000 * time.Millisecond

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
	sleepTime := time.Second
	// should not time out wating for client
	err = client.Call("RPCTestType.Sleep", sleepTime, &reply, 2*time.Second)
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
	client.Call("RPCTestType.NonAuthenticatingCall", nil, nil, 1)
}

func (s *MySuite) TestNonAuthenticatingCall(c *C) {
	auth.ClearKeys()
	client, err := newClient("localhost:32111", 1, DiscardClientTimeout, connectRPC)
	c.Assert(err, IsNil)
	var p, r string
	p = "Expected"
	err = client.Call("RPCTestType.NonAuthenticatingCall", p, &r, 10*time.Second)
	c.Assert(err, IsNil)
	c.Assert(p, Equals, r)
}

func (s *MySuite) TestUnauthenticatedClient(c *C) {
	auth.ClearKeys()
	// Attempt an RPC call without a token
	client, err := newClient("localhost:32111", 1, DiscardClientTimeout, connectRPC)
	c.Assert(err, IsNil)
	var p, r string
	p = "Expected"
	err = client.Call("RPCTestType.Echo", p, &r, 10*time.Second)
	c.Assert(err, Equals, auth.ErrNotAuthenticated)
}

func (s *MySuite) TestBadToken(c *C) {
	// Create and load a set of keys for this client
	_, dPriv, err := auth.GenerateRSAKeyPairPEM(nil)
	c.Assert(err, IsNil)
	mPub, err := auth.GetMasterPublicKey()
	c.Assert(err, IsNil)
	mPubPEM, err := auth.PEMFromRSAPublicKey(mPub, nil)
	c.Assert(err, IsNil)
	err = auth.LoadDelegateKeysFromPEM(mPubPEM, dPriv)

	// Create and load a bad token
	fakeToken := "This is not a token"
	expiration := time.Now().Add(time.Hour).UTC().Unix()
	getToken := func() (string, int64, error) {
		return fakeToken, expiration, nil
	}
	tmpDir := c.MkDir()
	tokenFile := fmt.Sprintf("%s/token", tmpDir)
	_, err = auth.RefreshToken(getToken, tokenFile)
	c.Assert(err, IsNil)

	// Attempt RPC call with bad token
	client, err := newClient("localhost:32111", 1, DiscardClientTimeout, connectRPC)
	c.Assert(err, IsNil)
	var p, r string
	p = "Expected"
	err = client.Call("RPCTestType.Echo", p, &r, 10*time.Second)
	c.Assert(err, Equals, rpc.ServerError(auth.ErrBadToken.Error()))

}

func (s *MySuite) TestExpiredToken(c *C) {
	// Create and load a set of keys for this client
	dPub, dPriv, err := auth.GenerateRSAKeyPairPEM(nil)
	c.Assert(err, IsNil)
	mPub, err := auth.GetMasterPublicKey()
	c.Assert(err, IsNil)
	mPubPEM, err := auth.PEMFromRSAPublicKey(mPub, nil)
	c.Assert(err, IsNil)
	err = auth.LoadDelegateKeysFromPEM(mPubPEM, dPriv)

	// Create a token that expires in 1 s
	fakeToken, _, err := auth.CreateJWTIdentity("fakehost", "default", true, true, dPub, time.Second)
	// Trick the client into thinking the token doesn't expire for another hour
	expiration := time.Now().Add(time.Hour).UTC().Unix()
	c.Assert(err, IsNil)
	c.Assert(fakeToken, NotNil)

	// Token getter will return the expired token with a non-expired expiration
	getToken := func() (string, int64, error) {
		return fakeToken, expiration, nil
	}

	// Load the token
	tmpDir := c.MkDir()
	tokenFile := fmt.Sprintf("%s/token", tmpDir)
	_, err = auth.RefreshToken(getToken, tokenFile)
	c.Assert(err, IsNil)

	// Make RPC call after token has expired
	fakenow := time.Now().UTC().Add(1 * time.Second)
	auth.At(fakenow, func() {
		// Attempt RPC call with expired token
		client, err := newClient("localhost:32111", 1, DiscardClientTimeout, connectRPC)
		c.Assert(err, IsNil)
		var p, r string
		p = "Expected"
		err = client.Call("RPCTestType.Echo", p, &r, 10*time.Second)
		c.Assert(err, Equals, rpc.ServerError(auth.ErrIdentityTokenExpired.Error()))
	})
}

func (s *MySuite) TestNotAdmin(c *C) {
	// Create and load a set of keys for this client
	dPub, dPriv, err := auth.GenerateRSAKeyPairPEM(nil)
	c.Assert(err, IsNil)
	mPub, err := auth.GetMasterPublicKey()
	c.Assert(err, IsNil)
	mPubPEM, err := auth.PEMFromRSAPublicKey(mPub, nil)
	c.Assert(err, IsNil)
	err = auth.LoadDelegateKeysFromPEM(mPubPEM, dPriv)

	// Create a token that does not have admin prvileges
	fakeToken, expiration, err := auth.CreateJWTIdentity("fakehost", "default", false, true, dPub, time.Hour)
	c.Assert(err, IsNil)
	c.Assert(fakeToken, NotNil)

	// Token getter will return the non-admin token
	getToken := func() (string, int64, error) {
		return fakeToken, expiration, nil
	}

	// Load the token
	tmpDir := c.MkDir()
	tokenFile := fmt.Sprintf("%s/token", tmpDir)
	_, err = auth.RefreshToken(getToken, tokenFile)
	c.Assert(err, IsNil)

	// Attempt RPC call that requires admin
	client, err := newClient("localhost:32111", 1, DiscardClientTimeout, connectRPC)
	c.Assert(err, IsNil)
	var p, r string
	p = "Expected"
	err = client.Call("RPCTestType.Echo", p, &r, 10*time.Second)
	c.Assert(err, Equals, rpc.ServerError(ErrNoAdmin.Error()))
}

// Test multiple calls on the same client to shake out race conditions
func (s *MySuite) TestConcurrentClientCalls(c *C) {
	// Replace rpc.ServeCodec with one that will insert a delay
	serveCodecMutex.Lock()
	serveCodec = func(c rpc.ServerCodec) {
		time.Sleep(200 * time.Millisecond)
		rpc.ServeCodec(c)
	}
	serveCodecMutex.Unlock()

	client, err := connectRPC("localhost:32111")
	c.Assert(err, IsNil)

	// We delay the server and then make 20 simultaneous calls on the same client
	numCalls := 20
	echoStrings := make([]string, numCalls)

	for i := 0; i < numCalls; i++ {
		echoStrings[i] = fmt.Sprintf("String%s", i)
	}
	wg := sync.WaitGroup{}
	for _, s := range echoStrings {
		wg.Add(1)
		go func(param string) {
			defer wg.Done()
			var reply string
			err := client.Call("RPCTestType.Echo", param, &reply)
			c.Assert(err, IsNil)
			c.Assert(reply, Equals, param)
		}(s)
	}

	wg.Wait()
}
