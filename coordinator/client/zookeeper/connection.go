package zk_driver

import (
	"time"

	zklib "github.com/samuel/go-zookeeper/zk"
	"github.com/zenoss/serviced/coordinator/client"
)

type Connection struct {
	conn    *zklib.Conn
	servers []string
	timeout time.Duration
	onClose *func()
}

var _ client.Connection = &Connection{}

func (zk *Connection) Close() {
	zk.conn.Close()
}

func (zk *Connection) SetOnClose(f func()) {
	zk.onClose = &f
}

func (zk *Connection) Create(path string, data []byte) error {
	_, err := zk.conn.Create(path, data, 0, zklib.WorldACL(zklib.PermAll))
	return err
}

func (zk *Connection) Lock(path string) (lockId string, err error) {
	return "", nil
}

func (zk *Connection) Unlock(path, lockId string) error {
	return nil
}

func (zk *Connection) CreateDir(path string) error {
	return zk.Create(path, []byte{})
}

func (zk *Connection) Exists(path string) (bool, error) {
	exists, _, err := zk.conn.Exists(path)
	return exists, err
}

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
