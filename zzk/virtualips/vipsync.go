// Copyright 2015 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
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

	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/domain/pool"
	"github.com/control-center/serviced/utils"
)

func poolpath(nodes ...string) string {
	return path.Join(append([]string{"/pools"}, nodes...)...)
}

// VirtualIPSync is the zookeeper synchronization object for vips
type VirtualIPSync struct {
	conn   client.Connection // the global client connector
	poolID string
}

// SyncVirtualIPs performs the vip synchronization
func SyncVirtualIPs(conn client.Connection, poolID string, vips []pool.VirtualIP) error {
	vipmap := make(map[string]interface{})
	for i, vip := range vips {
		vipmap[vip.IP] = &vips[i]
	}
	return utils.Sync(&VirtualIPSync{conn, poolID}, vipmap)
}

// IDs returns the current data by id
func (sync *VirtualIPSync) IDs() ([]string, error) {
	if ids, err := sync.conn.Children(poolpath(sync.poolID, vippath())); err == client.ErrNoNode {
		return []string{}, nil
	} else {
		return ids, err
	}
}

// Create creates the new object data
func (sync *VirtualIPSync) Create(data interface{}) error {
	path, vip, err := sync.convert(data)
	if err != nil {
		return err
	}
	var node VirtualIPNode
	if err := sync.conn.Create(path, &node); err != nil {
		return err
	}
	node.VirtualIP = vip
	return sync.conn.Set(path, &node)
}

// Update updates the existing object data
func (sync *VirtualIPSync) Update(data interface{}) error {
	path, vip, err := sync.convert(data)
	if err != nil {
		return err
	}
	var node VirtualIPNode
	if err := sync.conn.Get(path, &node); err != nil {
		return err
	}
	node.VirtualIP = vip
	return sync.conn.Set(path, &node)
}

// Delete deletes an existing vip by its id
func (sync *VirtualIPSync) Delete(id string) error {
	return sync.conn.Delete(poolpath(sync.poolID, vippath(id)))
}

// convert transforms the generic object data into its proper type
func (sync *VirtualIPSync) convert(data interface{}) (string, *pool.VirtualIP, error) {
	if vip, ok := data.(*pool.VirtualIP); ok {
		return poolpath(sync.poolID, vippath(vip.IP)), vip, nil
	}
	return "", nil, errors.New("could not convert to a virtual ip object")
}