package storage

import (
	"github.com/zenoss/serviced/domain/host"
)

// Node is a server that participate in serviced storage as a server or client
type Node struct {
	host.Host
	Network    string
	ExportPath string
	version    interface{}
}

// Version returns the node version to implement the client.Node interface
func (n *Node) Version() interface{} {
	return n.version
}

// SetVersion sets the node version to implement the client.Node interface
func (n *Node) SetVersion(version interface{}) {
	n.version = version
}
