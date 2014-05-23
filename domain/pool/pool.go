// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package pool

import (
	"reflect"
	"sort"
	"time"
)

type VirtualIP struct {
	PoolID        string
	IP            string
	Netmask       string
	BindInterface string
}

// ResourcePool A collection of computing resources with optional quotas.
type ResourcePool struct {
	ID               string      // Unique identifier for resource pool, eg "default"
	Description      string      // Description of the resource pool
	ParentID         string      // The pool id of the parent pool, if this pool is embeded in another pool. An empty string means it is not embeded.
	VirtualIPs       []VirtualIP // All virtual IPs associated with a pool
	Priority         int         // relative priority of resource pools, used for CPU priority
	CoreLimit        int         // Number of cores on the host available to serviced
	MemoryLimit      uint64      // A quota on the amount (bytes) of RAM in the pool, 0 = unlimited
	CoreCapacity     int         // Number of cores available as a sum of all cores on all hosts in the pool
	MemoryCapacity   uint64      // Amount (bytes) of RAM available as a sum of all memory on all hosts in the pool
	MemoryCommitment uint64      // Amount (bytes) of RAM committed to services
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type ByIP []VirtualIP

func (b ByIP) Len() int           { return len(b) }
func (b ByIP) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
func (b ByIP) Less(i, j int) bool { return b[i].IP < b[j].IP } // sort by IP address

func (a *ResourcePool) VirtualIPsEqual(b *ResourcePool) bool {
	// DeepEqual requires the order to be identical, therefore, sort!
	sort.Sort(ByIP(a.VirtualIPs))
	sort.Sort(ByIP(b.VirtualIPs))
	if reflect.DeepEqual(a.VirtualIPs, b.VirtualIPs) {
		return true
	}
	return false
}

// Equal returns true if two resource pools are equal
func (a *ResourcePool) Equals(b *ResourcePool) bool {
	if a.ID != b.ID {
		return false
	}
	if a.Description != b.Description {
		return false
	}
	if a.ParentID != b.ParentID {
		return false
	}
	if !a.VirtualIPsEqual(b) {
		return false
	}
	if a.Priority != b.Priority {
		return false
	}
	if a.CoreLimit != b.CoreLimit {
		return false
	}
	if a.MemoryLimit != b.MemoryLimit {
		return false
	}
	if a.CoreCapacity != b.CoreCapacity {
		return false
	}
	if a.MemoryCapacity != b.MemoryCapacity {
		return false
	}
	if a.MemoryCommitment != b.MemoryCommitment {
		return false
	}
	if a.CreatedAt.Unix() != b.CreatedAt.Unix() {
		return false
	}
	if a.UpdatedAt.Unix() != b.CreatedAt.Unix() {
		return false
	}

	return true
}

// New creates new ResourcePool
func New(id string) *ResourcePool {
	pool := &ResourcePool{}
	pool.ID = id

	return pool
}
