package client

import (
	"time"

	zklib "github.com/samuel/go-zookeeper/zk"
)

type ZkDriver struct {
	conn    *zklib.Conn
	servers []string
	timeout time.Duration
}

// Assert that the Zookeeper driver meets the Driver interface
var _ Driver = ZkDriver{}

func NewZkDriver(servers []string, timeout time.Duration) (driver Driver, err error) {

	conn, _, err := zklib.Connect(servers, timeout)
	if err != nil {
		return nil, err
	}

	driver = &ZkDriver{
		conn:    conn,
		servers: servers,
		timeout: timeout,
	}
	return driver, nil
}

func init() {
	registeredDrivers["zookeeper"] = NewZkDriver
}

func (zk ZkDriver) Create(path string, data []byte) error {
	_, err := zk.conn.Create(path, data, 0, zklib.WorldACL(zklib.PermAll))
	return err
}

func (zk ZkDriver) CreateDir(path string) error {
	return zk.Create(path, []byte{})
}

func (zk ZkDriver) Exists(path string) (bool, error) {
	exists, _, err := zk.conn.Exists(path)
	return exists, err
}

func (zk ZkDriver) Delete(path string) error {
	return zk.conn.Delete(path, 0)
}
