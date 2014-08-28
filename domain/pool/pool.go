// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package pool

import (
	"github.com/control-center/serviced/domain"

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
	ID                string      // Unique identifier for resource pool, eg "default"
	Description       string      // Description of the resource pool
	ParentID          string      // The pool id of the parent pool, if this pool is embeded in another pool. An empty string means it is not embeded.
	VirtualIPs        []VirtualIP // All virtual IPs associated with a pool
	Priority          int         // relative priority of resource pools, used for CPU priority
	CoreLimit         int         // Number of cores on the host available to serviced
	MemoryLimit       uint64      // A quota on the amount (bytes) of RAM in the pool, 0 = unlimited
	CoreCapacity      int         // Number of cores available as a sum of all cores on all hosts in the pool
	MemoryCapacity    uint64      // Amount (bytes) of RAM available as a sum of all memory on all hosts in the pool
	MemoryCommitment  uint64      // Amount (bytes) of RAM committed to services
	CreatedAt         time.Time
	UpdatedAt         time.Time
	MonitoringProfile domain.MonitorProfile
}

type ByIP []VirtualIP

func (b ByIP) Len() int           { return len(b) }
func (b ByIP) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
func (b ByIP) Less(i, j int) bool { return b[i].IP < b[j].IP } // sort by IP address

func (a *ResourcePool) VirtualIPsEqual(b *ResourcePool) bool {
	// create a deep copy in order to avoid side effects (leaving the VirtualIPs slice sorted)
	aVIPs := make([]VirtualIP, len(a.VirtualIPs))
	bVIPs := make([]VirtualIP, len(b.VirtualIPs))
	copy(aVIPs, a.VirtualIPs)
	copy(bVIPs, b.VirtualIPs)
	// DeepEqual requires the order to be identical, therefore, sort!
	sort.Sort(ByIP(aVIPs))
	sort.Sort(ByIP(bVIPs))
	return reflect.DeepEqual(aVIPs, bVIPs)
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
	if !a.MonitoringProfile.Equals(&b.MonitoringProfile) {
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
