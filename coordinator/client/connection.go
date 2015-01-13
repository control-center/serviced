// Copyright 2014 The Serviced Authors.
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

package client

// Driver is an interface that allows the coordination.Client
// to get a connection from a driver
type Driver interface {
	GetConnection(dsn, basePath string) (Connection, error)
}

// Lock is an interface that allows the coordination.Client to establish
// a distributed lock for a particular node
type Lock interface {
	Lock() error
	Unlock() error
}

// Leader is an interface that allows the coordination.Client to establish
// a distributed lock that contains pertinant information about the leader
type Leader interface {
	TakeLead() (<-chan Event, error)
	ReleaseLead() error
	Current(node Node) error
}

// Queue is the interface that allows the coordination.Client to establish
// a distributed locking queue
type Queue interface {
	Put(Node) (string, error)
	Get(Node) error
	Consume() error
	HasLock() bool
	Current(Node) error
	Next(Node) error
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
	NewQueue(path string) Queue
}
