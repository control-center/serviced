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
	"net/rpc"
)

// Server Codec
type AuthServerCodec struct {
	wrappedcodec rpc.ServerCodec
}

func NewAuthServerCodec(codecToWrap rpc.ServerCodec) rpc.ServerCodec {
	return AuthServerCodec{codecToWrap}
}

func (a AuthServerCodec) ReadRequestHeader(r *rpc.Request) error {
	return a.wrappedcodec.ReadRequestHeader(r)
}

func (a AuthServerCodec) ReadRequestBody(body interface{}) error {
	// TODO: Get the token out of the body and authenticate first

	return a.wrappedcodec.ReadRequestBody(body)
}

func (a AuthServerCodec) WriteResponse(r *rpc.Response, body interface{}) error {
	return a.wrappedcodec.WriteResponse(r, body)
}

func (a AuthServerCodec) Close() error {
	return a.wrappedcodec.Close()
}

// Client Codec
type AuthClientCodec struct {
	wrappedcodec rpc.ClientCodec
}

func NewAuthClientCodec(codecToWrap rpc.ClientCodec) rpc.ClientCodec {
	return AuthClientCodec{codecToWrap}
}

func (a AuthClientCodec) WriteRequest(r *rpc.Request, body interface{}) error {
	// TODO: Wrap the body up with the token

	return a.wrappedcodec.WriteRequest(r, body)
}

func (a AuthClientCodec) ReadResponseHeader(r *rpc.Response) error {
	return a.wrappedcodec.ReadResponseHeader(r)
}

func (a AuthClientCodec) ReadResponseBody(body interface{}) error {
	return a.wrappedcodec.ReadResponseBody(body)
}

func (a AuthClientCodec) Close() error {
	return a.wrappedcodec.Close()
}

// NewClient returns a new rpc.Client to handle requests to the
// set of services at the other end of the connection.
func NewAuthClient(wrappedCodec rpc.ClientCodec) *rpc.Client {
	return rpc.NewClientWithCodec(NewAuthClientCodec(wrappedCodec))
}
