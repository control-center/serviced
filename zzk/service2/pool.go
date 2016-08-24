// Copyright 2016 The Serviced Authors.
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

	log "github.com/Sirupsen/logrus"
	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/domain/pool"
)

// PoolNode is the storage object for resource pool data
type PoolNode struct {
	*pool.ResourcePool
	version interface{}
}

// Version implements client.Node
func (p *PoolNode) Version() interface{} {
	return p.version
}

// SetVersion implements client.Node
func (p *PoolNode) SetVersion(version interface{}) {
	p.version = version
}

// UpdateResourcePool creates the resource pool if it doesn't exist or updates
// it if it does exist.
func UpdateResourcePool(conn client.Connection, p pool.ResourcePool) error {
	pth := path.Join("/pools", p.ID)

	logger := plog.WithFields(log.Fields{
		"poolid": p.ID,
		"zkpath": pth,
	})

	// create the resource pool if it doesn't exist
	if err := conn.Create(pth, &PoolNode{ResourcePool: &p}); err == client.ErrNodeExists {

		// the node exists, so get it and update it
		node := &PoolNode{ResourcePool: &pool.ResourcePool{}}
		if err := conn.Get(pth, node); err != nil && err != client.ErrEmptyNode {

			logger.WithError(err).Debug("Could not get resource pool entry from zookeeper")
			return err
		}

		node.ResourcePool = &p
		if err := conn.Set(pth, node); err != nil {

			logger.WithError(err).Debug("Could not update resource pool entry in zookeeper")
			return err
		}

		logger.Debug("Updated entry for resource pool in zookeeper")
		return nil
	} else if err != nil {

		logger.WithError(err).Debug("Could not create resource pool entry in zookeeper")
		return err
	}

	logger.Debug("Created entry for resource pool in zookeeper")
	return nil
}

// RemoveResourcePool removes the resource pool
func RemoveResourcePool(conn client.Connection, poolid string) error {
	pth := path.Join("/pools", poolid)

	logger := plog.WithFields(log.Fields{
		"poolid": poolid,
		"zkpath": pth,
	})

	if err := conn.Delete(pth); err != nil {

		logger.WithError(err).Debug("Could not delete resource pool entry from zookeeper")
		return err
	}

	logger.Debug("Deleted resource pool entry from zookeeper")
	return nil
}

// SyncResourcePools synchronizes the resource pools to the provided list
func SyncResourcePools(conn client.Connection, pools []pool.ResourcePool) error {
	pth := path.Join("/pools")

	logger := plog.WithField("zkpath", pth)

	// look up the children pool ids
	ch, err := conn.Children(pth)
	if err != nil && err != client.ErrNoNode {
		logger.WithError(err).Debug("Could not look up resource pools")
		return err
	}

	// store the pool ids in a hash map
	chmap := make(map[string]struct{})
	for _, poolid := range ch {
		chmap[poolid] = struct{}{}
	}

	// set the resource pools
	for _, p := range pools {
		if err := UpdateResourcePool(conn, p); err != nil {
			return err
		}

		// delete matching records
		if _, ok := chmap[p.ID]; ok {
			delete(chmap, p.ID)
		}
	}

	// remove any leftovers
	for poolid := range chmap {
		if err := RemoveResourcePool(conn, poolid); err != nil {
			return err
		}
	}
	return nil
}
