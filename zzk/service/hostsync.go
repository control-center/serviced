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

package service

import (
	"errors"

	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/utils"
)

// HostSync is the zookeeper synchronization object for hosts
type HostSync struct {
	conn   client.Connection // the global client connector
	poolID string
}

// SyncHosts performs the host synchronization
func SyncHosts(conn client.Connection, poolID string, hosts []host.Host) error {
	hostmap := make(map[string]interface{})
	for i, host := range hosts {
		hostmap[host.ID] = &hosts[i]
	}
	return utils.Sync(&HostSync{conn, poolID}, hostmap)
}

// IDs returns the current data by id
func (sync *HostSync) IDs() ([]string, error) {
	return sync.conn.Children(poolpath(sync.poolID, hostpath()))
}

// Create creates the new object data
func (sync *HostSync) Create(data interface{}) error {
	path, host, err := sync.convert(data)
	if err != nil {
		return err
	}
	var node HostNode
	if err := sync.conn.Create(path, &node); err != nil {
		return err
	}
	node.Host = host
	return sync.conn.Set(path, &node)
}

// Update updates the existing object data
func (sync *HostSync) Update(data interface{}) error {
	path, host, err := sync.convert(data)
	if err != nil {
		return err
	}
	var node HostNode
	if err := sync.conn.Get(path, &node); err != nil {
		return err
	}
	node.Host = host
	return sync.conn.Set(path, &node)
}

// Delete deletes an existing host by its id
func (sync *HostSync) Delete(id string) error {
	return sync.conn.Delete(poolpath(sync.poolID, hostpath(id)))
}

// convert transforms the generic object data into its proper type
func (sync *HostSync) convert(data interface{}) (string, *host.Host, error) {
	if host, ok := data.(*host.Host); ok {
		return poolpath(sync.poolID, hostpath(host.ID)), host, nil
	}
	return "", nil, errors.New("invalid type")
}