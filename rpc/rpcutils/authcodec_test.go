// +build unit

package rpcutils

import (
	"errors"
	"io"
	"net/rpc"

	"github.com/stretchr/testify/mock"
	. "gopkg.in/check.v1"

	"github.com/control-center/serviced/auth"
	authmocks "github.com/control-center/serviced/auth/mocks"
	"github.com/control-center/serviced/rpc/rpcutils/mocks"
)

type AuthCodecTest struct {
	headerParser       *authmocks.RPCHeaderParser
	headerBuilder      *authmocks.RPCHeaderBuilder
	conn               *mocks.ReadWriteCloser
	wrappedClientCodec *mocks.ClientCodec
	wrappedServerCodec *mocks.ServerCodec
	authClientCodec    rpc.ClientCodec
	authServerCodec    rpc.ServerCodec
	buffer             io.ReadWriteCloser
}

func (act *AuthCodecTest) Reset() {
	wrappedClientCodec := &mocks.ClientCodec{}
	wrappedServerCodec := &mocks.ServerCodec{}
	var buffer io.ReadWriteCloser
	clientCodecCreator := func(b io.ReadWriteCloser) rpc.ClientCodec {
		buffer = b
		return wrappedClientCodec
	}
	serverCodecCreator := func(b io.ReadWriteCloser) rpc.ServerCodec {
		buffer = b
		return wrappedServerCodec
	}

	conn := &mocks.ReadWriteCloser{}
	headerParser := &authmocks.RPCHeaderParser{}
	headerBuilder := &authmocks.RPCHeaderBuilder{}
	asc := NewAuthServerCodec(conn, serverCodecCreator, headerParser)
	acc := NewAuthClientCodec(conn, clientCodecCreator, headerBuilder)

	act.conn = conn
	act.headerParser = headerParser
	act.headerBuilder = headerBuilder
	act.wrappedClientCodec = wrappedClientCodec
	act.wrappedServerCodec = wrappedServerCodec
	act.authClientCodec = acc
	act.authServerCodec = asc
	act.buffer = buffer
}

func NewAuthCodecTest() *AuthCodecTest {
	test := AuthCodecTest{}
	test.Reset()
	return &test
}

var (
	ErrTestCodec      = errors.New("Error calling codec method")
	ErrTestConnection = errors.New("Error calling connection method")
	codectest         = NewAuthCodecTest()
)

// AuthServerCodec Tests
func (s *MySuite) TestReadRequestHeader(c *C) {
	// Set up some objects we'll need
	req := &rpc.Request{ServiceMethod: "AuthenticatingCall"}
	ident := &authmocks.Identity{}
	body := []byte("Body1")

	// Test errors make it through to the client
	codectest.headerParser.On("ReadHeader", codectest.conn).Return(ident, body, ErrTestConnection).Once()
	err := codectest.authServerCodec.ReadRequestHeader(req)
	c.Assert(err, Equals, ErrTestConnection)
	codectest.conn.AssertExpectations(c)

	// Test wrapped Codec error
	codectest.headerParser.On("ReadHeader", codectest.conn).Return(ident, body, nil).Once()
	codectest.wrappedServerCodec.On("ReadRequestHeader", req).Return(ErrTestConnection).Once()
	err = codectest.authServerCodec.ReadRequestHeader(req)
	c.Assert(err, Equals, ErrTestConnection)
	codectest.conn.AssertExpectations(c)

	// Test error from identity parser
	codectest.wrappedServerCodec.On("ReadRequestHeader", req).Return(nil).Once()
	e := &auth.AuthHeaderError{ErrTestCodec, body}
	codectest.headerParser.On("ReadHeader", codectest.conn).Return(ident, nil, e).Once()
	err = codectest.authServerCodec.ReadRequestHeader(req)
	// Error won't come through until we call ReadRequestBody
	c.Assert(err, IsNil)
	b := struct{}{}
	err = codectest.authServerCodec.ReadRequestBody(&b)
	c.Assert(err, Equals, ErrTestCodec)
	codectest.conn.AssertExpectations(c)

	// Test error no admin access
	codectest.wrappedServerCodec.On("ReadRequestHeader", req).Return(nil).Once()
	codectest.headerParser.On("ReadHeader", codectest.conn).Return(ident, body, nil).Once()
	ident.On("HasAdminAccess").Return(false).Once()
	err = codectest.authServerCodec.ReadRequestHeader(req)
	// Error won't come through until we call ReadRequestBody
	c.Assert(err, IsNil)
	err = codectest.authServerCodec.ReadRequestBody(&b)
	c.Assert(err, Equals, ErrNoAdmin)
	codectest.conn.AssertExpectations(c)

	// Test success with admin access
	codectest.wrappedServerCodec.On("ReadRequestHeader", req).Return(nil).Once()
	codectest.headerParser.On("ReadHeader", codectest.conn).Return(ident, body, nil).Once()
	ident.On("HasAdminAccess").Return(true).Once()
	err = codectest.authServerCodec.ReadRequestHeader(req)
	c.Assert(err, IsNil)
	codectest.conn.AssertExpectations(c)

	// Test success with a non-authentication required method
	req = &rpc.Request{ServiceMethod: "RPCTestType.NonAuthenticatingCall"}
	codectest.wrappedServerCodec.On("ReadRequestHeader", req).Return(nil).Once()
	codectest.headerParser.On("ReadHeader", codectest.conn).Return(ident, body, nil).Once()
	err = codectest.authServerCodec.ReadRequestHeader(req)
	c.Assert(err, IsNil)
	codectest.conn.AssertExpectations(c)

	// Test success with a non-admin required method
	req = &rpc.Request{ServiceMethod: "RPCTestType.NonAdminRequiredCall"}
	codectest.wrappedServerCodec.On("ReadRequestHeader", req).Return(nil).Once()
	codectest.headerParser.On("ReadHeader", codectest.conn).Return(ident, body, nil).Once()
	err = codectest.authServerCodec.ReadRequestHeader(req)
	c.Assert(err, IsNil)
	codectest.conn.AssertExpectations(c)
}

