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

import "time"

// ReadPool is the read model for resource pools which contains properties for viewing
// information about resource pools.
type ReadPool struct {
	ID                string     // Unique identifier for resource pool, eg "default"
	Description       string     // Description of the resource pool
	CoreCapacity      int        // Sum of all cores on all hosts in the pool
	MemoryCapacity    uint64     // Sum of all RAM available (bytes) on all hosts in the pool
	MemoryCommitment  uint64     // Sum of RAM committed (bytes) to services in the pool
	ConnectionTimeout int        // Wait delay on service rescheduling when an outage is reported (milliseconds)
	CreatedAt         time.Time  // When the pool was created
	UpdatedAt         time.Time  // When the poool was last updated
	Permissions       Permission // A bitset of pemissions for this pool's hosts
}
