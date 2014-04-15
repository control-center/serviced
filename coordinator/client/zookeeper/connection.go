package zk_driver

import (
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
	onClose  *func()
}

// Assert that Connection implements client.Connection.
var _ client.Connection = &Connection{}

func (zk *Connection) NewLock(path string) client.Lock {
	return &Lock{
		lock: zklib.NewLock(zk.conn, join(zk.basePath, path), zklib.WorldACL(zklib.PermAll)),
	}
}

func (c *Connection) NewLeader(path string, data []byte) client.Leader {
	return &Leader{
		c:    c,
		path: join(c.basePath, path),
		data: data,
	}
}

// Close the zk connection.
func (zk *Connection) Close() {
	zk.conn.Close()
}

// SetOnClose sets the callback f to be called when Close is called on zk.
func (zk *Connection) SetOnClose(f func()) {
	zk.onClose = &f
}

// Create places data at the node at the given path.
func (zk *Connection) Create(path string, data []byte) error {

	p := join(zk.basePath, path)
	_, err := zk.conn.Create(p, data, 0, zklib.WorldACL(zklib.PermAll))
	if err == zklib.ErrNoNode {
		// Create parent node.
		parts := strings.Split(p, "/")
		pth := ""
		for _, p := range parts[1:] {
			pth += "/" + p
			_, err = zk.conn.Create(pth, []byte{}, 0, zklib.WorldACL(zklib.PermAll))
			if err != nil && err != zklib.ErrNodeExists {
				return err
			}
		}
	}
	return err
}

// CreateDir creates an empty node at the given path.
func (zk *Connection) CreateDir(path string) error {
	return zk.Create(path, []byte{})
}

// Exists checks if a node exists at the given path.
func (zk *Connection) Exists(path string) (bool, error) {
	exists, _, err := zk.conn.Exists(join(zk.basePath, path))
	return exists, err
}

// Delete will delete all nodes at the given path or any subpath
func (zk *Connection) Delete(path string) error {
	children, _, err := zk.conn.Children(join(zk.basePath, path))
	if err != nil {
		return err
	}
	// recursively delete children
	for _, child := range children {
		err = zk.Delete(join(path, child))
		if err != nil {
			return err
		}
	}
	return zk.conn.Delete(join(zk.basePath, path), 0)
}