func (s *MySuite) TestReadRequestBody(c *C) {
	body := 0
	codectest.wrappedServerCodec.On("ReadRequestBody", body).Return(ErrTestCodec).Once()
	err := codectest.authServerCodec.ReadRequestBody(body)
	c.Assert(err, Equals, ErrTestCodec)
	codectest.wrappedServerCodec.On("ReadRequestBody", body).Return(nil).Once()
	err = codectest.authServerCodec.ReadRequestBody(body)
	c.Assert(err, IsNil)
}

func (s *MySuite) TestWriteResponse(c *C) {
	body := 0
	resp := &rpc.Response{}
	emptyLenBuff := []byte{0, 0}
	var emptyBodyBuff []byte

	// Failure on wrapped codec
	codectest.wrappedServerCodec.On("WriteResponse", resp, body).Return(ErrTestCodec).Once()
	err := codectest.authServerCodec.WriteResponse(resp, body)
	c.Assert(err, Equals, ErrTestCodec)

	// Success
	codectest.wrappedServerCodec.On("WriteResponse", resp, body).Return(nil).Once()
	codectest.conn.On("Write", emptyLenBuff).Return(auth.BodyLenLen, nil).Once()
	codectest.conn.On("Write", emptyBodyBuff).Return(0, nil).Once()
	err = codectest.authServerCodec.WriteResponse(resp, body)
	c.Assert(err, IsNil)
	codectest.conn.AssertExpectations(c)
}

func (s *MySuite) TestCloseServerCodec(c *C) {
	// Error on wrapped codec close
	codectest.wrappedServerCodec.On("Close").Return(ErrTestCodec).Once()
	codectest.conn.On("Close").Return(nil).Once()
	err := codectest.authServerCodec.Close()
	c.Assert(err, Equals, ErrTestCodec)
	codectest.wrappedServerCodec.AssertExpectations(c)

	// Error on connection close
	codectest.wrappedServerCodec.On("Close").Return(nil).Once()
	codectest.conn.On("Close").Return(ErrTestConnection).Once()
	err = codectest.authServerCodec.Close()
	c.Assert(err, Equals, ErrTestConnection)
	codectest.wrappedServerCodec.AssertExpectations(c)

	// Error on both
	codectest.wrappedServerCodec.On("Close").Return(ErrTestCodec).Once()
	codectest.conn.On("Close").Return(ErrTestConnection).Once()
	err = codectest.authServerCodec.Close()
	c.Assert(err, Equals, ErrTestConnection)
	codectest.wrappedServerCodec.AssertExpectations(c)

	// No errors
	codectest.wrappedServerCodec.On("Close").Return(nil).Once()
	codectest.conn.On("Close").Return(nil).Once()
	err = codectest.authServerCodec.Close()
	c.Assert(err, IsNil)
}

