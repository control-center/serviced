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
