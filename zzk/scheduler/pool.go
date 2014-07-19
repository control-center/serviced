package scheduler

import (
	"path"

	"github.com/zenoss/serviced/coordinator/client"
	"github.com/zenoss/serviced/domain/pools"
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
func (node *PoolNode) ID() string {
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

// Delete implements zzk.Node
func (node *PoolNode) Delete(conn client.Connection) error {
	return RemoveResourcePool(conn, node.ID)
}

func (node *PoolNode) Version() interface{}           { return node.version }
func (node *PoolNode) SetVersion(version interface{}) { node.version = version }


func SyncResourcePools(conn client.Connection, pools []*pool.ResourcePool) error {
	nodes := make([]*PoolNode, len(pools))
	for i := range pools {
		nodes[i] = &HostNode{ResourcePool:pools[i]}
	}
	return zzk.Sync(conn, nodes, poolpath())
}

func AddResourcePool(conn client.Connection, poolID string) error {
	return conn.CreateDir(poolpath(poolID))
}

func RemoveResourcePool(conn client.Connection, poolID string) error {
	return conn.Delete(poolpath(poolID))
}