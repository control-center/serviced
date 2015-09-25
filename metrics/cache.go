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
)

// Clock is an abstraction of a clock, for testing purposes
type Clock interface {
	After(d time.Duration) <-chan time.Time
}

type realClock struct{}

func (realClock) After(d time.Duration) <-chan time.Time {
	return time.After(d)
}

// MemoryUsageQuery is a function that will return a value to be cached in the event of a miss
type MemoryUsageQuery func() ([]MemoryUsageStats, error)

// MemoryUsageCache is a simple TTL cache for MemoryUsageStats objects
type MemoryUsageCache struct {
	sync.Mutex
	Locks  map[string]sync.Mutex
	Usages map[string][]MemoryUsageStats
	TTL    time.Duration
	Clock  Clock
}

// getkeylock returns a lock specifically for this key
func (c *MemoryUsageCache) getkeylock(key string) *sync.Mutex {
	var (
		lock sync.Mutex
		ok   bool
	)
	c.Lock()
	defer c.Unlock()
	if lock, ok = c.Locks[key]; !ok {
		lock = sync.Mutex{}
		c.Locks[key] = lock
	}
	return &lock
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
		Locks:  make(map[string]sync.Mutex),
		Usages: make(map[string][]MemoryUsageStats),
		TTL:    ttl,
		Clock:  realClock{},
	}
}
