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
	// RPC Calls that do not require authentication or who handle authentication separately
	NonAuthenticatingCalls = []string{"Master.AuthenticateHost"}
	// TODO: When we implement admin, switch this to a list that does NOT require admin, and update requiresAdmin appropriately
	AdminRequiredCalls = []string{} // RPC calls that require admin access
	endian             = binary.BigEndian
)

var (
	ErrReadingHeader = errors.New("Unable to parse header from RPC request")
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
	for _, name := range AdminRequiredCalls {
		if name == callName {
			return true
		}
	}
	return false
}

// Server Codec
type HeaderParserFunc func([]byte) (auth.Identity, error)
type AuthServerCodec struct {
	conn         io.ReadWriteCloser
	wrappedcodec rpc.ServerCodec
	parseHeader  HeaderParserFunc
	mutex        sync.Mutex // Makes sure we read in order
}

func NewDefaultAuthServerCodec(conn io.ReadWriteCloser) rpc.ServerCodec {
	return NewAuthServerCodec(conn, jsonrpc.NewServerCodec(conn), auth.ExtractRPCHeader)
}

func NewAuthServerCodec(conn io.ReadWriteCloser, codecToWrap rpc.ServerCodec, parseHeader HeaderParserFunc) rpc.ServerCodec {
	return &AuthServerCodec{
		conn:         conn,
		wrappedcodec: codecToWrap,
		parseHeader:  parseHeader,
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

	// Read the first 4 bytes to get the length of the header
	headerLenBuf := make([]byte, 4)
	n, err := a.conn.Read(headerLenBuf)
	if err != nil {
		return err
	}
	if n != 4 {
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
		_, err := a.parseHeader(header)
		if err != nil {
			return err
		}

		//TODO:  Check Admin

		//TODO: We need to get the identity into the request body or somehow check the pool ID
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
type BuildHeaderFunc func() ([]byte, error)
type AuthClientCodec struct {
	conn         io.ReadWriteCloser
	wrappedcodec rpc.ClientCodec
	getHeader    BuildHeaderFunc
	mutex        sync.Mutex
}

func NewDefaultAuthClientCodec(conn io.ReadWriteCloser) rpc.ClientCodec {
	return NewAuthClientCodec(conn, jsonrpc.NewClientCodec(conn), auth.BuildRPCHeader)
}

func NewAuthClientCodec(conn io.ReadWriteCloser, codecToWrap rpc.ClientCodec, getHeader BuildHeaderFunc) rpc.ClientCodec {
	return &AuthClientCodec{
		conn:         conn,
		wrappedcodec: codecToWrap,
		getHeader:    getHeader,
	}
}

// Encodes the request and sends it to the server.
// This implementation gets an auth header when appropriate, and writes it to the stream
//  before letting the underlying codec send the rest of the request.
func (a *AuthClientCodec) WriteRequest(r *rpc.Request, body interface{}) error {
	var (
		header []byte
		err    error
	)

	if requiresAuthentication(r.ServiceMethod) {
		header, err = a.getHeader()
		if err != nil {
			return err
		}
	}

	// add header length
	var headerLen uint32 = uint32(len(header))
	headerLenBuf := make([]byte, 4)
	endian.PutUint32(headerLenBuf, headerLen)

	// Lock to ensure we write the header and the rest of the request back-to-back
	a.mutex.Lock()
	defer a.mutex.Unlock()

	a.conn.Write(headerLenBuf)
	a.conn.Write(header)

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

// NewClient returns a new rpc.Client that uses our client codec
func NewAuthClient(conn io.ReadWriteCloser, codecToWrap rpc.ClientCodec) *rpc.Client {
	return rpc.NewClientWithCodec(NewAuthClientCodec(conn, codecToWrap, auth.BuildRPCHeader))
}

func NewDefaultAuthClient(conn io.ReadWriteCloser) *rpc.Client {
	return rpc.NewClientWithCodec(NewDefaultAuthClientCodec(conn))
}
