// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package rpcutils

import (
	"sync"
)

var RPC_CLIENT_SIZE = 1

//map of address to clientList
var clientCache = make(map[string]*clientList)
var cacheLock = sync.RWMutex{}

// GetCachedClient createa or gets a cached Client.
func GetCachedClient(addr string) (Client, error) {
	cList, err := getClientList(addr)
	if err != nil {
		return nil, err
	}
	return cList.getNext()
}

func getClientList(addr string) (*clientList, error) {
	cacheLock.RLock()
	clients, found := clientCache[addr]
	cacheLock.RUnlock()
	if !found {
		return setAndGetClientList(addr)
	}
	return clients, nil
}

func setAndGetClientList(addr string) (*clientList, error) {
	cacheLock.Lock()
	defer cacheLock.Unlock()
	var err error
	clients, found := clientCache[addr]
	if !found {
		clients, err = newClientList(addr, RPC_CLIENT_SIZE)
		if err != nil {
			return nil, err
		}
		clientCache[addr] = clients
	}
	return clients, nil

}
