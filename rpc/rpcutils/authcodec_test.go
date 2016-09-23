// +build unit

package rpcutils

import (
	"bytes"
	"errors"
	. "gopkg.in/check.v1"
	"net/rpc"

	"github.com/control-center/serviced/auth"
	"github.com/control-center/serviced/rpc/rpcutils/mocks"
)

// AuthCodec Helpers
// We need a ReadWriteCloser we can pass to our codecs
type ByteBufferReadWriteCloser struct {
	bytes.Buffer
}

func (b *ByteBufferReadWriteCloser) Close() error {
	return nil
}

type AuthCodecTest struct {
	headerPassed          []byte
	headerToReturn        []byte
	headerErrToReturn     error
	identityToReturn      auth.Identity
	identityErrorToReturn error
	buff                  *ByteBufferReadWriteCloser
	wrappedClientCodec    *mocks.ClientCodec
	wrappedServerCodec    *mocks.ServerCodec
	authClientCodec       rpc.ClientCodec
	authServerCodec       rpc.ServerCodec
}

func NewAuthCodecTest() *AuthCodecTest {

	buff := &ByteBufferReadWriteCloser{}
	test := AuthCodecTest{
		headerPassed:          []byte{},
		headerToReturn:        []byte{},
		headerErrToReturn:     nil,
		identityToReturn:      nil,
		identityErrorToReturn: nil,
		buff: buff,
	}

	wrappedClientCodec := &mocks.ClientCodec{}
	wrappedServerCodec := &mocks.ServerCodec{}
	asc := NewAuthServerCodec(buff, wrappedServerCodec, test.ParseHeader)
	acc := NewAuthClientCodec(buff, wrappedClientCodec, test.GetHeader)

	test.wrappedClientCodec = wrappedClientCodec
	test.wrappedServerCodec = wrappedServerCodec
	test.authClientCodec = acc
	test.authServerCodec = asc

	return &test
}

func (a *AuthCodecTest) Reset() {
	a.buff.Reset()
	a.headerToReturn = []byte{}
	a.headerErrToReturn = nil
	a.headerPassed = []byte{}
	a.identityToReturn = nil
	a.identityErrorToReturn = nil
}

// We need a header getter
func (a *AuthCodecTest) GetHeader() ([]byte, error) {
	return a.headerToReturn, a.headerErrToReturn
}

func (a *AuthCodecTest) ParseHeader(h []byte) (auth.Identity, error) {
	a.headerPassed = h
	return a.identityToReturn, a.identityErrorToReturn
}

func (a *AuthCodecTest) WriteHeaderToBuffer(h []byte) {
	headerLength := uint32(len(h))
	headerLenBuf := make([]byte, 4)
	endian.PutUint32(headerLenBuf, headerLength)
	a.buff.Write(headerLenBuf)
	a.buff.Write(h)
}

func (a *AuthCodecTest) ReadHeaderFromBuffer(length int) []byte {
	return a.buff.Bytes()[4 : length+4]
}

var (
	ErrTestCodec = errors.New("Error calling codec method")
	act          = NewAuthCodecTest()
)

// AuthServerCodec Tests
func (s *MySuite) TestReadRequestHeader(c *C) {

	// Put a header onto the buffer
	header := []byte("Header1")
	act.WriteHeaderToBuffer(header)

	req := &rpc.Request{ServiceMethod: "AuthenticatingCall"}
	act.wrappedServerCodec.On("ReadRequestHeader", req).Return(ErrTestCodec).Once()
	err := act.authServerCodec.ReadRequestHeader(req)
	c.Assert(err, Equals, ErrTestCodec)

	// Test with an authentication required method
	act.Reset()
	act.WriteHeaderToBuffer(header)

	act.identityErrorToReturn = ErrTestCodec
	act.wrappedServerCodec.On("ReadRequestHeader", req).Return(nil).Once()
	err = act.authServerCodec.ReadRequestHeader(req)
	c.Assert(err, Equals, ErrTestCodec)
	c.Assert(act.headerPassed, DeepEquals, header)

	act.Reset()
	act.WriteHeaderToBuffer(header)

	act.wrappedServerCodec.On("ReadRequestHeader", req).Return(nil).Once()
	err = act.authServerCodec.ReadRequestHeader(req)
	c.Assert(err, IsNil)
	c.Assert(act.headerPassed, DeepEquals, header)

	// Test with a non-authentication required method
	act.Reset()
	act.WriteHeaderToBuffer(header)

	req = &rpc.Request{ServiceMethod: "NonAuthenticatingCall"}
	act.wrappedServerCodec.On("ReadRequestHeader", req).Return(nil).Once()
	err = act.authServerCodec.ReadRequestHeader(req)
	c.Assert(err, IsNil)
	c.Assert(act.headerPassed, DeepEquals, []byte{})

}

func (s *MySuite) TestReadRequestBody(c *C) {
	body := 0
	act.wrappedServerCodec.On("ReadRequestBody", body).Return(ErrTestCodec).Once()
	err := act.authServerCodec.ReadRequestBody(body)
	c.Assert(err, Equals, ErrTestCodec)
	act.wrappedServerCodec.On("ReadRequestBody", body).Return(nil).Once()
	err = act.authServerCodec.ReadRequestBody(body)
	c.Assert(err, IsNil)
}

