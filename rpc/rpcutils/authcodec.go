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
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"net/rpc"
	"net/rpc/jsonrpc"
	"sync"

	"github.com/control-center/serviced/auth"
	"github.com/control-center/serviced/logging"
)

var (
	// RPC Calls that do not require authentication or who handle authentication separately:
	NonAuthenticatingCalls = []string{
		"Master.AuthenticateHost",
		"Agent.BuildHost",
		"ControlCenterAgent.Ping",
		"Agent.AddHostPrivate",
		"Master.AddHostPrivate",
	}
	// RPC calls that do not require admin access:
	NonAdminRequiredCalls = map[string]struct{}{
		"Master.GetHost":                         struct{}{},
		"Master.GetHosts":                        struct{}{},
		"Master.GetEvaluatedService":             struct{}{},
		"Master.GetSystemUser":                   struct{}{},
		"Master.ReportHealthStatus":              struct{}{},
		"Master.ReportInstanceDead":              struct{}{},
		"Master.UpdateHost":                      struct{}{},
		"ControlCenterAgent.GetEvaluatedService": struct{}{},
		"ControlCenterAgent.GetHostID":           struct{}{},
		"ControlCenterAgent.GetZkInfo":           struct{}{},
		"ControlCenterAgent.GetISvcEndpoints":    struct{}{},
		"ControlCenterAgent.ReportHealthStatus":  struct{}{},
		"ControlCenterAgent.ReportInstanceDead":  struct{}{},
		"ControlCenterAgent.SendLogMessage":      struct{}{},
		"ControlCenterAgent.AddHostPrivate":      struct{}{},
		//"Master.AddHostPrivate":                  struct{}{},
	}
	endian = binary.BigEndian

	ErrNoAdmin = errors.New("Delegate does not have admin access")

	log = logging.PackageLogger()
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
	_, ok := NonAdminRequiredCalls[callName]
	return !ok
}

// We nead a ReadWriteCloser that we can pass to the underlying codec and use
//  To buffer requests and responses from the actual connection
type ByteBufferReadWriteCloser struct {
	ReadBuff  bytes.Buffer // Reads will happen from this buffer
	WriteBuff bytes.Buffer // Writes will happen to this buffer
}

func (b *ByteBufferReadWriteCloser) Read(p []byte) (int, error) {
	return b.ReadBuff.Read(p)
}

func (b *ByteBufferReadWriteCloser) Write(p []byte) (int, error) {
	return b.WriteBuff.Write(p)
}

func (b *ByteBufferReadWriteCloser) Close() error {
	return nil
}

type ServerCodecCreator func(io.ReadWriteCloser) rpc.ServerCodec

// Server Codec
type AuthServerCodec struct {
	conn         io.ReadWriteCloser
	buff         *ByteBufferReadWriteCloser
	wrappedcodec rpc.ServerCodec
	parser       auth.RPCHeaderParser
	wBuffMutex   sync.Mutex // Make sure we buffer one response at a time
	lastError    error
}

func NewDefaultAuthServerCodec(conn io.ReadWriteCloser) rpc.ServerCodec {
	return NewAuthServerCodec(conn, jsonrpc.NewServerCodec, &auth.RPCHeaderHandler{})
}

func NewAuthServerCodec(conn io.ReadWriteCloser, createCodec ServerCodecCreator, parser auth.RPCHeaderParser) rpc.ServerCodec {
	buff := &ByteBufferReadWriteCloser{}
	return &AuthServerCodec{
		conn:         conn,
		buff:         buff,
		wrappedcodec: createCodec(buff),
		parser:       parser,
	}
}

// Reads the request header and populates the rpc.Request object.
//  This implementation reads the auth header off the stream first, then
//  lets the underlying codec read the rest.
//  Finally, it validates the identity if necessary.
func (a *AuthServerCodec) ReadRequestHeader(r *rpc.Request) error {

	// There is no need for synchronization here, since go's RPC server
	//  ensures that requests are read one-at-a-time

	// Reset state
	a.lastError = nil
	a.buff.ReadBuff.Reset()

	ident, body, err := a.parser.ReadHeader(a.conn)
	if err != nil {
		log.WithError(err).WithField("ServiceMethod", r.ServiceMethod).Debug("Could not authenticate RPC request")
		if e, ok := err.(*auth.AuthHeaderError); ok {
			body = e.Payload
			a.lastError = e.Err
		} else {
			return err
		}
	}

	// Now write the actual request to the buffer
	if _, err = a.buff.ReadBuff.Write(body); err != nil {
		return err
	}

	// Let the underlying codec read the request from the buffer and parse it
	if err := a.wrappedcodec.ReadRequestHeader(r); err != nil {
		return err
	}

	log.WithField("ServiceMethod", r.ServiceMethod).Debug("Received RPC request")

	// Now we can get the method name from r and authenticate if required
	//  If this fails, save the error to return later
	//  If we return an error now, the server will simply close the connection
	//  This is safe because go's rpc server always calls ReadRequestHeader and ReadRequestBody back-to-back
	//   (unless ReadRequestHeader returns an error)
	if requiresAuthentication(r.ServiceMethod) {
		if a.lastError == nil {
			if requiresAdmin(r.ServiceMethod) && (ident == nil || !ident.HasAdminAccess()) {
				log.WithField("ServiceMethod", r.ServiceMethod).Debug("Received unauthorized RPC request")
				a.lastError = ErrNoAdmin
			}
		}
		//TODO: save the identity so we can inject it into the request body later
	}
	return nil
}

