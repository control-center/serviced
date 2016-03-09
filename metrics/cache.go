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

type MemoryUsageItem struct {
	value  []MemoryUsageStats
	expire <-chan time.Time
}

func (item *MemoryUsageItem) Expired() bool {
	select {
	case <-item.expire:
		return true
	default:
		return false
	}
}

// MemoryUsageCache is a simple TTL cache for MemoryUsageStats objects
type MemoryUsageCache struct {
	sync.Mutex
	Usages map[string]*MemoryUsageItem
	TTL    time.Duration
	Clock  utils.Clock
}

// Get retrieves a cached value if one exists; otherwise it calls getter, caches the result, and returns it
func (c *MemoryUsageCache) Get(key string, getter MemoryUsageQuery) (val []MemoryUsageStats, err error) {
	c.Lock()
	defer c.Unlock()
	item, ok := c.Usages[key]
	if ok {
		if item.Expired() {
			delete(c.Usages, key)
		} else {
			return item.value, nil
		}
	}
	if val, err = getter(); err != nil {
		return
	}
	c.Usages[key] = &MemoryUsageItem{
		value:  val,
		expire: c.Clock.After(c.TTL),
	}
	return
}

func NewMemoryUsageCache(ttl time.Duration) *MemoryUsageCache {
	return &MemoryUsageCache{
		Usages: make(map[string]*MemoryUsageItem),
		TTL:    ttl,
		Clock:  utils.RealClock,
	}
}
