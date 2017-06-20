// Copyright 2017 The Serviced Authors.
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

package service

import (
	"path"
)

// ZKPath can be used to build slash seperated paths to zookeeper nodes.
type ZKPath struct {
	path string
}

// Base returns the starting point for a zookeeper path which is a single
// forward slash.
func Base() *ZKPath {
	return &ZKPath{path: "/"}
}

// Path returns the slash seperated string representation for a zookeeper path.
func (p *ZKPath) Path() string {
	return p.path
}

// Pools appends the node name for pools to the zookeeper path.
func (p *ZKPath) Pools() *ZKPath {
	return p.concat("pools")
}

// VirtualIPs appends the node name for virtual IPs to the zookeeper path.
func (p *ZKPath) VirtualIPs() *ZKPath {
	return p.concat("virtualIPs")
}

// Hosts appends the node name for hosts to the zookeeper path.
func (p *ZKPath) Hosts() *ZKPath {
	return p.concat("hosts")
}

// Online appends the node name for online to the zookeeper path.
func (p *ZKPath) Online() *ZKPath {
	return p.concat("online")
}

// IPs appends the node name for IPs to the zookeeper path.
func (p *ZKPath) IPs() *ZKPath {
	return p.concat("ips")
}

// Locked appends the node name for locked to the zookeeper path.
func (p *ZKPath) Locked() *ZKPath {
	return p.concat("locked")
}

// ID appends the given id to the zookeeper path.  If the string is empty,
// the method will add nothing to the path.  If this behavior is not desired,
// then checks for a empty string should be done before this method is called.
func (p *ZKPath) ID(id string) *ZKPath {
	return p.concat(id)
}

func (p *ZKPath) concat(s string) *ZKPath {
	return &ZKPath{
		path: path.Join(p.path, s),
	}
}
