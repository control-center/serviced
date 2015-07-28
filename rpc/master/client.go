// Copyright 2014 The Serviced Authors.
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

package master

import (
	"github.com/control-center/serviced/rpc/rpcutils"
)

var (
	empty = struct{}{}
)

// Client a client for interacting with the serviced master
type Client struct {
	addr      string
	rpcClient rpcutils.Client
}

// NewClient Create a new rpc client.
func NewClient(addr string) (*Client, error) {
	client, err := rpcutils.GetCachedClient(addr)
	if err != nil {
		return nil, err
	}
	s := new(Client)
	s.addr = addr
	s.rpcClient = client
	return s, nil
}

func (c *Client) call(name string, request interface{}, response interface{}) error {
	return c.rpcClient.Call("Master."+name, request, response, 0)
}

// Close closes rpc client
func (c *Client) Close() (err error) {
	return c.rpcClient.Close()
}
