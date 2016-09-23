// Copyright 2016 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package rpcutils

import (
	"encoding/binary"
	"errors"
	"github.com/control-center/serviced/auth"
	"io"
	"net/rpc"
	"net/rpc/jsonrpc"
	"sync"
)

var (
	// RPC Calls that do not require authentication or who handle authentication separately:
	NonAuthenticatingCalls = []string{"Master.AuthenticateHost"}
	// RPC calls that do not require admin access:
	NonAdminRequiredCalls = []string{}
	endian                = binary.BigEndian

	ErrReadingHeader = errors.New("Unable to parse header from RPC request")
	ErrWritingHeader = errors.New("Unable to write RPC auth header")
	ErrNoAdmin       = errors.New("Delegate does not have admin access")
)

const (
	HEADER_LEN_BYTES = 4
)

// Checks the RPC method name to see if authentication is required.
//  If it is, calls on the client side will include a signed header, which will be
//  Verified on the server side
func requiresAuthentication(callName string) bool {
	for _, name := range NonAuthenticatingCalls {
		if name == callName {
			return false
		}
	}
	return true
}

// Checks the RPC method name to see if admin-level permissions are required.
//  If they are, it will also check the "admin" attribute on the identity after validating it.
func requiresAdmin(callName string) bool {
	for _, name := range NonAdminRequiredCalls {
		if name == callName {
			return false
		}
	}
	return true
}

// Server Codec
type AuthServerCodec struct {
	conn         io.ReadWriteCloser
	wrappedcodec rpc.ServerCodec
	parser       auth.RPCHeaderParser
	mutex        sync.Mutex // Makes sure we read in order
}

func NewDefaultAuthServerCodec(conn io.ReadWriteCloser) rpc.ServerCodec {
	return NewAuthServerCodec(conn, jsonrpc.NewServerCodec(conn), &auth.RPCHeaderHandler{})
}

func NewAuthServerCodec(conn io.ReadWriteCloser, codecToWrap rpc.ServerCodec, parser auth.RPCHeaderParser) rpc.ServerCodec {
	return &AuthServerCodec{
		conn:         conn,
		wrappedcodec: codecToWrap,
		parser:       parser,
	}
}

// Reads the request header and populates the rpc.Request object.
//  This implementation reads the auth header off the stream first, then
//  lets the underlying codec read the rest.
//  Finally, it validates the identity if necessary.
func (a *AuthServerCodec) ReadRequestHeader(r *rpc.Request) error {
	// Lock so we read both values back-to-back
	a.mutex.Lock()
	defer a.mutex.Unlock()

	// Read the header

	// Read the first HEADER_LEN_BYTES bytes to get the length of the header
	headerLenBuf := make([]byte, HEADER_LEN_BYTES)
	n, err := a.conn.Read(headerLenBuf)
	if err != nil {
		return err
	}
	if n != HEADER_LEN_BYTES {
		return ErrReadingHeader
	}

	headerLength := endian.Uint32(headerLenBuf)

	// Now read the header
	var header []byte
	if headerLength > 0 {
		header = make([]byte, headerLength)
		n, err = a.conn.Read(header)
		if err != nil {
			return err
		}
		if uint32(n) != headerLength {
			return ErrReadingHeader
		}
	}

	// Now let the underlying codec read the rest
	if err := a.wrappedcodec.ReadRequestHeader(r); err != nil {
		return err
	}

	// Now we can get the method name from r and authenticate if required
	if requiresAuthentication(r.ServiceMethod) {
		ident, err := a.parser.ParseHeader(header, r)
		if err != nil {
			return err
		}

		if requiresAdmin(r.ServiceMethod) && !ident.HasAdminAccess() {
			return ErrNoAdmin
		}

		//TODO: save the identity so we can inject it into the request body later

	}
	return nil
}

// Decodes the request and populates the body object with the body of the request
//  We don't change anything here, just let the underlying codec handle it.
//  This always gets called after ReadRequestHeader
func (a *AuthServerCodec) ReadRequestBody(body interface{}) error {
	// TODO: Use reflection and add the identity to the body if necessary
	return a.wrappedcodec.ReadRequestBody(body)
}

//  Encodes the response before sending it back down to the client.
//  We don't change anything here, just let the underlying codec handle it.
func (a *AuthServerCodec) WriteResponse(r *rpc.Response, body interface{}) error {
	return a.wrappedcodec.WriteResponse(r, body)
}

// Closes the connection on the server side
//  We don't change anything here, just let the underlying codec handle it.
func (a *AuthServerCodec) Close() error {
	return a.wrappedcodec.Close()
}

// Client Codec
type AuthClientCodec struct {
	conn          io.ReadWriteCloser
	wrappedcodec  rpc.ClientCodec
	headerBuilder auth.RPCHeaderBuilder
	mutex         sync.Mutex
}

func NewDefaultAuthClientCodec(conn io.ReadWriteCloser) rpc.ClientCodec {
	return NewAuthClientCodec(conn, jsonrpc.NewClientCodec(conn), &auth.RPCHeaderHandler{})
}

func NewAuthClientCodec(conn io.ReadWriteCloser, codecToWrap rpc.ClientCodec, headerBuilder auth.RPCHeaderBuilder) rpc.ClientCodec {
	return &AuthClientCodec{
		conn:          conn,
		wrappedcodec:  codecToWrap,
		headerBuilder: headerBuilder,
	}
}

// Encodes the request and sends it to the server.
// This implementation gets an auth header when appropriate, and writes it to the stream
//  before letting the underlying codec send the rest of the request.
func (a *AuthClientCodec) WriteRequest(r *rpc.Request, body interface{}) error {
	var (
		header []byte = []byte{}
		err    error
	)

	if requiresAuthentication(r.ServiceMethod) {
		header, err = a.headerBuilder.BuildHeader(r)
		if err != nil {
			return err
		}
	}

	// add header length
	var headerLen uint32 = uint32(len(header))
	headerLenBuf := make([]byte, HEADER_LEN_BYTES)
	endian.PutUint32(headerLenBuf, headerLen)

	// Lock to ensure we write the header and the rest of the request back-to-back
	a.mutex.Lock()
	defer a.mutex.Unlock()

	n, err := a.conn.Write(headerLenBuf)
	if err != nil {
		return err
	}
	if n != HEADER_LEN_BYTES {
		return ErrWritingHeader
	}

	n, err = a.conn.Write(header)
	if err != nil {
		return err
	}
	if uint32(n) != headerLen {
		return ErrWritingHeader
	}

	// let the underlying codec write the rest of the request
	if err := a.wrappedcodec.WriteRequest(r, body); err != nil {
		return err
	}

	return nil
}

// Decodes the response and reads the header, building the rpc.Response object
//  We don't change anything here, just let the underlying codec handle it.
func (a *AuthClientCodec) ReadResponseHeader(r *rpc.Response) error {
	return a.wrappedcodec.ReadResponseHeader(r)
}

// Decodes the body of the response and builds the body object.
// We don't change anything here, just let the underlying codec handle it.
func (a *AuthClientCodec) ReadResponseBody(body interface{}) error {
	return a.wrappedcodec.ReadResponseBody(body)
}

// Closes the connection on the client side
//  We don't change anything here, just let the underlying codec handle it.
func (a *AuthClientCodec) Close() error {
	return a.wrappedcodec.Close()
}

// NewDefaultAuthClient returns a new rpc.Client that uses our default client codec
func NewDefaultAuthClient(conn io.ReadWriteCloser) *rpc.Client {
	return rpc.NewClientWithCodec(NewDefaultAuthClientCodec(conn))
}
