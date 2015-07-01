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

import (
	"bytes"
	"encoding/json"
	"path"
	"sort"
	"strings"
	"sync"
)

// TestConnection is a test connection type
type TestConnection struct {
	id      int
	nodes   map[string][]byte
	watches map[string][]chan<- Event
	Err     error // Connection Error set by the user
	lock    sync.RWMutex
}

// NewTestConnection initializes a new test connection
func NewTestConnection() *TestConnection {
	return new(TestConnection).init()
}

func (conn *TestConnection) init() *TestConnection {
	conn = &TestConnection{
		nodes:   make(map[string][]byte),
		watches: make(map[string][]chan<- Event),
	}
	conn.lock = sync.RWMutex{}
	conn.nodes["/"] = nil
	return conn
}

func (conn *TestConnection) checkpath(p *string) error {
	*p = path.Clean(*p)
	return conn.Err
}

func (conn *TestConnection) updatewatch(p string, eventtype EventType) {
	conn.lock.Lock()
	defer conn.lock.Unlock()
	if watches := conn.watches[p]; watches != nil && len(watches) > 0 {
		delete(conn.watches, p)
		for _, watch := range watches {
			watch <- Event{eventtype, p, nil}
		}
	}

	parent := path.Dir(p)
	if watches := conn.watches[parent]; watches != nil && len(watches) > 0 {
		delete(conn.watches, parent)
		for _, watch := range watches {
			watch <- Event{EventNodeChildrenChanged, parent, nil}
		}
	}
}

func (conn *TestConnection) addwatch(p string) <-chan Event {
	eventC := make(chan Event, 1)
	conn.lock.Lock()
	watches := conn.watches[p]
	conn.watches[p] = append(watches, eventC)
	conn.lock.Unlock()
	return eventC
}

// Close implements Connection.Close
func (conn *TestConnection) Close() {
	var dirW, nodeW []string

	// Organize by child watch vs node watch
	for p := range conn.watches {
		if strings.HasSuffix(p, "/") {
			dirW = append(dirW, p)
		} else {
			nodeW = append(nodeW, p)
		}
	}

	// Signal the top-level path first and continue down the node tree
	sort.Strings(dirW)
	for _, d := range dirW {
		conn.updatewatch(d, EventSession)
	}

	// Signal all of the nodes
	for _, n := range nodeW {
		conn.updatewatch(n, EventSession)
	}
}

// SetID implements Connection.SetID
func (conn *TestConnection) SetID(id int) { conn.id = id }

// ID implements Connection.ID
func (conn *TestConnection) ID() int { return conn.id }

// SetOnClose implements Connection.SetOnClose
func (conn *TestConnection) SetOnClose(f func(int)) {}

// Create implements Connection.Create
func (conn *TestConnection) Create(p string, node Node) error {
	if err := conn.checkpath(&p); err != nil {
		return err
	}

	if node := conn.nodes[p]; node != nil {
		return ErrNodeExists
	} else if err := conn.CreateDir(p); err != nil {
		return err
	}

	data, err := json.Marshal(node)
	if err != nil {
		return err
	}
	conn.lock.Lock()
	conn.nodes[p] = data
	conn.lock.Unlock()
	return nil
}

// CreateDir implements Connection.CreateDir
func (conn *TestConnection) CreateDir(p string) error {
	if err := conn.checkpath(&p); err != nil {
		return err
	}

	if _, exists := conn.nodes[p]; exists {
		return nil
	} else if err := conn.CreateDir(path.Dir(p)); err != nil {
		return err
	}
	conn.lock.Lock()
	conn.nodes[p] = nil
	conn.lock.Unlock()
	conn.updatewatch(p, EventNodeCreated)
	return nil
}

// Exists implements Connection.Exists
func (conn *TestConnection) Exists(p string) (bool, error) {
	if err := conn.checkpath(&p); err != nil {
		return false, err
	}

	conn.lock.RLock()
	defer conn.lock.RUnlock()
	if _, exists := conn.nodes[p]; !exists {
		return false, ErrNoNode
	}

	return true, nil
}

// Delete implements Connection.Delete
func (conn *TestConnection) Delete(p string) error {
	if err := conn.checkpath(&p); err != nil {
		return err
	}

	children, _ := conn.Children(p)
	for _, c := range children {
		if err := conn.Delete(path.Join(p, c)); err != nil {
			return err
		}
	}
	if _, ok := conn.nodes[p]; ok {
		delete(conn.nodes, p)
	}
	conn.updatewatch(p, EventNodeDeleted)
	return nil
}

// ChildrenW implements Connection.ChildrenW
func (conn *TestConnection) ChildrenW(p string) ([]string, <-chan Event, error) {
	if err := conn.checkpath(&p); err != nil {
		return nil, nil, err
	}

	children, err := conn.Children(p)
	if err != nil {
		return nil, nil, err
	}

	return children, conn.addwatch(p), nil
}

// Children implements Connection.Children
func (conn *TestConnection) Children(p string) (children []string, err error) {
	if err := conn.checkpath(&p); err != nil {
		return nil, err
	}

	pattern := path.Join(p, "*")
	conn.lock.Lock()
	defer conn.lock.Unlock()
	for nodepath, _ := range conn.nodes {
		if match, _ := path.Match(pattern, nodepath); match {
			children = append(children, strings.TrimPrefix(nodepath, p+"/"))
		}
	}

	return children, nil
}

// GetW implements Connection.GetW
func (conn *TestConnection) GetW(p string, node Node) (<-chan Event, error) {
	if err := conn.checkpath(&p); err != nil {
		return nil, err
	}

	if err := conn.Get(p, node); err != nil {
		return nil, err
	}

	return conn.addwatch(p), nil
}

// Get implements Connection.Get
func (conn *TestConnection) Get(p string, node Node) error {
	if err := conn.checkpath(&p); err != nil {
		return err
	}

	conn.lock.Lock()
	defer conn.lock.Unlock()
	if data, ok := conn.nodes[p]; !ok {
		return ErrNoNode
	} else if data == nil {
		return ErrEmptyNode
	} else if err := json.Unmarshal(data, node); err != nil {
		return err
	}
	return nil
}

// Set implements Connection.Set
func (conn *TestConnection) Set(p string, node Node) error {
	if err := conn.checkpath(&p); err != nil {
		return err
	}
	conn.lock.Lock()
	if _, ok := conn.nodes[p]; !ok {
		return ErrNoNode
	}
	data, err := json.Marshal(node)
	if err != nil {
		return err
	}
	// only update if something actually changed
	if bytes.Compare(conn.nodes[p], data) != 0 {
		conn.nodes[p] = data
		conn.updatewatch(p, EventNodeDataChanged)
	}
	conn.lock.Unlock()
	return nil
}

type TestLock struct {
}

func (l TestLock) Lock() error {
	return nil
}

func (l TestLock) Unlock() error {
	return nil
}

// NewLock implements Connection.NewLock
func (conn *TestConnection) NewLock(path string) Lock {
	return TestLock{}
}

// NewLeader implements Connection.NewLeader
func (conn *TestConnection) NewLeader(path string, data Node) Leader {
	return nil
}

// CreateEphemeral implements Connection.CreateEphemeral
func (conn *TestConnection) CreateEphemeral(path string, node Node) (string, error) {
	if err := conn.Create(path, node); err != nil {
		return "", err
	}
	return path, nil
}