func (s *MySuite) TestWriteResponse(c *C) {
	body := 0
	resp := &rpc.Response{}
	act.wrappedServerCodec.On("WriteResponse", resp, body).Return(ErrTestCodec).Once()
	err := act.authServerCodec.WriteResponse(resp, body)
	c.Assert(err, Equals, ErrTestCodec)
	act.wrappedServerCodec.On("WriteResponse", resp, body).Return(nil).Once()
	err = act.authServerCodec.WriteResponse(resp, body)
	c.Assert(err, IsNil)
}

func (s *MySuite) TestCloseServerCodec(c *C) {
	act.wrappedServerCodec.On("Close").Return(ErrTestCodec).Once()
	err := act.authServerCodec.Close()
	c.Assert(err, Equals, ErrTestCodec)
	act.wrappedServerCodec.On("Close").Return(nil).Once()
	err = act.authServerCodec.Close()
	c.Assert(err, IsNil)
}

// AuthClientCodec Tests
func (s *MySuite) TestWriteRequest(c *C) {
	req := &rpc.Request{ServiceMethod: "AuthenticatingCall"}
	body := 0

	act.headerErrToReturn = ErrTestCodec
	err := act.authClientCodec.WriteRequest(req, body)
	c.Assert(err, Equals, ErrTestCodec)

	act.headerErrToReturn = nil
	act.wrappedClientCodec.On("WriteRequest", req, body).Return(ErrTestCodec).Once()
	err = act.authClientCodec.WriteRequest(req, body)
	c.Assert(err, Equals, ErrTestCodec)

	act.Reset()
	header := []byte("Header1")
	act.headerToReturn = header
	act.wrappedClientCodec.On("WriteRequest", req, body).Return(nil).Once()
	err = act.authClientCodec.WriteRequest(req, body)
	c.Assert(err, IsNil)
	headerWritten := act.ReadHeaderFromBuffer(len(header))
	c.Assert(headerWritten, DeepEquals, header)

	// Try it with a non-authenticating call, and make sure the header is empty
	act.Reset()
	act.headerToReturn = header
	req.ServiceMethod = "NonAuthenticatingCall"
	act.wrappedClientCodec.On("WriteRequest", req, body).Return(nil).Once()
	err = act.authClientCodec.WriteRequest(req, body)
	c.Assert(err, IsNil)
	// First 4 bytes of buffer should be 0s
	bts := act.buff.Bytes()
	c.Assert(bts[:4], DeepEquals, []byte{0, 0, 0, 0})

}

func (s *MySuite) TestReadResponseHeader(c *C) {
	resp := &rpc.Response{}
	act.wrappedClientCodec.On("ReadResponseHeader", resp).Return(ErrTestCodec).Once()
	err := act.authClientCodec.ReadResponseHeader(resp)
	c.Assert(err, Equals, ErrTestCodec)
	act.wrappedClientCodec.On("ReadResponseHeader", resp).Return(nil).Once()
	err = act.authClientCodec.ReadResponseHeader(resp)
	c.Assert(err, IsNil)
}

func (s *MySuite) TestReadResponseBody(c *C) {
	body := 0
	act.wrappedClientCodec.On("ReadResponseBody", body).Return(ErrTestCodec).Once()
	err := act.authClientCodec.ReadResponseBody(body)
	c.Assert(err, Equals, ErrTestCodec)
	act.wrappedClientCodec.On("ReadResponseBody", body).Return(nil).Once()
	err = act.authClientCodec.ReadResponseBody(body)
	c.Assert(err, IsNil)
}

func (s *MySuite) TestCloseClientCodec(c *C) {
	act.wrappedClientCodec.On("Close").Return(ErrTestCodec).Once()
	err := act.authClientCodec.Close()
	c.Assert(err, Equals, ErrTestCodec)
	act.wrappedClientCodec.On("Close").Return(nil).Once()
	err = act.authClientCodec.Close()
	c.Assert(err, IsNil)
}

// Make sure that the header written by the client can be read
//  by the server
func (s *MySuite) TestWriteAndRead(c *C) {
	// Test for an authenticating call
	req := &rpc.Request{ServiceMethod: "AuthenticatingCall"}
	body := 0
	header := []byte("Header1")
	act.headerToReturn = header
	act.wrappedClientCodec.On("WriteRequest", req, body).Return(nil).Once()
	err := act.authClientCodec.WriteRequest(req, body)
	c.Assert(err, IsNil)

	// Token is now on the buffer
	act.wrappedServerCodec.On("ReadRequestHeader", req).Return(nil).Once()
	err = act.authServerCodec.ReadRequestHeader(req)
	c.Assert(err, IsNil)
	c.Assert(act.headerPassed, DeepEquals, header)

	// Test for a non-authenticating call
	act.Reset()
	act.headerToReturn = header
	req.ServiceMethod = "NonAuthenticatingCall"
	act.wrappedClientCodec.On("WriteRequest", req, body).Return(nil).Once()
	err = act.authClientCodec.WriteRequest(req, body)
	c.Assert(err, IsNil)
	// Token is now on the buffer
	act.wrappedServerCodec.On("ReadRequestHeader", req).Return(nil).Once()
	err = act.authServerCodec.ReadRequestHeader(req)
	c.Assert(err, IsNil)
	c.Assert(act.headerPassed, DeepEquals, []byte{})

}
