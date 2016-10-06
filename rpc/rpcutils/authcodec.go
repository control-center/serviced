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
	NonAuthenticatingCalls = []string{"Master.AuthenticateHost", "Agent.BuildHost"}
	// RPC calls that do not require admin access:
	NonAdminRequiredCalls = map[string]struct{}{
		"Master.GetHost":                         struct{}{},
		"Master.GetHosts":                        struct{}{},
		"Master.GetEvaluatedService":             struct{}{},
		"Master.GetSystemUser":                   struct{}{},
		"Master.ReportHealthStatus":              struct{}{},
		"Master.ReportInstanceDead":              struct{}{},
		"ControlCenter.GetServices":              struct{}{},
		"ControlCenterAgent.GetEvaluatedService": struct{}{},
		"ControlCenterAgent.GetHostID":           struct{}{},
		"ControlCenterAgent.GetZkInfo":           struct{}{},
		"ControlCenterAgent.Ping":                struct{}{},
		"ControlCenterAgent.GetISvcEndpoints":    struct{}{},
		"ControlCenterAgent.ReportHealthStatus":  struct{}{},
		"ControlCenterAgent.ReportInstanceDead":  struct{}{},
		"ControlCenterAgent.SendLogMessage":      struct{}{},
	}
	endian = binary.BigEndian

	ErrWritingLength = errors.New("Wrote too few bytes for message length")
	ErrWritingBody   = errors.New("Wrote too few bytes for message body")
	ErrReadingLength = errors.New("Read too few bytes for message length")
	ErrReadingBody   = errors.New("Read too few bytes for message body")
	ErrNoAdmin       = errors.New("Delegate does not have admin access")

	log = logging.PackageLogger()
)

