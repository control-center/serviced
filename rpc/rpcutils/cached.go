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

var addrLocks = make(map[string]*sync.RWMutex)

// GetCachedClient createa or gets a cached Client.
func GetCachedClient(addr string) (Client, error) {
	cList, err := getClientList(addr)
	if err != nil {
		return nil, err
	}
	return cList.getNext()
}

func getClientList(addr string) (*clientList, error) {
	addrLock := getAddrLock(addr)
	addrLock.RLock()
	clients, found := clientCache[addr]
	addrLock.RUnlock()
	if !found {
		return setAndGetClientList(addr)
	}
	return clients, nil
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

func setAndGetClientList(addr string) (*clientList, error) {
	var err error
	addrLock := getAddrLock(addr)
	addrLock.Lock()
	defer addrLock.Unlock()
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
