package zzk

import (
	"errors"
	"path"
	"time"

	"github.com/control-center/serviced/coordinator/client"
	"github.com/zenoss/glog"
)

const (
	DefaultConnectionTimeout = time.Minute
	local                    = "local"
	remote                   = "remote"
)

var (
	ErrNotInitialized = errors.New("client not initialized")
)

var (
	manager = make(map[string]*zclient)
)

// GeneratePoolPath generates the path for a pool-based connection
func GeneratePoolPath(poolID string) string {
	return path.Join("/pools", poolID)
}

// InitializeLocalClient initializes the local zookeeper client
func InitializeLocalClient(client *client.Client) {
	manager[local] = &zclient{client, make(map[string]*zconn)}
}

// GetLocalConnection acquires a connection from the local zookeeper client
func GetLocalConnection(path string) (client.Connection, error) {
	localclient, ok := manager[local]
	if !ok || localclient.Client == nil {
		glog.Fatalf("zClient has not been initialized!")
	}
	return localclient.GetConnection(path)
}

// InitializeRemoteClient initializes the remote zookeeper client
func InitializeRemoteClient(client *client.Client) {
	manager[remote] = &zclient{client, make(map[string]*zconn)}
}

// GetRemoteConnection acquires a connection from the remote zookeeper client
func GetRemoteConnection(path string) (client.Connection, error) {
	client, ok := manager[remote]
	if !ok || client.Client == nil {
		return nil, ErrNotInitialized
	}
	return client.GetConnection(path)
}

// ShutdownConnections closes all local and remote zookeeper connections
func ShutdownConnections() {
	for _, client := range manager {
		client.Shutdown()
	}
}

type zclient struct {
	*client.Client
	connections map[string]*zconn
}

func (zclient *zclient) GetConnection(path string) (client.Connection, error) {
	if _, ok := zclient.connections[path]; !ok {
		zclient.connections[path] = newzconn(zclient.Client, path)
	}
	zconn := zclient.connections[path]
	return zconn.connect(DefaultConnectionTimeout)
}

func (zclient *zclient) Shutdown() {
	for _, zconn := range zclient.connections {
		zconn.shutdown()
	}
}

// zconn is the connection listener for the coordinator client
type zconn struct {
	client    *client.Client
	connC     chan chan<- client.Connection
	shutdownC chan struct{}
}

// newzconn instantiates a new connection listener
func newzconn(zclient *client.Client, path string) *zconn {
	zconn := &zconn{
		client:    zclient,
		connC:     make(chan chan<- client.Connection),
		shutdownC: make(chan struct{}),
	}

	go zconn.monitor(path)
	return zconn
}

// monitor checks for changes in a path-based connection
func (zconn *zconn) monitor(path string) {
	var (
		connC chan<- client.Connection
		conn  client.Connection
		err   error
	)

	defer func() {
		if conn != nil {
			conn.Close()
		}
	}()

	for {
		// wait for someone to request a connection, or shutdown
		select {
		case connC = <-zconn.connC:
		case <-zconn.shutdownC:
			return
		}

	retry:
		// create a connection if it doesn't exist or ping the existing connection
		if conn == nil {
			conn, err = zconn.client.GetCustomConnection(path)
			if err != nil {
				glog.Warningf("Could not obtain a connection to %s: %s", path, err)
			}
		} else if _, err := conn.Children("/"); err == client.ErrConnectionClosed {
			glog.Warningf("Could not ping connection to %s: %s", path, err)
			conn = nil
		}

		// send the connection back
		if conn != nil {
			connC <- conn
			continue
		}

		// if conn is nil, try to create a new connection
		select {
		case <-time.After(time.Second):
			glog.Infof("Refreshing connection to zookeeper")
			goto retry
		case <-zconn.shutdownC:
			return
		}
	}
}

// connect returns a connection object or times out trying
func (zconn *zconn) connect(timeout time.Duration) (client.Connection, error) {
	connC := make(chan client.Connection, 1)
	zconn.connC <- connC
	select {
	case conn := <-connC:
		return conn, nil
	case <-time.After(timeout):
		glog.Warningf("timed out waiting for connection")
		return nil, ErrTimeout
	case <-zconn.shutdownC:
		glog.Warningf("receieved signal to shutdown")
		return nil, ErrShutdown
	}
}

// shutdown stops the connection listener
func (zconn *zconn) shutdown() {
	close(zconn.shutdownC)
}
