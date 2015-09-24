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

type Clock interface {
	After(d time.Duration) <-chan time.Time
}

type realClock struct{}

func (realClock) After(d time.Duration) <-chan time.Time {
	return time.After(d)
}

type MemoryUsageQuery func() (*[]MemoryUsageStats, error)

type MemoryUsageCache struct {
	sync.Mutex
	Locks  map[string]sync.Mutex
	Usages map[string]*[]MemoryUsageStats
	TTL    time.Duration
	Clock  Clock
}

func (c *MemoryUsageCache) gethostlock(key string) *sync.Mutex {
	c.Lock()
	defer c.Unlock()
	var (
		lock sync.Mutex
		ok   bool
	)
	if lock, ok = c.Locks[key]; !ok {
		lock = sync.Mutex{}
		c.Locks[key] = lock
	}
	return &lock
}

func (c *MemoryUsageCache) Get(key string, getter MemoryUsageQuery) (val *[]MemoryUsageStats, err error) {
	var ok bool
	l := c.gethostlock(key)
	l.Lock()
	defer l.Unlock()

	if val, ok = c.Usages[key]; !ok {
		if val, err = getter(); err != nil {
			return
		}
		c.Usages[key] = val
		go func() {
			<-c.Clock.After(c.TTL)
			l := c.gethostlock(key)
			l.Lock()
			defer l.Unlock()
			delete(c.Usages, key)
		}()
	}
	return
}

func NewMemoryUsageCache(ttl time.Duration) *MemoryUsageCache {
	return &MemoryUsageCache{
		Locks:  make(map[string]sync.Mutex),
		Usages: make(map[string]*[]MemoryUsageStats),
		TTL:    ttl,
		Clock:  realClock{},
	}
}
