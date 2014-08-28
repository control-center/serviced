// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package zookeeper

import (
	zklib "github.com/samuel/go-zookeeper/zk"
)

// Lock creates a object to facilitate create a locking pattern in zookeeper.
type Lock struct {
	lock *zklib.Lock
}

// Lock attempts to acquire the lock.
func (l *Lock) Lock() (err error) {
	return l.lock.Lock()
}

// Unlock attempts to release the lock.
func (l *Lock) Unlock() error {
	return l.lock.Unlock()
}
