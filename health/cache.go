// Copyright 2016 The Serviced Authors.
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

package health

import (
	"sync"
	"time"
)

// HealthStatusKey is the key to the health status item in the cache.
type HealthStatusKey struct {
	ServiceID       string
	InstanceID      int
	HealthCheckName string
}

// HealthStatusItem is an item stored in the health status cache.
type HealthStatusItem struct {
	value   HealthStatus
	expires time.Time
}

// Value returns the HealthStatus data
func (item *HealthStatusItem) Value() HealthStatus {
	return item.value
}

// Expired returns true when a health status item has expired in the cache.
func (item *HealthStatusItem) Expired() bool {
	return time.Now().After(item.expires)
}

// HealthStatusCache keeps track of the health status items in memory.
type HealthStatusCache struct {
	mu   *sync.Mutex
	data map[HealthStatusKey]HealthStatusItem
	stop chan struct{}
}

// New returns a new HealthStatusCache instance
func New() *HealthStatusCache {
	cache := &HealthStatusCache{
		mu:   &sync.Mutex{},
		data: make(map[HealthStatusKey]HealthStatusItem),
	}
	return cache
}

// SetPurgeFrequency sets the autopurge interval for cache cleanup.
// Stops autopurge if interval is <= 0.
func (cache *HealthStatusCache) SetPurgeFrequency(interval time.Duration) {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	if cache.stop != nil {
		close(cache.stop)
	}
	if interval > 0 {
		cache.stop = make(chan struct{})
		go func(stop <-chan struct{}) {
			timer := time.NewTimer(interval)
			defer timer.Stop()
			for {
				select {
				case <-timer.C:
					cache.DeleteExpired()
					timer.Reset(interval)
				case <-stop:
					return
				}
			}
		}(cache.stop)
	} else {
		cache.stop = nil
	}
	return
}

// Size returns the size of the cache.
func (cache *HealthStatusCache) Size() int {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	return len(cache.data)
}

// Get returns an item from the cache if it hasn't yet expired.
func (cache *HealthStatusCache) Get(key HealthStatusKey) (HealthStatus, bool) {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	item, ok := cache.get(key)
	return item.Value(), ok
}

// get is non thread-safe
func (cache *HealthStatusCache) get(key HealthStatusKey) (item HealthStatusItem, ok bool) {
	if item, ok = cache.data[key]; ok {
		if item.Expired() {
			cache.delete(key)
			return HealthStatusItem{}, false
		}
	}
	return
}

// Set sets an item into the cache.
func (cache *HealthStatusCache) Set(key HealthStatusKey, value HealthStatus, expire time.Duration) {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	cache.set(key, value, time.Now().Add(expire))
}

// set is non thread-safe
func (cache *HealthStatusCache) set(key HealthStatusKey, value HealthStatus, expires time.Time) {
	cache.data[key] = HealthStatusItem{value: value, expires: expires}
}

// Delete removes an item from the cache.
func (cache *HealthStatusCache) Delete(key HealthStatusKey) {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	if _, ok := cache.get(key); ok {
		cache.delete(key)
	}
}

// DeleteExpired removes all expired items from the cache.
func (cache *HealthStatusCache) DeleteExpired() {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	for key, item := range cache.data {
		if item.Expired() {
			cache.delete(key)
		}
	}
}

// DeleteInstance removes all health checks per instance.
func (cache *HealthStatusCache) DeleteInstance(serviceID string, instanceID int) {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	for key := range cache.data {
		if key.ServiceID == serviceID && key.InstanceID == instanceID {
			cache.delete(key)
		}
	}
}

// delete is non thread-safe
func (cache *HealthStatusCache) delete(key HealthStatusKey) {
	delete(cache.data, key)
}
