// Copyright 2014 The Serviced Authors.
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

package pool

import (
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain"
	"github.com/control-center/serviced/domain/host"

	"fmt"
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

type Permission uint

const (
	AdminAccess Permission = 1 << iota
	DFSAccess
)

// ResourcePool A collection of computing resources with optional quotas.
type ResourcePool struct {
	ID                string      // Unique identifier for resource pool, eg "default"
	Realm             string      // The name of the realm where this pool resides
	Description       string      // Description of the resource pool
	VirtualIPs        []VirtualIP // All virtual IPs associated with a pool
	CoreLimit         int         // Number of cores on the host available to serviced
	MemoryLimit       uint64      // A quota on the amount (bytes) of RAM in the pool, 0 = unlimited
	CoreCapacity      int         // Number of cores available as a sum of all cores on all hosts in the pool
	MemoryCapacity    uint64      // Amount (bytes) of RAM available as a sum of all memory on all hosts in the pool
	MemoryCommitment  uint64      // Amount (bytes) of RAM committed to services
	ConnectionTimeout int         // Wait delay on service rescheduling when an outage is reported (milliseconds)
	CreatedAt         time.Time
	UpdatedAt         time.Time
	MonitoringProfile domain.MonitorProfile
	Permissions       Permission
	datastore.VersionedEntity
}

func (p ResourcePool) ToString() string {
	permissionStr := ""
	switch p.Permissions {
	case 0:
		permissionStr = "No Permissions"
	case 1:
		permissionStr = "DFS Permissions Only"
	case 2:
		permissionStr = "Admin Permissions Only"
	case 3:
		permissionStr = "DFS and Admin Permissions"
	}
	pStr := ""
	pStr += fmt.Sprintf("%s\n", p.ID)
	pStr += fmt.Sprintf("%-20s%s\n", "Realm:", p.Realm)
	pStr += fmt.Sprintf("%-20s%s\n", "Description:", p.Description)
	pStr += fmt.Sprintf("%-20s", "Virtual IPs:")
	if len(p.VirtualIPs) > 0 {
		pStr += fmt.Sprintf("\n")
		for _, vIP := range p.VirtualIPs {
			pStr += fmt.Sprintf("  %-20s%s\n", "Bind Interface:", vIP.BindInterface)
			pStr += fmt.Sprintf("  %-20s%s\n", "IP:", vIP.IP)
			pStr += fmt.Sprintf("  %-20s%s\n", "Netmask:", vIP.Netmask)
		}
	} else {
		pStr += fmt.Sprintf("None\n")
	}
	pStr += fmt.Sprintf("%-20s%d\n", "Core Limit:", p.CoreLimit)
	pStr += fmt.Sprintf("%-20s%d\n", "Memory Limit:", p.MemoryLimit)
	pStr += fmt.Sprintf("%-20s%d\n", "Core Capacity:", p.CoreCapacity)
	pStr += fmt.Sprintf("%-20s%d\n", "Memory Capacity:", p.MemoryCapacity)
	pStr += fmt.Sprintf("%-20s%d\n", "Memory Commitment:", p.MemoryCommitment)
	pStr += fmt.Sprintf("%-20s%d\n", "Connection Timout:", p.ConnectionTimeout)
	pStr += fmt.Sprintf("%-20s%s\n", "Created At:", p.CreatedAt)
	pStr += fmt.Sprintf("%-20s%s\n", "Updated At:", p.UpdatedAt)
	pStr += fmt.Sprintf("%-20s%s\n", "Permissions:", permissionStr)
	return pStr
}

func (p ResourcePool) GetConnectionTimeout() time.Duration {
	return time.Duration(p.ConnectionTimeout) * time.Millisecond
}

// PoolIPs type for IP resources available in a ResourcePool
type PoolIPs struct {
	PoolID     string
	HostIPs    []host.HostIPResource
	VirtualIPs []VirtualIP
}

// An association between a host and a pool.
type PoolHost struct {
	HostID string
	PoolID string
	HostIP string
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
	if a.Realm != b.Realm {
		return false
	}
	if a.Description != b.Description {
		return false
	}
	if !a.VirtualIPsEqual(b) {
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
	if a.UpdatedAt.Unix() != b.UpdatedAt.Unix() {
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

func (a *ResourcePool) HasDfsAccess() bool {
	return a.Permissions&DFSAccess != 0
}

func (a *ResourcePool) HasAdminAccess() bool {
	return a.Permissions&AdminAccess != 0
}
