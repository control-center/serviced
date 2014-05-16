package zookeeper

import (
	"encoding/json"
	lpath "path"
	"strings"
	"time"

	zklib "github.com/samuel/go-zookeeper/zk"
	"github.com/zenoss/serviced/coordinator/client"
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

// Create places data at the node at the given path.
func (c *Connection) Create(path string, node client.Node) error {
	if c.conn == nil {
		return client.ErrClosedConnection
	}

	p := join(c.basePath, path)

	bytes, err := json.Marshal(node)
	if err != nil {
		return client.ErrSerialization
	}

	_, err = c.conn.Create(p, bytes, node.Version(), zklib.WorldACL(zklib.PermAll))
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
	return xlateError(err)
}

type dirNode struct{}

func (d *dirNode) Version() int32   { return 0 }
func (d *dirNode) SetVersion(int32) {}

// CreateDir creates an empty node at the given path.
func (c *Connection) CreateDir(path string) error {
	if c.conn == nil {
		return client.ErrClosedConnection
	}
	return xlateError(c.Create(path, &dirNode{}))
}

// Exists checks if a node exists at the given path.
func (c *Connection) Exists(path string) (bool, error) {
	if c.conn == nil {
		return false, client.ErrClosedConnection
	}
	exists, _, err := c.conn.Exists(join(c.basePath, path))
	return exists, xlateError(err)
}

// Delete will delete all nodes at the given path or any subpath
func (c *Connection) Delete(path string) error {
	if c.conn == nil {
		return client.ErrClosedConnection
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
	echan := make(chan client.Event)
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
		return children, event, client.ErrClosedConnection
	}
	children, _, zkEvent, err := c.conn.ChildrenW(join(c.basePath, path))
	if err != nil {
		return children, nil, err
	}
	return children, toClientEvent(zkEvent), xlateError(err)
}

// GetW gets the node at the given path and return a channel to watch for events on that node.
func (c *Connection) GetW(path string, node client.Node) (event <-chan client.Event, err error) {
	if c.conn == nil {
		return nil, client.ErrClosedConnection
	}
	return c.getW(join(c.basePath, path), node)
}

func (c *Connection) getW(path string, node client.Node) (event <-chan client.Event, err error) {

	data, stat, zkEvent, err := c.conn.GetW(path)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(data, node)
	node.SetVersion(stat.Version)
	return toClientEvent(zkEvent), xlateError(err)
}

// Children returns the children of the node at the given path.
func (c *Connection) Children(path string) (children []string, err error) {
	if c.conn == nil {
		return children, client.ErrClosedConnection
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
		return client.ErrClosedConnection
	}
	return c.get(join(c.basePath, path), node)
}

func (c *Connection) get(path string, node client.Node) (err error) {
	data, stat, err := c.conn.Get(path)
	if err != nil {
		return err
	}
	err = json.Unmarshal(data, node)
	node.SetVersion(stat.Version)
	return xlateError(err)
}

// Set serializes the give node and places it at the given path.
func (c *Connection) Set(path string, node client.Node) error {
	if c.conn == nil {
		return client.ErrClosedConnection
	}
	data, err := json.Marshal(node)
	if err != nil {
		return err
	}
	_, err = c.conn.Set(path, data, node.Version())
	return xlateError(err)
}
