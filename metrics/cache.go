// Copyright 2015 The Serviced Authors.
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

package metrics

import (
	"sync"
	"time"

	"github.com/control-center/serviced/utils"
)

// MemoryUsageQuery is a function that will return a value to be cached in the event of a miss
type MemoryUsageQuery func() ([]MemoryUsageStats, error)

// MemoryUsageCache is a simple TTL cache for MemoryUsageStats objects
type MemoryUsageCache struct {
	sync.Mutex
	Locks  map[string]*sync.Mutex
	Usages map[string][]MemoryUsageStats
	TTL    time.Duration
	Clock  utils.Clock
}

// getkeylock returns a lock specifically for this key
func (c *MemoryUsageCache) getkeylock(key string) *sync.Mutex {
	if _, ok := c.Locks[key]; !ok {
		c.Locks[key] = &sync.Mutex{}
	}
	return c.Locks[key]
}

// Get retrieves a cached value if one exists; otherwise it calls getter, caches the result, and returns it
func (c *MemoryUsageCache) Get(key string, getter MemoryUsageQuery) (val []MemoryUsageStats, err error) {
	var ok bool
	// Acquire a lock for this key to update or not
	l := c.getkeylock(key)
	l.Lock()
	defer l.Unlock()

	if val, ok = c.Usages[key]; !ok {
		if val, err = getter(); err != nil {
			return
		}
		c.Usages[key] = val
		// Start the expiration
		go func() {
			<-c.Clock.After(c.TTL)
			c.Lock()
			defer c.Unlock()
			delete(c.Usages, key)
		}()
	}
	return
}

func NewMemoryUsageCache(ttl time.Duration) *MemoryUsageCache {
	return &MemoryUsageCache{
		Locks:  make(map[string]*sync.Mutex),
		Usages: make(map[string][]MemoryUsageStats),
		TTL:    ttl,
		Clock:  utils.RealClock,
	}
}
