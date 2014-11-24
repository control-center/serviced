// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package rpcutils

import (
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
	"sync"
	"time"

	"github.com/zenoss/glog"
)

var dialTimeoutSecs = 30

func SetDialTimeout(timeout int) {
	dialTimeoutSecs = timeout
}

type Client interface {
	Close() error
	Call(serviceMethod string, args interface{}, reply interface{}) error
}

// NewReconnectingClient creates a client that reuses the same connection and does not close the underlying connection unless an error occurs.
// If an RPC call results in an RPC error the underlying connection is reset.
func NewReconnectingClient(addr string) (Client, error) {
	rc := reconnectingClient{}
	rc.addr = addr
	if _, err := rc.connectAndSet(); err != nil {
		return nil, err
	}
	return &rc, nil
}

type reconnectingClient struct {
	sync.RWMutex
	addr         string
	remoteClient *rpc.Client
}

// connectAndSet will create an underlying rpc client, set it and return it if current rpc client is nil
func (rc *reconnectingClient) connectAndSet() (*rpc.Client, error) {
	if rc.remoteClient == nil {
		glog.V(4).Infof("Connecting to %s", rc.addr)
		conn, err := net.DialTimeout("tcp", rc.addr, time.Duration(dialTimeoutSecs) * time.Second)
		if err != nil {
			return nil, err
		}
		rc.remoteClient = jsonrpc.NewClient(conn)
	}
	return rc.remoteClient, nil
}

func (rc *reconnectingClient) Close() error {
	//ignore close as we want to reuse the underlying connections
	return nil
}

func (rc *reconnectingClient) Call(serviceMethod string, args interface{}, reply interface{}) error {
	rc.RLock()
	rpcClient := rc.remoteClient
	var err error
	if rpcClient == nil {
		//release read lock and get write lock
		rc.RUnlock()
		rc.Lock()
		rpcClient, err = rc.connectAndSet()
		//release write lock
		rc.Unlock()
		if err != nil {
			return err
		}
		//get read lock again
		rc.RLock()
	}
	err = rpcClient.Call(serviceMethod, args, reply)
	rc.RUnlock()
	if err != nil {
		glog.V(3).Infof("rpc error, resetting cached client: %v", err)
		rc.Lock()
		rpcClient.Close()
		rc.remoteClient = nil
		rc.connectAndSet()
		rc.Unlock()
	}

	return err
}
