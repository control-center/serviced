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
	"github.com/zenoss/glog"

	"net/rpc"
)

var (
	empty = struct{}{}
)

// Client a client for interacting with the serviced master
type Client struct {
	addr      string
	rpcClient *rpc.Client
}

// NewClient Create a new rpc client.
func NewClient(addr string) (*Client, error) {
	s := new(Client)
	s.addr = addr
	glog.V(4).Infof("Connecting to %s", addr)
	rpcClient, err := rpc.DialHTTP("tcp", s.addr)
	s.rpcClient = rpcClient
	return s, err
}

func (c *Client) call(name string, request interface{}, response interface{}) error {
	return c.rpcClient.Call("Master."+name, request, response)
}

// Close closes rpc client
func (c *Client) Close() (err error) {
	return c.rpcClient.Close()
}
