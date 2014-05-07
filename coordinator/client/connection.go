// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.
package client

type Driver interface {
	GetConnection(dsn, basePath string) (Connection, error)
}

type Lock interface {
	Lock() error
	Unlock() error
}
type Leader interface {
	TakeLead() (<-chan Event, error)
	ReleaseLead() error
	Current(node Node) error
}

type Node interface {
	Version() interface{}
	SetVersion(interface{})
}

type Connection interface {
	Close()
	SetID(int)
	ID() int
	SetOnClose(func(int))
	Create(path string, node Node) error
	CreateDir(path string) error
	Exists(path string) (bool, error)
	Delete(path string) error
	ChildrenW(path string) (children []string, event <-chan Event, err error)
	Children(path string) (children []string, err error)
	Get(path string, node Node) error
	GetW(path string, node Node) (<-chan Event, error)
	Set(path string, node Node) error

	NewLock(path string) Lock
	NewLeader(path string, data Node) Leader
}
