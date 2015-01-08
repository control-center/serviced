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
	"log"
	"os/exec"
	lpath "path"
	"path/filepath"
	"strings"
	"time"

	"github.com/control-center/serviced/coordinator/client"
	zklib "github.com/samuel/go-zookeeper/zk"
	"github.com/zenoss/glog"
)

var join = lpath.Join

// Connection is a Zookeeper based implementation of client.Connection.
type Connection struct {
	basePath string
	conn     *zklib.Conn
	servers  []string
	timeout  time.Duration
	onClose  *func(int)
	id       int
}

// Assert that Connection implements client.Connection.
var _ client.Connection = &Connection{}

// NewLock returns a managed lock object at the given path bound to the current
// connection.
func (c *Connection) NewLock(path string) client.Lock {
	return &Lock{
		lock: zklib.NewLock(c.conn, join(c.basePath, path), zklib.WorldACL(zklib.PermAll)),
	}
}

// ID returns the ID of the connection.
func (c *Connection) ID() int {
	return c.id
}

// SetID sets the ID of a connection.
func (c *Connection) SetID(id int) {
	c.id = id
}

// NewLeader returns a managed leader object at the given path bound to the current
// connection.
func (c *Connection) NewLeader(path string, node client.Node) client.Leader {
	return &Leader{
		c:    c,
		path: join(c.basePath, path),
		node: node,
	}
}

// Close the zk connection. Calling close() twice will result in a panic.
func (c *Connection) Close() {
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
		if c.onClose != nil {
			f := *c.onClose
			c.onClose = nil
			f(c.id)
		}
	}
}

// SetOnClose sets the callback f to be called when Close is called on c.
func (c *Connection) SetOnClose(f func(int)) {
	c.onClose = &f
}

func (c *Connection) CreateEphemeral(path string, node client.Node) (string, error) {
	if c.conn == nil {
		return "", client.ErrConnectionClosed
	}

	p := join(c.basePath, path)

	bytes, err := json.Marshal(node)
	if err != nil {
		return "", client.ErrSerialization
	}

	path, err = c.conn.CreateProtectedEphemeralSequential(p, bytes, zklib.WorldACL(zklib.PermAll))
	if err == zklib.ErrNoNode {
		// Create parent node.
		parts := strings.Split(p, "/")
		pth := ""
		if len(parts) > 1 {
			for _, p := range parts[1 : len(parts)-1] {
				pth += "/" + p
				_, err = c.conn.Create(pth, []byte{}, 0, zklib.WorldACL(zklib.PermAll))
				if err != nil && err != zklib.ErrNodeExists {
					return "", xlateError(err)
				}
			}
			path, err = c.conn.CreateProtectedEphemeralSequential(p, bytes, zklib.WorldACL(zklib.PermAll))
		}
	}
	if err == nil {
		node.SetVersion(&zklib.Stat{})
	}
	return path, xlateError(err)
}

// Create places data at the node at the given path.
func (c *Connection) Create(path string, node client.Node) error {
	if c.conn == nil {
		return client.ErrConnectionClosed
	}

	p := join(c.basePath, path)

	bytes, err := json.Marshal(node)
	if err != nil {
		return client.ErrSerialization
	}

	_, err = c.conn.Create(p, bytes, 0, zklib.WorldACL(zklib.PermAll))
	if err == zklib.ErrNoNode {
		// Create parent node.
		parts := strings.Split(p, "/")
		pth := ""
		for _, p := range parts[1:] {
			pth += "/" + p
			_, err = c.conn.Create(pth, []byte{}, 0, zklib.WorldACL(zklib.PermAll))
			if err != nil && err != zklib.ErrNodeExists {
				return xlateError(err)
			}
		}
	}
	if err == nil {
		node.SetVersion(&zklib.Stat{})
	}
	return xlateError(err)
}

type dirNode struct {
	version interface{}
}

func (d *dirNode) Version() interface{}     { return d.version }
func (d *dirNode) SetVersion(v interface{}) { d.version = v }

// CreateDir creates an empty node at the given path.
func (c *Connection) CreateDir(path string) error {
	if c.conn == nil {
		return client.ErrConnectionClosed
	}
	return xlateError(c.Create(path, &dirNode{}))
}

// Exists checks if a node exists at the given path.
func (c *Connection) Exists(path string) (bool, error) {
	if c.conn == nil {
		return false, client.ErrConnectionClosed
	}
	exists, _, err := c.conn.Exists(join(c.basePath, path))
	return exists, xlateError(err)
}

// Delete will delete all nodes at the given path or any subpath
func (c *Connection) Delete(path string) error {
	if c.conn == nil {
		return client.ErrConnectionClosed
	}
	children, _, err := c.conn.Children(join(c.basePath, path))
	if err != nil {
		return xlateError(err)
	}
	// recursively delete children
	for _, child := range children {
		err = c.Delete(join(path, child))
		if err != nil {
			return xlateError(err)
		}
	}
	_, stat, err := c.conn.Get(join(c.basePath, path))
	if err != nil {
		return xlateError(err)
	}
	return xlateError(c.conn.Delete(join(c.basePath, path), stat.Version))
}

