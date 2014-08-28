// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package storage

import (
	"github.com/control-center/serviced/domain/host"
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