const (
	LEN_BYTES = 4
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

// Convenience methods for Reading/Writing data in the format [LENGTH|DATA]
func WriteLengthAndBytes(b []byte, writer io.Writer) error {
	// write length
	var bLen uint32 = uint32(len(b))
	bLenBuf := make([]byte, LEN_BYTES)
	endian.PutUint32(bLenBuf, bLen)

	n, err := writer.Write(bLenBuf)
	if err != nil {
		return err
	}
	if n != LEN_BYTES {
		return ErrWritingLength
	}

	n, err = writer.Write(b)
	if err != nil {
		return err
	}
	if uint32(n) != bLen {
		return ErrWritingBody
	}

	return nil
}

func ReadLengthAndBytes(reader io.Reader) ([]byte, error) {
	// Read the length of the data
	bLenBuf := make([]byte, LEN_BYTES)
	n, err := io.ReadFull(reader, bLenBuf)
	if err != nil {
		return nil, err
	}
	if n != LEN_BYTES {
		return nil, ErrReadingLength
	}

	bLength := endian.Uint32(bLenBuf)

	// Now read the data
	var b []byte
	if bLength > 0 {
		b = make([]byte, bLength)
		n, err = io.ReadFull(reader, b)
		if err != nil {
			return nil, err
		}
		if uint32(n) != bLength {
			return nil, ErrReadingBody
		}
	}

	return b, nil
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

	// Read the header
	header, err := ReadLengthAndBytes(a.conn)
	if err != nil {
		if err != io.EOF {
			log.WithError(err).Errorf("Error reading authentication header")
		}
		return err
	}

	// Read the rest of the request
	body, err := ReadLengthAndBytes(a.conn)
	if err != nil {
		log.WithError(err).Errorf("Error reading RPC request")
		return err
	}

	// Now write the actual request to the buffer
	if _, err = a.buff.ReadBuff.Write(body); err != nil {
		log.WithError(err).Errorf("Error buffering RPC request")
		return err
	}

	// Let the underlying codec read the request from the buffer and parse it
	if err := a.wrappedcodec.ReadRequestHeader(r); err != nil {
		log.WithError(err).Errorf("Error parsing RPC request")
		return err
	}

	log.WithField("ServiceMethod", r.ServiceMethod).Debugf("Received RPC request")

	// Now we can get the method name from r and authenticate if required
	//  If this fails, save the error to return later
	//  If we return an error now, the server will simply close the connection
	//  This is safe because go's rpc server always calls ReadRequestHeader and ReadRequestBody back-to-back
	//   (unless ReadRequestHeader returns an error)
	if requiresAuthentication(r.ServiceMethod) {
		ident, err := a.parser.ParseHeader(header, body)
		if err != nil {
			log.WithError(err).WithField("ServiceMethod", r.ServiceMethod).Errorf("Could not authenticate RPC request")
			a.lastError = err
		} else if requiresAdmin(r.ServiceMethod) && !ident.HasAdminAccess() {
			log.WithField("ServiceMethod", r.ServiceMethod).Errorf("Received unauthorized RPC request")
			a.lastError = ErrNoAdmin
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
		log.WithError(a.lastError).Errorf("Rejecting RPC request due to authentication error")
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
		log.WithError(err).Errorf("Error encoding RPC response")
		return err
	}

	// Get the response from the buffer and write it to the actual connection
	response := a.buff.WriteBuff.Bytes()
	if err := WriteLengthAndBytes(response, a.conn); err != nil {
		log.WithError(err).WithField("ServiceMethod", r.ServiceMethod).Errorf("Error sending RPC response")
		return err
	}

	return nil
}

// Closes the connection on the server side
//  We don't change anything here, just let the underlying codec handle it.
func (a *AuthServerCodec) Close() error {
	var err error
	if err = a.wrappedcodec.Close(); err != nil {
		log.WithError(err).Errorf("Error closing wrapped RPC client codec")
	}
	if ourErr := a.conn.Close(); ourErr != nil {
		log.WithError(ourErr).Errorf("Error closing RPC client connection")
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
	var (
		header []byte = []byte{}
		err    error
	)

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

	if requiresAuthentication(r.ServiceMethod) {
		header, err = a.headerBuilder.BuildHeader(request)
		if err != nil {
			log.WithError(err).WithField("ServiceMethod", r.ServiceMethod).Errorf("Error building authentication header")
			return err
		}
	}

	// Write the header (may be empty)
	if err = WriteLengthAndBytes(header, a.conn); err != nil {
		log.WithError(err).WithField("ServiceMethod", r.ServiceMethod).Errorf("Error sending authentication header")
		return err
	}

	// Write the rest of the request
	if err = WriteLengthAndBytes(request, a.conn); err != nil {
		log.WithError(err).WithField("ServiceMethod", r.ServiceMethod).Errorf("Error sending rpc request")
		return err
	}

	return nil
}

// Decodes the response and reads the header, building the rpc.Response object
//  We don't change anything here, just let the underlying codec handle it.
func (a *AuthClientCodec) ReadResponseHeader(r *rpc.Response) error {

	// No need for synchronization here, Go's RPC Client makes sure only
	//  One response is read at a time.

	a.buff.ReadBuff.Reset()

	// Read the response from the connection
	response, err := ReadLengthAndBytes(a.conn)
	if err != nil {
		log.WithError(err).WithField("ServiceMethod", r.ServiceMethod).Errorf("Error reading RPC response")
		return err
	}

	// Write the response to the buffer
	if _, err = a.buff.ReadBuff.Write(response); err != nil {
		log.WithError(err).WithField("ServiceMethod", r.ServiceMethod).Errorf("Error buffering RPC response")
		return err
	}

	// Let the underlying codec read and parse the response from the buffer
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
	var err error
	if err = a.wrappedcodec.Close(); err != nil {
		log.WithError(err).Errorf("Error closing wrapped RPC client codec")
	}
	if ourErr := a.conn.Close(); ourErr != nil {
		log.WithError(ourErr).Errorf("Error closing RPC client connection")
		// This error is more important
		err = ourErr
	}
	return err
}

// NewDefaultAuthClient returns a new rpc.Client that uses our default client codec
func NewDefaultAuthClient(conn io.ReadWriteCloser) *rpc.Client {
	return rpc.NewClientWithCodec(NewDefaultAuthClientCodec(conn))
}