// Decodes the request and populates the body object with the body of the request
//  We don't change anything here, just let the underlying codec handle it.
//  This always gets called after ReadRequestHeader
func (a *AuthServerCodec) ReadRequestBody(body interface{}) error {
	if a.lastError != nil {
		return a.lastError
	}
	// TODO: Use reflection and add the identity to the body if necessary
	return a.wrappedcodec.ReadRequestBody(body)
}

//  Encodes the response before sending it back down to the client.
//  We don't change anything here, just let the underlying codec handle it.
func (a *AuthServerCodec) WriteResponse(r *rpc.Response, body interface{}) error {
	// We do need a lock here, because the ServerCodec interface specifies
	//  that WriteResponse must be safe for concurrent use by multiple goroutines
	a.wBuffMutex.Lock()
	defer a.wBuffMutex.Unlock()

	a.buff.WriteBuff.Reset()

	// Let the underlying codec write the response to the buffer
	if err := a.wrappedcodec.WriteResponse(r, body); err != nil {
		return err
	}

	// Get the response from the buffer and write it to the actual connection
	response := a.buff.WriteBuff.Bytes()
	if err := auth.WriteLengthAndBytes(response, a.conn); err != nil {
		return err
	}

	return nil
}

// Closes the connection on the server side
//  We don't change anything here, just let the underlying codec handle it.
func (a *AuthServerCodec) Close() error {
	var err error
	if err = a.wrappedcodec.Close(); err != nil {
		log.WithError(err).Debug("Error closing wrapped RPC client codec")
	}
	if ourErr := a.conn.Close(); ourErr != nil {
		log.WithError(ourErr).Debug("Error closing RPC client connection")
		// This error is probably more important
		err = ourErr
	}
	return err
}

// Client Codec
type ClientCodecCreator func(io.ReadWriteCloser) rpc.ClientCodec
type AuthClientCodec struct {
	conn          io.ReadWriteCloser
	buff          *ByteBufferReadWriteCloser
	wrappedcodec  rpc.ClientCodec
	headerBuilder auth.RPCHeaderBuilder
	wBuffMutex    sync.Mutex // Make sure we buffer a whole request before starting the next one
}

func NewDefaultAuthClientCodec(conn io.ReadWriteCloser) rpc.ClientCodec {
	return NewAuthClientCodec(conn, jsonrpc.NewClientCodec, &auth.RPCHeaderHandler{})
}

func NewAuthClientCodec(conn io.ReadWriteCloser, createCodec ClientCodecCreator, headerBuilder auth.RPCHeaderBuilder) rpc.ClientCodec {
	buff := &ByteBufferReadWriteCloser{}
	return &AuthClientCodec{
		conn:          conn,
		buff:          buff,
		wrappedcodec:  createCodec(buff),
		headerBuilder: headerBuilder,
	}
}

// Encodes the request and sends it to the server.
// This implementation gets an auth header when appropriate, and writes it to the stream
//  before letting the underlying codec send the rest of the request.
func (a *AuthClientCodec) WriteRequest(r *rpc.Request, body interface{}) error {
	// Lock to ensure we write the header and the rest of the request back-to-back
	//  This method may be called by multiple goroutines concurrently
	a.wBuffMutex.Lock()
	defer a.wBuffMutex.Unlock()
	a.buff.WriteBuff.Reset()

	// Let the underlying codec write the request to the buffer
	if err := a.wrappedcodec.WriteRequest(r, body); err != nil {
		return err
	}

	// Get the request off the buffer
	request := a.buff.WriteBuff.Bytes()

	needsAuth := requiresAuthentication(r.ServiceMethod)
	if err := a.headerBuilder.WriteHeader(a.conn, request, needsAuth); err != nil {
		return err
	}

	log.WithField("ServiceMethod", r.ServiceMethod).Debug("Successfully sent RPC request")

	return nil
}

// Decodes the response and reads the header, building the rpc.Response object
//  We don't change anything here, just let the underlying codec handle it.
func (a *AuthClientCodec) ReadResponseHeader(r *rpc.Response) error {

	// No need for synchronization here, Go's RPC Client makes sure only
	//  One response is read at a time.

	a.buff.ReadBuff.Reset()

	// Read the response from the connection
	response, err := auth.ReadLengthAndBytes(a.conn)
	if err != nil {
		// It is common to get harmless errors here whenever the client is closed
		return err
	}

	// Write the response to the buffer
	if _, err = a.buff.ReadBuff.Write(response); err != nil {
		return err
	}

	// Let the underlying codec read and parse the response from the buffer
	if err = a.wrappedcodec.ReadResponseHeader(r); err != nil {
		return err
	}

	log.WithField("ServiceMethod", r.ServiceMethod).Debug("Successfully read RPC response")

	return nil
}

// Decodes the body of the response and builds the body object.
// We don't change anything here, just let the underlying codec handle it.
func (a *AuthClientCodec) ReadResponseBody(body interface{}) error {
	return a.wrappedcodec.ReadResponseBody(body)
}

// Closes the connection on the client side
//  We don't change anything here, just let the underlying codec handle it.
func (a *AuthClientCodec) Close() error {
	var err error
	if err = a.wrappedcodec.Close(); err != nil {
		log.WithError(err).Debug("Error closing wrapped RPC client codec")
	}
	if ourErr := a.conn.Close(); ourErr != nil {
		log.WithError(ourErr).Debug("Error closing RPC client connection")
		// This error is more important
		err = ourErr
	}
	return err
}

// NewDefaultAuthClient returns a new rpc.Client that uses our default client codec
func NewDefaultAuthClient(conn io.ReadWriteCloser) *rpc.Client {
	return rpc.NewClientWithCodec(NewDefaultAuthClientCodec(conn))
}
