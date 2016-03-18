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

package zookeeper

import (
	"encoding/json"
	"path"
	"sync"

	zklib "github.com/control-center/go-zookeeper/zk"
	"github.com/control-center/serviced/coordinator/client"
)

// Connection is a Zookeeper based implementation of client.Connection.
type Connection struct {
	sync.RWMutex
	conn     *zklib.Conn
	basePath string
	onClose  func(int)
	id       int
}

// Assert that Connection implements client.Connection.
var _ client.Connection = &Connection{}

// IsClosed returns connection closed error if true, otherwise returns nil.
func (c *Connection) isClosed() error {
	if c.conn == nil {
		return client.ErrConnectionClosed
	}
	return nil
}

// Close closes the client connection to zookeeper. Calling close twice will
// result in a no-op.
func (c *Connection) Close() {
	c.Lock()
	defer c.Unlock()
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
		if c.onClose != nil {
			c.onClose(c.id)
			c.onClose = nil
		}
	}
}

// SetID sets the connection ID
func (c *Connection) SetID(i int) {
	c.Lock()
	defer c.Unlock()
	c.id = i
}

// ID gets the connection ID
func (c *Connection) ID() int {
	c.RLock()
	defer c.RUnlock()
	return c.id
}

// SetOnClose performs cleanup when a connection is closed
func (c *Connection) SetOnClose(onClose func(int)) {
	c.Lock()
	defer c.Unlock()
	if err := c.isClosed(); err == nil {
		c.onClose = onClose
	}
}

// NewTransaction creates a new transaction object
func (c *Connection) NewTransaction() client.Transaction {
	return &Transaction{
		conn: c,
		ops:  []multiReq{},
	}
}

// NewLock creates a new lock object
func (c *Connection) NewLock(p string) (client.Lock, error) {
	c.RLock()
	defer c.RUnlock()
	if err := c.isClosed(); err != nil {
		return nil, err
	}
	lock := &Lock{
		lock: zklib.NewLock(c.conn, path.Join(c.basePath, p), zklib.WorldACL(zklib.PermAll)),
	}
	return lock, nil
}

// NewLeader returns a managed leader object at the given path bound to the
// current connection.
func (c *Connection) NewLeader(p string) (client.Leader, error) {
	c.RLock()
	defer c.RUnlock()
	if err := c.isClosed(); err != nil {
		return nil, err
	}
	return NewLeader(c.conn, path.Join(path.Join(c.basePath, p))), nil
}

// Create adds a node at the specified path
func (c *Connection) Create(path string, node client.Node) error {
	c.RLock()
	defer c.RUnlock()
	if err := c.isClosed(); err != nil {
		return err
	}
	if err := c.ensurePath(path); err != nil {
		return err
	}
	return c.create(path, node)
}

func (c *Connection) create(p string, node client.Node) error {
	bytes, err := json.Marshal(node)
	if err != nil {
		return client.ErrSerialization
	}
	pth := path.Join(c.basePath, p)
	if _, err := c.conn.Create(pth, bytes, 0, zklib.WorldACL(zklib.PermAll)); err != nil {
		return xlateError(err)
	}
	node.SetVersion(&zklib.Stat{})
	return nil
}

// CreateDir adds a dir at the specified path
func (c *Connection) CreateDir(path string) error {
	c.RLock()
	defer c.RUnlock()
	if err := c.isClosed(); err != nil {
		return err
	}
	if err := c.ensurePath(path); err != nil {
		return err
	}
	return c.createDir(path)
}

func (c *Connection) createDir(p string) error {
	pth := path.Join(c.basePath, p)
	_, err := c.conn.Create(pth, []byte{}, 0, zklib.WorldACL(zklib.PermAll))
	return xlateError(err)
}

func (c *Connection) ensurePath(p string) error {
	dp := path.Dir(p)
	// check p instead of dp because if the path is /a/b, we still need to make
	// sure /a is created and if we return nil because the dirpath is root and
	// not the node itself, then the node will not get created below.
	if p == "/" || p == "" {
		return nil
	}
	if exists, err := c.exists(dp); err != nil {
		return err
	} else if exists {
		return nil
	}
	if err := c.ensurePath(dp); err != nil {
		return err
	} else if err := c.createDir(dp); err != client.ErrNodeExists {
		return err
	}
	return nil
}

// CreateEphemeral creates a node whose existance depends on the persistence of
// the connection.
func (c *Connection) CreateEphemeral(path string, node client.Node) (string, error) {
	c.RLock()
	defer c.RUnlock()
	if err := c.isClosed(); err != nil {
		return "", err
	}
	if err := c.ensurePath(path); err != nil {
		return "", err
	}
	return c.createEphemeral(path, node)
}

func (c *Connection) createEphemeral(p string, node client.Node) (string, error) {
	bytes, err := json.Marshal(node)
	if err != nil {
		return "", client.ErrSerialization
	}
	pth := path.Join(c.basePath, p)
	epth, err := c.conn.CreateProtectedEphemeralSequential(pth, bytes, zklib.WorldACL(zklib.PermAll))
	return epth, xlateError(err)
}

// Set assigns a value to an existing node at a given path
func (c *Connection) Set(path string, node client.Node) error {
	c.RLock()
	defer c.RUnlock()
	if err := c.isClosed(); err != nil {
		return err
	}
	return c.set(path, node)
}

