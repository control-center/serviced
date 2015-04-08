// Copyright 2014 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package storage

import (
	"github.com/control-center/serviced/domain/host"
)

// Node is a server that participate in serviced storage as a server or client
type Node struct {
	host.Host
	Network    string
	ExportPath string
	ExportTime string
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
