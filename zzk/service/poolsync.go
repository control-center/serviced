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
	"github.com/control-center/serviced/domain/pool"
	"github.com/control-center/serviced/utils"
)

// PoolSync is the zookeeper synchronization object for pools
type PoolSync struct {
	conn client.Connection // the global client connector
}

// SyncPools performs the pool synchronization
func SyncPools(conn client.Connection, pools []pool.ResourcePool) error {
	poolmap := make(map[string]interface{})
	for i, pool := range pools {
		poolmap[pool.ID] = &pools[i]
	}
	return utils.Sync(&PoolSync{conn}, poolmap)
}

// IDs returns the current data by id
func (sync *PoolSync) IDs() ([]string, error) {
	if ids, err := sync.conn.Children(poolpath()); err == client.ErrNoNode {
		return []string{}, nil
	} else {
		return ids, err
	}
}

// Create creates the new object data
func (sync *PoolSync) Create(data interface{}) error {
	path, pool, err := sync.convert(data)
	if err != nil {
		return err
	}
	var node PoolNode
	if err := sync.conn.Create(path, &node); err != nil {
		return err
	}
	node.ResourcePool = pool
	return sync.conn.Set(path, &node)
}

// Update updates the existing object data
func (sync *PoolSync) Update(data interface{}) error {
	path, pool, err := sync.convert(data)
	if err != nil {
		return err
	}
	var node PoolNode
	if err := sync.conn.Get(path, &node); err != nil {
		return err
	}
	node.ResourcePool = pool
	return sync.conn.Set(path, &node)
}

// Delete deletes an existing pool by its id
func (sync *PoolSync) Delete(id string) error {
	return sync.conn.Delete(poolpath(id))
}

// convert transforms the generic object data into its proper type
func (sync *PoolSync) convert(data interface{}) (string, *pool.ResourcePool, error) {
	if pool, ok := data.(*pool.ResourcePool); ok {
		return poolpath(pool.ID), pool, nil
	}
	return "", nil, errors.New("could not convert data to a resource pool object")
}