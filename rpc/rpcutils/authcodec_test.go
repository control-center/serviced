// +build unit

package rpcutils

import (
	"errors"
	"github.com/stretchr/testify/mock"
	. "gopkg.in/check.v1"
	"net/rpc"

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
}

func NewAuthCodecTest() *AuthCodecTest {

	conn := &mocks.ReadWriteCloser{}
	headerParser := &authmocks.RPCHeaderParser{}
	headerBuilder := &authmocks.RPCHeaderBuilder{}
	wrappedClientCodec := &mocks.ClientCodec{}
	wrappedServerCodec := &mocks.ServerCodec{}
	asc := NewAuthServerCodec(conn, wrappedServerCodec, headerParser)
	acc := NewAuthClientCodec(conn, wrappedClientCodec, headerBuilder)

	test := AuthCodecTest{
		conn:               conn,
		headerParser:       headerParser,
		headerBuilder:      headerBuilder,
		wrappedClientCodec: wrappedClientCodec,
		wrappedServerCodec: wrappedServerCodec,
		authClientCodec:    acc,
		authServerCodec:    asc,
	}

	return &test
}

var (
	ErrTestCodec = errors.New("Error calling codec method")
	codectest    = NewAuthCodecTest()
)

// AuthServerCodec Tests
func (s *MySuite) TestReadRequestHeader(c *C) {
	// Set up some objects we'll need
	req := &rpc.Request{ServiceMethod: "AuthenticatingCall"}
	ident := &authmocks.Identity{}
	header := []byte("Header1")
	headerLen := uint32(len(header))
	emptyHeaderBuff := make([]byte, len(header))
	emptyHeaderLenBuff := make([]byte, HEADER_LEN_BYTES)

	// Test error reading header length
	codectest.conn.On("Read", emptyHeaderLenBuff).Return(0, ErrTestCodec).Once()
	err := codectest.authServerCodec.ReadRequestHeader(req)
	c.Assert(err, Equals, ErrTestCodec)

	codectest.conn.On("Read", emptyHeaderLenBuff).Return(0, nil).Once()
	err = codectest.authServerCodec.ReadRequestHeader(req)
	c.Assert(err, Equals, ErrReadingHeader)

	// Behavior for rest of this test:
	codectest.conn.On("Read", emptyHeaderLenBuff).Return(HEADER_LEN_BYTES, nil).Run(func(args mock.Arguments) {
		buffer := args[0].([]byte)
		endian.PutUint32(buffer, headerLen)
	})

	// Test error reading header
	codectest.conn.On("Read", emptyHeaderBuff).Return(0, ErrTestCodec).Once()
	err = codectest.authServerCodec.ReadRequestHeader(req)
	c.Assert(err, Equals, ErrTestCodec)

	codectest.conn.On("Read", emptyHeaderBuff).Return(0, nil).Once()
	err = codectest.authServerCodec.ReadRequestHeader(req)
	c.Assert(err, Equals, ErrReadingHeader)

	// Read behavior for rest of this test:
	codectest.conn.On("Read", emptyHeaderBuff).Return(len(header), nil).Run(func(args mock.Arguments) {
		buffer := args[0].([]byte)
		_ = copy(header, buffer)
	})

	// Test wrapped Codec error
	codectest.wrappedServerCodec.On("ReadRequestHeader", req).Return(ErrTestCodec).Once()
	err = codectest.authServerCodec.ReadRequestHeader(req)
	c.Assert(err, Equals, ErrTestCodec)

	// Test error from identity parser
	codectest.wrappedServerCodec.On("ReadRequestHeader", req).Return(nil).Once()
	codectest.headerParser.On("ParseHeader", header, req).Return(ident, ErrTestCodec).Once()
	err = codectest.authServerCodec.ReadRequestHeader(req)
	c.Assert(err, Equals, ErrTestCodec)

	// Test error no admin access
	codectest.wrappedServerCodec.On("ReadRequestHeader", req).Return(nil).Once()
	codectest.headerParser.On("ParseHeader", header, req).Return(ident, nil).Once()
	ident.On("HasAdminAccess").Return(false).Once()
	err = codectest.authServerCodec.ReadRequestHeader(req)
	c.Assert(err, Equals, ErrNoAdmin)

	// Test success with admin access
	codectest.wrappedServerCodec.On("ReadRequestHeader", req).Return(nil).Once()
	codectest.headerParser.On("ParseHeader", header, req).Return(ident, nil).Once()
	ident.On("HasAdminAccess").Return(true).Once()
	err = codectest.authServerCodec.ReadRequestHeader(req)
	c.Assert(err, IsNil)

	// Test success with a non-authentication required method
	req = &rpc.Request{ServiceMethod: "NonAuthenticatingCall"}
	codectest.wrappedServerCodec.On("ReadRequestHeader", req).Return(nil).Once()
	err = codectest.authServerCodec.ReadRequestHeader(req)
	c.Assert(err, IsNil)

	// Test success with a non-admin required method
	req = &rpc.Request{ServiceMethod: "NonAdminRequiredCall"}
	codectest.wrappedServerCodec.On("ReadRequestHeader", req).Return(nil).Once()
	codectest.headerParser.On("ParseHeader", header, req).Return(ident, nil).Once()
	err = codectest.authServerCodec.ReadRequestHeader(req)
	c.Assert(err, IsNil)
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
	codectest.wrappedServerCodec.On("WriteResponse", resp, body).Return(ErrTestCodec).Once()
	err := codectest.authServerCodec.WriteResponse(resp, body)
	c.Assert(err, Equals, ErrTestCodec)
	codectest.wrappedServerCodec.On("WriteResponse", resp, body).Return(nil).Once()
	err = codectest.authServerCodec.WriteResponse(resp, body)
	c.Assert(err, IsNil)
}

