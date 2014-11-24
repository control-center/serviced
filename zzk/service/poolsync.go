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

package service

import (
	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/domain/pool"
	"github.com/control-center/serviced/zzk"
	"github.com/zenoss/glog"
)

// PoolSyncHandler is the handler for synchronizing ResourcePool data
type PoolSyncHandler interface {
	// GetResourcePools gets all resource pools
	GetResourcePools() ([]pool.ResourcePool, error)
	// AddUpdateResourcePool adds or updates a resource pool
	AddUpdateResourcePool(*pool.ResourcePool) error
	// RemoveResourcePool deletes a resource pool
	RemoveResourcePool(string) error
}

// PoolSynchronizer is the synchronizer for ResourcePool data
type PoolSynchronizer struct {
	handler       PoolSyncHandler
	getConnection zzk.GetConnection
}

// NewPoolSynchronizer initializes a new Synchronizer
func NewPoolSynchronizer(handler PoolSyncHandler, getConnection zzk.GetConnection) *zzk.Synchronizer {
	pSync := &PoolSynchronizer{handler, getConnection}
	return zzk.NewSynchronizer(pSync)
}

// Allocate implements zzk.SyncHandler
func (l *PoolSynchronizer) Allocate() zzk.Node { return &PoolNode{} }

// GetConnection implements zzk.SyncHandler
func (l *PoolSynchronizer) GetConnection(path string) (client.Connection, error) {
	return l.getConnection(path)
}

// GetPath implements zzk.SyncHandler
func (l *PoolSynchronizer) GetPath(nodes ...string) string { return poolpath(nodes...) }

// Ready implements zzk.SyncHandler
func (l *PoolSynchronizer) Ready() error { return nil }

// Done implements zzk.SyncHandler
func (l *PoolSynchronizer) Done() {}

// GetAll implements zzk.SyncHandler
func (l *PoolSynchronizer) GetAll() ([]zzk.Node, error) {
	pools, err := l.handler.GetResourcePools()
	if err != nil {
		return nil, err
	}

	nodes := make([]zzk.Node, len(pools))
	for i := range pools {
		nodes[i] = &PoolNode{ResourcePool: &pools[i]}
	}
	return nodes, nil
}

// AddUpdate implements zzk.SyncHandler
func (l *PoolSynchronizer) AddUpdate(id string, node zzk.Node) (string, error) {
	if pnode, ok := node.(*PoolNode); !ok {
		glog.Errorf("Could not extract pool node data for %s", id)
		return "", zzk.ErrInvalidType
	} else if pool := pnode.ResourcePool; pool == nil {
		glog.Errorf("Pool is nil for %s", id)
		return "", zzk.ErrInvalidType
	} else if err := l.handler.AddUpdateResourcePool(pool); err != nil {
		return "", err
	}

	return node.GetID(), nil
}

// Delete implements zzk.SyncHandler
func (l *PoolSynchronizer) Delete(id string) error {
	return l.handler.RemoveResourcePool(id)
}
