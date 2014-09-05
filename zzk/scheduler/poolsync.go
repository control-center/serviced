// Copyright 2014 The Serviced Authors.
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
package scheduler

import (
	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/domain/pool"
	"github.com/control-center/serviced/zzk"
)

type PoolSyncHandler interface {
	GetPathBasedConnection(path string) (client.Connection, error)
	GetResourcePools() ([]*pool.ResourcePool, error)
	AddOrUpdateResourcePool(pool *pool.ResourcePool) error
	RemoveResourcePool(poolID string) error
}

type PoolSyncListener struct {
	handler PoolSyncHandler
}

func NewPoolSyncListener(conn client.Connection, handler PoolSyncHandler) *zzk.SyncListener {
	poolSync := &PoolSyncListener{handler}
	return zzk.NewSyncListener(conn, poolSync)
}

func (l *PoolSyncListener) GetPathBasedConnection(path string) (client.Connection, error) {
	return l.handler.GetPathBasedConnection(path)
}

func (l *PoolSyncListener) GetPath(nodes ...string) string { return poolpath(nodes...) }

func (l *PoolSyncListener) GetAll() ([]zzk.Node, error) {
	pools, err := l.handler.GetResourcePools()
	if err != nil {
		return nil, err
	}

	nodes := make([]zzk.Node, len(pools))
	for i, pool := range pools {
		nodes[i] = &PoolNode{ResourcePool: pool}
	}

	return nodes, nil
}

func (l *PoolSyncListener) AddOrUpdate(id string, node zzk.Node) error {
	return l.handler.AddOrUpdateResourcePool(node.(*PoolNode).ResourcePool)
}

func (l *PoolSyncListener) Delete(poolID string) error {
	return l.handler.RemoveResourcePool(poolID)
}