// AuthClientCodec Tests
func (s *MySuite) TestWriteRequest(c *C) {
	req := &rpc.Request{ServiceMethod: "AuthenticatingCall"}
	body := 0
	content := []byte("contents")

	// Use this method to write content to the internal buffer used by the codec
	writeContent := func(args mock.Arguments) {
		_, err := codectest.buffer.Write(content)
		c.Assert(err, IsNil)
	}

	// Test error on wrapped codec
	codectest.wrappedClientCodec.On("WriteRequest", req, body).Return(ErrTestCodec).Once()
	err := codectest.authClientCodec.WriteRequest(req, body)
	c.Assert(err, Equals, ErrTestCodec)

	// Test error on WriteHeader
	codectest.wrappedClientCodec.On("WriteRequest", req, body).Return(nil).Run(writeContent).Once()
	codectest.headerBuilder.On("WriteHeader", codectest.conn, content, true).Return(ErrTestCodec).Once()
	err = codectest.authClientCodec.WriteRequest(req, body)
	c.Assert(err, Equals, ErrTestCodec)
	codectest.conn.AssertExpectations(c)

	// Test success on authenticating call
	codectest.wrappedClientCodec.On("WriteRequest", req, body).Return(nil).Run(writeContent).Once()
	codectest.headerBuilder.On("WriteHeader", codectest.conn, content, true).Return(nil).Once()
	err = codectest.authClientCodec.WriteRequest(req, body)
	c.Assert(err, IsNil)
	codectest.conn.AssertExpectations(c)

	// Try it with a non-authenticating call, and make sure the header is empty
	req.ServiceMethod = "RPCTestType.NonAuthenticatingCall"
	codectest.wrappedClientCodec.On("WriteRequest", req, body).Return(nil).Run(writeContent).Once()
	codectest.headerBuilder.On("WriteHeader", codectest.conn, content, false).Return(nil).Once()
	err = codectest.authClientCodec.WriteRequest(req, body)
	c.Assert(err, IsNil)
	codectest.conn.AssertExpectations(c)

}

func (s *MySuite) TestReadResponseHeader(c *C) {
	resp := &rpc.Response{}
	emptyLenBuff := []byte{0, 0}
	var emptyBodyBuff []byte

	// Error on wrapped codec
	codectest.conn.On("Read", emptyLenBuff).Return(2, nil).Once()
	codectest.conn.On("Read", emptyBodyBuff).Return(0, nil).Once()
	codectest.wrappedClientCodec.On("ReadResponseHeader", resp).Return(ErrTestCodec).Once()
	err := codectest.authClientCodec.ReadResponseHeader(resp)
	c.Assert(err, Equals, ErrTestCodec)

	// Success
	codectest.conn.On("Read", emptyLenBuff).Return(2, nil).Once()
	codectest.wrappedClientCodec.On("ReadResponseHeader", resp).Return(nil).Once()
	err = codectest.authClientCodec.ReadResponseHeader(resp)
	c.Assert(err, IsNil)
}

func (s *MySuite) TestReadResponseBody(c *C) {
	body := 0
	codectest.wrappedClientCodec.On("ReadResponseBody", body).Return(ErrTestCodec).Once()
	err := codectest.authClientCodec.ReadResponseBody(body)
	c.Assert(err, Equals, ErrTestCodec)
	codectest.wrappedClientCodec.On("ReadResponseBody", body).Return(nil).Once()
	err = codectest.authClientCodec.ReadResponseBody(body)
	c.Assert(err, IsNil)
}

func (s *MySuite) TestCloseClientCodec(c *C) {
	// Error on wrapped codec close
	codectest.wrappedClientCodec.On("Close").Return(ErrTestCodec).Once()
	codectest.conn.On("Close").Return(nil).Once()
	err := codectest.authClientCodec.Close()
	c.Assert(err, Equals, ErrTestCodec)
	codectest.wrappedClientCodec.AssertExpectations(c)

	// Error on connection close
	codectest.wrappedClientCodec.On("Close").Return(nil).Once()
	codectest.conn.On("Close").Return(ErrTestConnection).Once()
	err = codectest.authClientCodec.Close()
	c.Assert(err, Equals, ErrTestConnection)
	codectest.wrappedClientCodec.AssertExpectations(c)

	// Error on both
	codectest.wrappedClientCodec.On("Close").Return(ErrTestCodec).Once()
	codectest.conn.On("Close").Return(ErrTestConnection).Once()
	err = codectest.authClientCodec.Close()
	c.Assert(err, Equals, ErrTestConnection)
	codectest.wrappedClientCodec.AssertExpectations(c)

	// No errors
	codectest.wrappedClientCodec.On("Close").Return(nil).Once()
	codectest.conn.On("Close").Return(nil).Once()
	err = codectest.authClientCodec.Close()
	c.Assert(err, IsNil)
	codectest.wrappedClientCodec.AssertExpectations(c)
}

func (s *MySuite) TestRequiresAdmin(c *C) {
	result := requiresAdmin("RPCTestType.NonAdminRequiredCall")
	c.Assert(result, Equals, false)
	result = requiresAdmin("RPCTestType.AdminRequiredCall")
	c.Assert(result, Equals, true)
}
