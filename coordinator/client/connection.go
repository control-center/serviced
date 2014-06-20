// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package client

// Driver is an interface that allows the coordination.Client
// to get a connection from a driver
type Driver interface {
	GetConnection(dsn, basePath string) (Connection, error)
}

// Lock is the interface that a lock implemenation must implement
type Lock interface {
	Lock() error
	Unlock() error
}

// Leader is the interface that a Leaer implementation must implement
type Leader interface {
	TakeLead() (<-chan Event, error)
	ReleaseLead() error
	Current(node Node) error
}

// Node is the interface that a serializable object must implement to
// be stored in a coordination service
type Node interface {
	Version() interface{}
	SetVersion(interface{})
}

// Connection is the interface that allows interaction with the coordination service
type Connection interface {
	Close()
	SetID(int)
	ID() int
	SetOnClose(func(int))
	Create(path string, node Node) error
	CreateDir(path string) error
	CreateEphemeral(path string, node Node) (string, error)
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
