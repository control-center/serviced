package zk_driver

import (
	zklib "github.com/samuel/go-zookeeper/zk"
)

type Lock struct {
	lock *zklib.Lock
}

func (l *Lock) Lock() (err error) {
	return l.lock.Lock()
}

func (l *Lock) Unlock() error {
	return l.lock.Unlock()
}
