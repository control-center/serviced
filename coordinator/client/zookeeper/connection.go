package zk_driver

import (
	"encoding/json"
	lpath "path"
	"strings"
	"time"

	zklib "github.com/samuel/go-zookeeper/zk"
	"github.com/zenoss/glog"
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

func (zk *Connection) NewLock(path string) client.Lock {
	return &Lock{
		lock: zklib.NewLock(zk.conn, join(zk.basePath, path), zklib.WorldACL(zklib.PermAll)),
	}
}

func (c *Connection) Id() int {
	return c.id
}

func (c *Connection) SetId(id int) {
	c.id = id
}

func (c *Connection) NewLeader(path string, node client.Node) client.Leader {
	return &Leader{
		c:    c,
		path: join(c.basePath, path),
		node: node,
	}
}

// Close the zk connection.
func (zk *Connection) Close() {
	glog.Infof("connection %s closed", zk)
	zk.conn.Close()
	if zk.onClose != nil {
		f := *zk.onClose
		zk.onClose = nil
		glog.Infof("calling callback %v", f)
		f(zk.id)
	}
}

// SetOnClose sets the callback f to be called when Close is called on zk.
func (zk *Connection) SetOnClose(f func(int)) {
	zk.onClose = &f
}

// Create places data at the node at the given path.
func (zk *Connection) Create(path string, node client.Node) error {

	p := join(zk.basePath, path)

	bytes, err := json.Marshal(node)
	if err != nil {
		return client.ErrSerialization
	}

	_, err = zk.conn.Create(p, bytes, node.Version(), zklib.WorldACL(zklib.PermAll))
	if err == zklib.ErrNoNode {
		// Create parent node.
		parts := strings.Split(p, "/")
		pth := ""
		for _, p := range parts[1:] {
			pth += "/" + p
			_, err = zk.conn.Create(pth, []byte{}, 0, zklib.WorldACL(zklib.PermAll))
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
func (zk *Connection) CreateDir(path string) error {
	return xlateError(zk.Create(path, &dirNode{}))
}

// Exists checks if a node exists at the given path.
func (zk *Connection) Exists(path string) (bool, error) {
	exists, _, err := zk.conn.Exists(join(zk.basePath, path))
	return exists, xlateError(err)
}

// Delete will delete all nodes at the given path or any subpath
func (zk *Connection) Delete(path string) error {
	children, _, err := zk.conn.Children(join(zk.basePath, path))
	if err != nil {
		return xlateError(err)
	}
	// recursively delete children
	for _, child := range children {
		err = zk.Delete(join(path, child))
		if err != nil {
			return xlateError(err)
		}
	}
	return zk.conn.Delete(join(zk.basePath, path), 0)
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

func (zk *Connection) ChildrenW(path string) (children []string, event <-chan client.Event, err error) {
	children, _, zkEvent, err := zk.conn.ChildrenW(join(zk.basePath, path))
	if err != nil {
		return children, nil, err
	}
	return children, toClientEvent(zkEvent), xlateError(err)
}

func (zk *Connection) GetW(path string, node client.Node) (event <-chan client.Event, err error) {
	return zk.getW(join(zk.basePath, path), node)
}

func (zk *Connection) getW(path string, node client.Node) (event <-chan client.Event, err error) {

	data, stat, zkEvent, err := zk.conn.GetW(path)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(data, node)
	node.SetVersion(stat.Version)
	return toClientEvent(zkEvent), xlateError(err)
}

func (zk *Connection) Children(path string) (children []string, err error) {
	children, _, err = zk.conn.Children(join(zk.basePath, path))
	if err != nil {
		return children, xlateError(err)
	}
	return children, xlateError(err)
}

func (zk *Connection) Get(path string, node client.Node) (err error) {
	return zk.get(join(zk.basePath, path), node)
}

func (zk *Connection) get(path string, node client.Node) (err error) {
	data, stat, err := zk.conn.Get(path)
	if err != nil {
		return err
	}
	glog.Infof("path %s, data %v", path, data)
	err = json.Unmarshal(data, node)
	node.SetVersion(stat.Version)
	return xlateError(err)
}

func (zk *Connection) Set(path string, node client.Node) error {
	data, err := json.Marshal(node)
	if err != nil {
		return err
	}
	_, err = zk.conn.Set(path, data, node.Version())
	return xlateError(err)
}
