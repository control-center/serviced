// +build unit

package rpcutils

import (
	// "net"
	"net/rpc"
	// "net/rpc/jsonrpc"
	// "sync"
	"github.com/control-center/serviced/rpc/rpcutils/mocks"
	. "gopkg.in/check.v1"
	// "testing"
	"errors"
)

var (
	wrappedClientCodec = &mocks.ClientCodec{}
	wrappedServerCodec = &mocks.ServerCodec{}
	authClientCodec    = NewAuthClientCodec(wrappedClientCodec)
	authServerCodec    = NewAuthServerCodec(wrappedServerCodec)

	ErrTestCodec = errors.New("Error calling codec method")
)

// AuthServerCodec Tests
func (s *MySuite) TestReadRequestHeader(c *C) {
	req := &rpc.Request{}
	wrappedServerCodec.On("ReadRequestHeader", req).Return(ErrTestCodec).Once()
	err := authServerCodec.ReadRequestHeader(req)
	c.Assert(err, Equals, ErrTestCodec)
	wrappedServerCodec.On("ReadRequestHeader", req).Return(nil).Once()
	err = authServerCodec.ReadRequestHeader(req)
	c.Assert(err, IsNil)
}

func (s *MySuite) TestReadRequestBody(c *C) {
	body := 0
	wrappedServerCodec.On("ReadRequestBody", body).Return(ErrTestCodec).Once()
	err := authServerCodec.ReadRequestBody(body)
	c.Assert(err, Equals, ErrTestCodec)
	wrappedServerCodec.On("ReadRequestBody", body).Return(nil).Once()
	err = authServerCodec.ReadRequestBody(body)
	c.Assert(err, IsNil)
}

func (s *MySuite) TestWriteResponse(c *C) {
	body := 0
	resp := &rpc.Response{}
	wrappedServerCodec.On("WriteResponse", resp, body).Return(ErrTestCodec).Once()
	err := authServerCodec.WriteResponse(resp, body)
	c.Assert(err, Equals, ErrTestCodec)
	wrappedServerCodec.On("WriteResponse", resp, body).Return(nil).Once()
	err = authServerCodec.WriteResponse(resp, body)
	c.Assert(err, IsNil)
}

func (s *MySuite) TestCloseServerCodec(c *C) {
	wrappedServerCodec.On("Close").Return(ErrTestCodec).Once()
	err := authServerCodec.Close()
	c.Assert(err, Equals, ErrTestCodec)
	wrappedServerCodec.On("Close").Return(nil).Once()
	err = authServerCodec.Close()
	c.Assert(err, IsNil)
}

// AuthClientCodec Tests
func (s *MySuite) TestWriteRequest(c *C) {
	req := &rpc.Request{}
	body := 0
	wrappedClientCodec.On("WriteRequest", req, body).Return(ErrTestCodec).Once()
	err := authClientCodec.WriteRequest(req, body)
	c.Assert(err, Equals, ErrTestCodec)
	wrappedClientCodec.On("WriteRequest", req, body).Return(nil).Once()
	err = authClientCodec.WriteRequest(req, body)
	c.Assert(err, IsNil)
}

func (s *MySuite) TestReadResponseHeader(c *C) {
	resp := &rpc.Response{}
	wrappedClientCodec.On("ReadResponseHeader", resp).Return(ErrTestCodec).Once()
	err := authClientCodec.ReadResponseHeader(resp)
	c.Assert(err, Equals, ErrTestCodec)
	wrappedClientCodec.On("ReadResponseHeader", resp).Return(nil).Once()
	err = authClientCodec.ReadResponseHeader(resp)
	c.Assert(err, IsNil)
}

func (s *MySuite) TestReadResponseBody(c *C) {
	body := 0
	wrappedClientCodec.On("ReadResponseBody", body).Return(ErrTestCodec).Once()
	err := authClientCodec.ReadResponseBody(body)
	c.Assert(err, Equals, ErrTestCodec)
	wrappedClientCodec.On("ReadResponseBody", body).Return(nil).Once()
	err = authClientCodec.ReadResponseBody(body)
	c.Assert(err, IsNil)
}

func (s *MySuite) TestCloseClientCodec(c *C) {
	wrappedClientCodec.On("Close").Return(ErrTestCodec).Once()
	err := authClientCodec.Close()
	c.Assert(err, Equals, ErrTestCodec)
	wrappedClientCodec.On("Close").Return(nil).Once()
	err = authClientCodec.Close()
	c.Assert(err, IsNil)
}
