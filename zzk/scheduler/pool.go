// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.
package scheduler

import (
	"path"

	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/domain/pool"
	"github.com/control-center/serviced/zzk"
)

const (
	zkPool = "/pools"
)

func poolpath(nodes ...string) string {
	p := append([]string{zkPool}, nodes...)
	return path.Join(p...)
}

type PoolNode struct {
	*pool.ResourcePool
	version interface{}
}

// ID implements zzk.Node
func (node *PoolNode) GetID() string {
	return node.ID
}

// Create implements zzk.Node
func (node *PoolNode) Create(conn client.Connection) error {
	return AddResourcePool(conn, node.ID)
}

// Update implements zzk.Node
func (node *PoolNode) Update(conn client.Connection) error {
	return nil
}

func (node *PoolNode) Version() interface{}           { return node.version }
func (node *PoolNode) SetVersion(version interface{}) { node.version = version }

func SyncResourcePools(conn client.Connection, pools []*pool.ResourcePool) error {
	nodes := make([]zzk.Node, len(pools))
	for i := range pools {
		nodes[i] = &PoolNode{ResourcePool: pools[i]}
	}
	return zzk.Sync(conn, nodes, poolpath())
}

func AddResourcePool(conn client.Connection, poolID string) error {
	return conn.CreateDir(poolpath(poolID))
}

func RemoveResourcePool(conn client.Connection, poolID string) error {
	return conn.Delete(poolpath(poolID))
}