func toClientEvent(zkEvent <-chan zklib.Event) <-chan client.Event {
	//use bufferred channel so go routine doesn't block in case the other end abandoned the channel
	echan := make(chan client.Event, 1)
	go func() {
		e := <-zkEvent
		echan <- client.Event{
			Type: client.EventType(e.Type),
		}
	}()
	return echan
}

// ChildrenW returns the children of the node at the give path and a channel of
// events that will yield the next event at that node.
func (c *Connection) ChildrenW(path string) (children []string, event <-chan client.Event, err error) {
	if c.conn == nil {
		return children, event, client.ErrConnectionClosed
	}
	children, _, zkEvent, err := c.conn.ChildrenW(join(c.basePath, path))
	if err != nil {
		return children, nil, xlateError(err)
	}
	return children, toClientEvent(zkEvent), xlateError(err)
}

// GetW gets the node at the given path and return a channel to watch for events on that node.
func (c *Connection) GetW(path string, node client.Node) (event <-chan client.Event, err error) {
	if c.conn == nil {
		return nil, client.ErrConnectionClosed
	}
	return c.getW(join(c.basePath, path), node)
}

func (c *Connection) getW(path string, node client.Node) (event <-chan client.Event, err error) {

	data, stat, zkEvent, err := c.conn.GetW(path)
	if err != nil {
		return nil, xlateError(err)
	}
	if len(data) > 0 {
		glog.V(11).Infof("got data %s", string(data))
		err = json.Unmarshal(data, node)
	} else {
		err = client.ErrEmptyNode
	}
	node.SetVersion(stat)
	return toClientEvent(zkEvent), xlateError(err)
}

// Children returns the children of the node at the given path.
func (c *Connection) Children(path string) (children []string, err error) {
	if c.conn == nil {
		return children, client.ErrConnectionClosed
	}
	children, _, err = c.conn.Children(join(c.basePath, path))
	if err != nil {
		return children, xlateError(err)
	}
	return children, xlateError(err)
}

// Get returns the node at the given path.
func (c *Connection) Get(path string, node client.Node) (err error) {
	if c.conn == nil {
		return client.ErrConnectionClosed
	}
	return c.get(join(c.basePath, path), node)
}

func (c *Connection) get(path string, node client.Node) (err error) {
	data, stat, err := c.conn.Get(path)
	if err != nil {
		return xlateError(err)
	}
	if len(data) > 0 {
		glog.V(11).Infof("got data %s", string(data))
		err = json.Unmarshal(data, node)
	} else {
		err = client.ErrEmptyNode
	}
	node.SetVersion(stat)
	return xlateError(err)
}

// Set serializes the given node and places it at the given path.
func (c *Connection) Set(path string, node client.Node) error {
	if c.conn == nil {
		return client.ErrConnectionClosed
	}
	data, err := json.Marshal(node)
	if err != nil {
		return xlateError(err)
	}

	stat := &zklib.Stat{}
	if node.Version() != nil {
		zstat, ok := node.Version().(*zklib.Stat)
		if !ok {
			return client.ErrInvalidVersionObj
		}
		*stat = *zstat
	}
	_, err = c.conn.Set(join(c.basePath, path), data, stat.Version)
	return xlateError(err)
}

// EnsureZkFatjar downloads the zookeeper binaries for use in unit tests
func EnsureZkFatjar() {
	_, err := exec.LookPath("java")
	if err != nil {
		log.Fatal("Can't find java in path")
	}

	jars, err := filepath.Glob("zookeeper-*/contrib/fatjar/zookeeper-*-fatjar.jar")
	if err != nil {
		log.Fatal("Error search for files")
	}
	if len(jars) > 0 {
		return
	}

	err = exec.Command("curl", "-O", "http://www.java2s.com/Code/JarDownload/zookeeper/zookeeper-3.3.3-fatjar.jar.zip").Run()
	if err != nil {
		log.Fatalf("Could not download fatjar: %s", err)
	}

	err = exec.Command("unzip", "zookeeper-3.3.3-fatjar.jar.zip").Run()
	if err != nil {
		log.Fatalf("Could not unzip fatjar: %s", err)
	}
	err = exec.Command("mkdir", "-p", "zookeeper-3.3.3/contrib/fatjar").Run()
	if err != nil {
		log.Fatalf("Could not make fatjar dir: %s", err)
	}

	err = exec.Command("mv", "zookeeper-3.3.3-fatjar.jar", "zookeeper-3.3.3/contrib/fatjar/").Run()
	if err != nil {
		log.Fatalf("Could not mv fatjar: %s", err)
	}

	err = exec.Command("rm", "zookeeper-3.3.3-fatjar.jar.zip").Run()
	if err != nil {
		log.Fatalf("Could not rm fatjar.zip: %s", err)
	}
}
