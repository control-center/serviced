// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package rpcutils

import (
	"sync"

	"github.com/zenoss/glog"
)

//map of address to client
var cache = make(map[string]Client)
var cacheLock = sync.RWMutex{}

func getClient(addr string) (Client, bool) {
	cacheLock.RLock()
	defer cacheLock.RUnlock()
	client, found := cache[addr]
	return client, found
}

func createAndSet(addr string) (Client, error) {
	cacheLock.Lock()
	defer cacheLock.Unlock()
	var err error
	client, found := cache[addr]
	if !found {
		client, err = NewReconnectingClient(addr)
		if err != nil {
			return nil, err
		}
		glog.Infof("created client %#v", client)
		cache[addr] = client
	}
	return client, nil
}

// GetCachedClient createa or gets a cached Client.
func GetCachedClient(addr string) (Client, error) {
	s, found := getClient(addr)
	if !found {
		return createAndSet(addr)
	}
	return s, nil
}
