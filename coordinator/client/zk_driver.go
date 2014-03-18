
package client

import (
	"time"

	zklib "github.com/samuel/go-zookeeper/zk"
)


type ZkDriver struct {
	conn *zklib.Conn
	servers []string
	timeout time.Duration
}


func NewZkDriver(servers []string, timeout time.Duration) (driver *ZkDriver, err error) {

	conn, _, err := zklib.Connect(servers, timeout)
	if err != nil {
		return nil, err
	}


	driver = &ZkDriver{
		conn: conn,
		servers: servers,
		timeout: timeout,
	}
	return driver, nil
}

func (zk ZkDriver) Create(path string, data []byte	) error {
	_, err := zk.conn.Create(path, []byte{}, 0, zklib.WorldACL(zklib.PermAll))
	return err
}


func (zk ZkDriver) Exists(path string) (bool, error) {
	exists, _, err := zk.conn.Exists(path)
	return exists, err
}

