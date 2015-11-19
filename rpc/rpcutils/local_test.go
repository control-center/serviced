// +build unit

package rpcutils

import (
	. "gopkg.in/check.v1"
)

func (s *MySuite) BenchmarkNilReplyDirect(c *C) {
	for n := 0; n < c.N; n++ {
		err := rtt.NilReply("", nil)
		c.Assert(err, IsNil)
	}
}

func (s *MySuite) BenchmarkNilReplyLocal(c *C) {
	client := localRpcClient
	for n := 0; n < c.N; n++ {
		err := client.Call("RPCTestType.NilReply", "", nil, 0)
		c.Assert(err, IsNil)
	}
}

func (s *MySuite) BenchmarkNilReplyCached(c *C) {
	client := rpcClient
	for n := 0; n < c.N; n++ {
		err := client.Call("RPCTestType.NilReply", "", nil, 0)
		c.Assert(err, IsNil)
	}
}

func (s *MySuite) BenchmarkNilReplyRemote(c *C) {
	client := bareRpcClient
	for n := 0; n < c.N; n++ {
		err := client.Call("RPCTestType.NilReply", "", nil)
		c.Assert(err, IsNil)
	}
}

func (s *MySuite) BenchmarkEchoDirect(c *C) {
	reply := ""
	for n := 0; n < c.N; n++ {
		arg := "hello" + string(n)
		err := rtt.Echo(arg, &reply)
		c.Assert(err, IsNil)
		c.Assert(reply, Equals, arg)
	}
}

func (s *MySuite) BenchmarkEchoCached(c *C) {
	client := rpcClient
	reply := ""
	for n := 0; n < c.N; n++ {
		arg := "hello" + string(n)
		err := client.Call("RPCTestType.Echo", arg, &reply, 0)
		c.Assert(err, IsNil)
		c.Assert(reply, Equals, arg)
	}
}

func (s *MySuite) BenchmarkEchoLocal(c *C) {
	client := localRpcClient
	reply := ""
	for n := 0; n < c.N; n++ {
		arg := "hello" + string(n)
		err := client.Call("RPCTestType.Echo", arg, &reply, 0)
		c.Assert(err, IsNil)
		c.Assert(reply, Equals, arg)
	}
}
func (s *MySuite) BenchmarkEchoBareRPC(c *C) {
	client := bareRpcClient
	reply := ""
	for n := 0; n < c.N; n++ {
		arg := "hello" + string(n)
		err := client.Call("RPCTestType.Echo", arg, &reply)
		c.Assert(err, IsNil)
		c.Assert(reply, Equals, arg)
	}
}

func (s *MySuite) BenchmarkStructDirect(c *C) {
	arg := TestArgs{A: "test", B: 10, C: true, D: make([]string, 1000)}
	var reply TestArgs
	for n := 0; n < c.N; n++ {
		arg.A = "hello" + string(n)
		err := rtt.StructCall(arg, &reply)
		c.Assert(err, IsNil)
	}
}

func (s *MySuite) BenchmarkStructCached(c *C) {
	client := rpcClient
	arg := TestArgs{A: "test", B: 10, C: true, D: make([]string, 1000)}
	var reply TestArgs
	for n := 0; n < c.N; n++ {
		arg.A = "hello" + string(n)
		err := client.Call("RPCTestType.StructCall", arg, &reply, 0)
		c.Assert(err, IsNil)
	}
}

func (s *MySuite) BenchmarkStructBareRPC(c *C) {
	client := bareRpcClient
	arg := TestArgs{A: "test", B: 10, C: true, D: make([]string, 1000)}
	var reply TestArgs
	for n := 0; n < c.N; n++ {
		arg.A = "hello" + string(n)
		err := client.Call("RPCTestType.StructCall", arg, &reply)
		c.Assert(err, IsNil)
	}
}

func (s *MySuite) BenchmarkStructLocal(c *C) {
	client := localRpcClient
	arg := TestArgs{A: "test", B: 10, C: true, D: make([]string, 1000)}
	var reply TestArgs
	for n := 0; n < c.N; n++ {
		err := client.Call("RPCTestType.StructCall", arg, &reply, 0)
		c.Assert(err, IsNil)
	}
}

func (s *MySuite) TestStructNil(c *C) {
	client := rpcClient
	arg := TestArgs{A: "test", B: 10, C: true, D: make([]string, 1000)}
	err := client.Call("RPCTestType.StructCall", arg, nil, 0)
	c.Assert(err, IsNil)
	err = localRpcClient.Call("RPCTestType.StructCall", arg, nil, 0)
	c.Assert(err, IsNil)

}

func (s *MySuite) TestStruct(c *C) {
	client := rpcClient

	reply := &TestArgs{}
	arg := TestArgs{A: "test", B: 10, C: true, D: make([]string, 1000)}

	err := client.Call("RPCTestType.StructCall", arg, reply, 0)
	c.Assert(err, IsNil)
	c.Assert(*reply, DeepEquals, arg)

	//reset
	reply = &TestArgs{}
	err = localRpcClient.Call("RPCTestType.StructCall", arg, reply, 0)
	c.Assert(err, IsNil)
	//make sure not same pointer to args
	c.Assert(reply, Not(Equals), &arg)
	c.Assert(*reply, DeepEquals, arg)

}

func (s *MySuite) TestStructUnitialized(c *C) {
	client := rpcClient

	var reply *TestArgs
	arg := TestArgs{A: "test", B: 10, C: true, D: make([]string, 1000)}
	err := client.Call("RPCTestType.StructCall", arg, reply, 0)
	c.Assert(err, ErrorMatches, "reading body json.*")

	err = localRpcClient.Call("RPCTestType.StructCall", arg, reply, 0)
	c.Assert(err, ErrorMatches, "processing response .*")
}

func (s *MySuite) TestNoMethod(c *C) {
	client := rpcClient
	err := client.Call("RPCTestType.blam", "", nil, 0)
	c.Assert(err, ErrorMatches, "rpc: can't find method RPCTestType.blam")

	err = localRpcClient.Call("RPCTestType.blam", "", nil, 0)
	c.Assert(err, ErrorMatches, "can't find method RPCTestType.blam")
}

func (s *MySuite) TestBadServer(c *C) {
	client := rpcClient
	err := client.Call("Blam.blam", "", nil, 0)
	c.Assert(err, ErrorMatches, "rpc: can't find service Blam.blam")

	err = localRpcClient.Call("Blam.blam", "", nil, 0)
	c.Assert(err, ErrorMatches, "can't find service Blam.blam")
}