func (c *Connection) set(p string, node client.Node) error {
	bytes, err := json.Marshal(node)
	if err != nil {
		return client.ErrSerialization
	}
	stat := &zklib.Stat{}
	if version := node.Version(); version != nil {
		var ok bool
		if stat, ok = version.(*zklib.Stat); !ok {
			return client.ErrInvalidVersionObj
		}
	}
	pth := path.Join(c.basePath, p)
	if _, err := c.conn.Set(pth, bytes, stat.Version); err != nil {
		return xlateError(err)
	}
	return nil
}

// Delete recursively removes a path and its children
func (c *Connection) Delete(path string) error {
	c.RLock()
	defer c.RUnlock()
	if err := c.isClosed(); err != nil {
		return err
	}
	return c.delete(path)
}

func (c *Connection) delete(p string) error {
	children, err := c.children(p)
	if err != nil {
		return err
	}
	for _, child := range children {
		if err := c.delete(path.Join(p, child)); err != nil {
			return err
		}
	}
	pth := path.Join(c.basePath, p)
	_, stat, err := c.conn.Get(pth)
	if err != nil {
		return xlateError(err)
	}
	return xlateError(c.conn.Delete(pth, stat.Version))
}

// Exists returns true if the path exists
func (c *Connection) Exists(path string) (bool, error) {
	c.RLock()
	defer c.RUnlock()
	if err := c.isClosed(); err != nil {
		return false, err
	}
	return c.exists(path)
}

func (c *Connection) exists(p string) (bool, error) {
	exists, _, err := c.conn.Exists(path.Join(c.basePath, p))
	if err == zklib.ErrNoNode {
		return false, nil
	}
	return exists, xlateError(err)
}

// Get returns the node at the given path.
func (c *Connection) Get(path string, node client.Node) error {
	c.RLock()
	defer c.RUnlock()
	if err := c.isClosed(); err != nil {
		return err
	}
	return c.get(path, node)
}

func (c *Connection) get(p string, node client.Node) (err error) {
	p = path.Join(c.basePath, p)
	bytes, stat, err := c.conn.Get(p)
	if err != nil {
		return xlateError(err)
	}
	if len(bytes) > 0 {
		if err := json.Unmarshal(bytes, node); err != nil {
			return client.ErrSerialization
		}
	} else {
		err = client.ErrEmptyNode
	}
	node.SetVersion(stat)
	return
}

// GetW returns the node at the given path as well as a channel to watch for
// events on that node.
func (c *Connection) GetW(path string, node client.Node, cancel <-chan struct{}) (<-chan client.Event, error) {
	c.RLock()
	defer c.RUnlock()
	if err := c.isClosed(); err != nil {
		return nil, err
	}
	return c.getW(path, node, cancel)
}

func (c *Connection) getW(p string, node client.Node, cancel <-chan struct{}) (<-chan client.Event, error) {
	p = path.Join(c.basePath, p)
	bytes, stat, ch, err := c.conn.GetW(p)
	if err != nil {
		return nil, xlateError(err)
	}
	if len(bytes) == 0 {
		return nil, client.ErrEmptyNode
	} else if err := json.Unmarshal(bytes, node); err != nil {
		return nil, client.ErrSerialization
	}
	node.SetVersion(stat)
	return c.toClientEvent(ch, cancel), nil
}

func (c *Connection) toClientEvent(ch <-chan zklib.Event, cancel <-chan struct{}) <-chan client.Event {
	evCh := make(chan client.Event, 1)
	go func() {
		select {
		case zkev := <-ch:
			ev := client.Event{Type: client.EventType(zkev.Type)}
			select {
			case evCh <- ev:
			case <-cancel:
			}
		case <-cancel:
			c.cancelEvent(ch)
		}
	}()
	return evCh
}

func (c *Connection) cancelEvent(ch <-chan zklib.Event) {
	c.RLock()
	defer c.RUnlock()
	if err := c.isClosed(); err != nil {
		return
	}
	c.conn.CancelEvent(ch)
}

// Children returns the children of the node at the given path.
func (c *Connection) Children(path string) ([]string, error) {
	c.RLock()
	defer c.RUnlock()
	if err := c.isClosed(); err != nil {
		return []string{}, err
	}
	return c.children(path)
}

func (c *Connection) children(p string) ([]string, error) {
	pth := path.Join(c.basePath, p)
	children, _, err := c.conn.Children(pth)
	if err != nil {
		return []string{}, xlateError(err)
	}
	return children, nil
}

// ChildrenW returns the children of the node at the given path as well as a
// channel to watch for events on that node.
func (c *Connection) ChildrenW(path string, cancel <-chan struct{}) ([]string, <-chan client.Event, error) {
	c.RLock()
	defer c.RUnlock()
	if err := c.isClosed(); err != nil {
		return []string{}, nil, err
	}
	return c.childrenW(path, cancel)
}

func (c *Connection) childrenW(p string, cancel <-chan struct{}) ([]string, <-chan client.Event, error) {
	p = path.Join(c.basePath, p)
	children, _, ch, err := c.conn.ChildrenW(p)
	if err != nil {
		return []string{}, nil, xlateError(err)
	}
	return children, c.toClientEvent(ch, cancel), nil
}
