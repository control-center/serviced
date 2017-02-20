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

package virtualips

import (
	"errors"
	"path"
	"time"

	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/domain/pool"
	"github.com/control-center/serviced/zzk"
	"github.com/zenoss/glog"
)

const (
	zkVirtualIP            = "/virtualIPs"
	virtualInterfacePrefix = ":z"
	maxRetries             = 2
	waitTimeout            = 30 * time.Second
)

var (
	ErrInvalidVirtualIP = errors.New("invalid virtual ip")
)

func vippath(nodes ...string) string {
	p := append([]string{zkVirtualIP}, nodes...)
	return path.Join(p...)
}

type VirtualIPNode struct {
	*pool.VirtualIP
	version interface{}
}

// ID implements zzk.Node
func (node *VirtualIPNode) GetID() string {
	return node.IP
}

// Create implements zzk.Node
func (node *VirtualIPNode) Create(conn client.Connection) error {
	return AddVirtualIP(conn, node.VirtualIP)
}

// Update implements zzk.Node
func (node *VirtualIPNode) Update(conn client.Connection) error {
	return nil
}

func (node *VirtualIPNode) Version() interface{}           { return node.version }
func (node *VirtualIPNode) SetVersion(version interface{}) { node.version = version }

func SyncVirtualIPs(conn client.Connection, virtualIPs []pool.VirtualIP) error {
	nodes := make([]zzk.Node, len(virtualIPs))
	for i := range virtualIPs {
		nodes[i] = &VirtualIPNode{VirtualIP: &virtualIPs[i]}
	}
	return zzk.Sync(conn, nodes, vippath())
}

func AddVirtualIP(conn client.Connection, virtualIP *pool.VirtualIP) error {
	var node VirtualIPNode
	path := vippath(virtualIP.IP)

	glog.V(1).Infof("Adding virtual ip to zookeeper: %s", path)
	if err := conn.Create(path, &node); err != nil {
		return err
	}
	node.VirtualIP = virtualIP
	return conn.Set(path, &node)
}

func RemoveVirtualIP(conn client.Connection, ip string) error {
	glog.V(1).Infof("Removing virtual ip from zookeeper: %s", vippath(ip))
	err := conn.Delete(vippath(ip))
	if err == nil || err == client.ErrNoNode {
		return nil
	}
	return err
}

func GetHostID(conn client.Connection, poolid, ip string) (string, error) {
	basepth := "/"
	if poolid != "" {
		basepth = path.Join("/pools", poolid)
	}
	leader, err := conn.NewLeader(path.Join(basepth, "/virtualIPs", ip))
	if err != nil {
		return "", err
	}
	return zzk.GetHostID(leader)
}
