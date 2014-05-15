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

func (n *Node) Version() interface{} {
	return n.version
}

func (n *Node) SetVersion(version interface{}) {
	n.version = version
}