func (s *MySuite) TestCloseServerCodec(c *C) {
	codectest.wrappedServerCodec.On("Close").Return(ErrTestCodec).Once()
	err := codectest.authServerCodec.Close()
	c.Assert(err, Equals, ErrTestCodec)
	codectest.wrappedServerCodec.On("Close").Return(nil).Once()
	err = codectest.authServerCodec.Close()
	c.Assert(err, IsNil)
}

// AuthClientCodec Tests
func (s *MySuite) TestWriteRequest(c *C) {
	req := &rpc.Request{ServiceMethod: "AuthenticatingCall"}
	body := 0
	header := []byte("Header1")
	expectedHeaderLen := []byte{0, 0, 0, 7}

	// Test error on BuildHeader
	codectest.headerBuilder.On("BuildHeader", req).Return(nil, ErrTestCodec).Once()
	err := codectest.authClientCodec.WriteRequest(req, body)
	c.Assert(err, Equals, ErrTestCodec)

	codectest.conn.On("Write", expectedHeaderLen).Return(4, nil).Once()
	codectest.conn.On("Write", header).Return(7, nil).Once()

	// Test error on wrapped codec
	codectest.headerBuilder.On("BuildHeader", req).Return(header, nil).Once()
	codectest.wrappedClientCodec.On("WriteRequest", req, body).Return(ErrTestCodec).Once()
	err = codectest.authClientCodec.WriteRequest(req, body)
	c.Assert(err, Equals, ErrTestCodec)

	// Test success on authenticating call
	codectest.conn.On("Write", expectedHeaderLen).Return(4, nil).Once()
	codectest.conn.On("Write", header).Return(7, nil).Once()
	codectest.headerBuilder.On("BuildHeader", req).Return(header, nil).Once()
	codectest.wrappedClientCodec.On("WriteRequest", req, body).Return(nil).Once()
	err = codectest.authClientCodec.WriteRequest(req, body)
	c.Assert(err, IsNil)

	// Try it with a non-authenticating call, and make sure the header is empty
	expectedHeader := []byte{}
	expectedHeaderLen = []byte{0, 0, 0, 0}
	codectest.conn.On("Write", expectedHeaderLen).Return(4, nil).Once()
	codectest.conn.On("Write", expectedHeader).Return(0, nil).Once()
	req.ServiceMethod = "NonAuthenticatingCall"
	codectest.wrappedClientCodec.On("WriteRequest", req, body).Return(nil).Once()
	err = codectest.authClientCodec.WriteRequest(req, body)
	c.Assert(err, IsNil)

}

func (s *MySuite) TestReadResponseHeader(c *C) {
	resp := &rpc.Response{}
	codectest.wrappedClientCodec.On("ReadResponseHeader", resp).Return(ErrTestCodec).Once()
	err := codectest.authClientCodec.ReadResponseHeader(resp)
	c.Assert(err, Equals, ErrTestCodec)
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
	codectest.wrappedClientCodec.On("Close").Return(ErrTestCodec).Once()
	err := codectest.authClientCodec.Close()
	c.Assert(err, Equals, ErrTestCodec)
	codectest.wrappedClientCodec.On("Close").Return(nil).Once()
	err = codectest.authClientCodec.Close()
	c.Assert(err, IsNil)
}
