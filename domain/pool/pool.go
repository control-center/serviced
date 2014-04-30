// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package pool

import "time"

type VirtualIP struct {
	ID            string
	PoolID        string
	IP            string
	Netmask       string
	BindInterface string
	Index         string
}

// ResourcePool A collection of computing resources with optional quotas.
type ResourcePool struct {
	ID          string // Unique identifier for resource pool, eg "default"
	Description string // Description of the resource pool
	ParentID    string // The pool id of the parent pool, if this pool is embeded in another pool. An empty string means it is not embeded.
	VirtualIPs  []VirtualIP
	Priority    int    // relative priority of resource pools, used for CPU priority
	CoreLimit   int    // Number of cores on the host available to serviced
	MemoryLimit uint64 // A quota on the amount (bytes) of RAM in the pool, 0 = unlimited
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// New creates new ResourcePool
func New(id string) *ResourcePool {
	pool := &ResourcePool{}
	pool.ID = id
	return pool
}
