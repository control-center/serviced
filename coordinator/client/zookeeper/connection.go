package zk_driver

import (
	"time"

	zklib "github.com/samuel/go-zookeeper/zk"
	"github.com/zenoss/serviced/coordinator/client"
)

// Connection is a Zookeeper based implementation of client.Connection.
type Connection struct {
	conn    *zklib.Conn
	servers []string
	timeout time.Duration
	onClose *func()
}

// Assert that Connection implements client.Connection.
var _ client.Connection = &Connection{}

func (zk *Connection) NewLock(path string) (client.Lock, error) {
	return &Lock{
		lock: zklib.NewLock(zk.conn, path, zklib.WorldACL(zklib.PermAll)),
	}, nil
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
	_, err := zk.conn.Create(path, data, 0, zklib.WorldACL(zklib.PermAll))
	return err
}

// CreateDir creates an empty node at the given path.
func (zk *Connection) CreateDir(path string) error {
	return zk.Create(path, []byte{})
}

// Exists checks if a node exists at the given path.
func (zk *Connection) Exists(path string) (bool, error) {
	exists, _, err := zk.conn.Exists(path)
	return exists, err
}

// Delete will delete all nodes at the given path or any subpath
func (zk *Connection) Delete(path string) error {
	children, _, err := zk.conn.Children(path)
	if err != nil {
		return err
	}
	// recursively delete children
	for _, child := range children {
		err = zk.Delete(path + "/" + child)
		if err != nil {
			return err
		}
	}
	return zk.conn.Delete(path, 0)
}
