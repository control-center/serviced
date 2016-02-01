// Copyright 2015 The Serviced Authors.
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
	"sync"
	"time"

	"github.com/zenoss/glog"
)

// RPC_CLIENT_SIZE max number of rpc clients per address
var RPC_CLIENT_SIZE = 1

// RPCCertVerify used to enable server certificate verification
var RPCCertVerify = false

// RPCDisableTLS used to disable TLS connections
var RPCDisableTLS = false

// DiscardClientTimeout timeout for removing client from pool if a call is taking too long. Does not interrupt call.
var DiscardClientTimeout = 30 * time.Second

//map of address to client
var clientCache = make(map[string]Client)
var cacheLock = sync.RWMutex{}

var addrLocks = make(map[string]*sync.RWMutex)

// Client for calling rpc methods
type Client interface {
	Close() error
	// TODO: CHANGE TIMEOUT TO MILLISECONDS, NOT SECONDS
	Call(serviceMethod string, args interface{}, reply interface{}, timeout time.Duration) error
}

// GetCachedClient createa or gets a cached Client.
func GetCachedClient(addr string) (Client, error) {
	return getClient(addr)
}

func getClient(addr string) (Client, error) {
	if _, found := localAddrs[addr]; found {
		glog.V(3).Infof("Getting local client for %s", addr)
		return localRpcClient, nil
	}

	addrLock := getAddrLock(addr)
	addrLock.RLock()
	client, found := clientCache[addr]
	addrLock.RUnlock()
	if !found {
		return setAndGetClient(addr)
	}
	return client, nil
}

func getAddrLock(addr string) *sync.RWMutex {
	cacheLock.RLock()
	addrLock, found := addrLocks[addr]
	cacheLock.RUnlock()
	if !found {
		cacheLock.Lock()
		defer cacheLock.Unlock()
		if addrLock, found = addrLocks[addr]; !found {
			addrLock = &sync.RWMutex{}
			addrLocks[addr] = addrLock
		}
	}
	return addrLock
}

func setAndGetClient(addr string) (Client, error) {
	var err error
	addrLock := getAddrLock(addr)
	addrLock.Lock()
	defer addrLock.Unlock()
	client, found := clientCache[addr]
	if !found {
		connFn := connectRPCTLS
		if RPCDisableTLS {
			connFn = connectRPC
		}
		client, err = newClient(addr, RPC_CLIENT_SIZE, DiscardClientTimeout, connFn)
		if err != nil {
			return nil, err
		}
		clientCache[addr] = client
	}
	return client, nil

}
