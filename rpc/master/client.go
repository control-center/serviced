// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

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
	glog.Infof(" *************************** Connecting to %s", s.addr)
	rpcClient, err := rpc.DialHTTP("tcp", s.addr)
	glog.Infof(" *************************** Dialing done...")
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
