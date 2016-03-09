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
	"path"
	"strings"
	"sync"
	"time"
)

// HealthStatusKey is the key to the health status item in the cache.
type HealthStatusKey struct {
	ServiceID       string
	InstanceID      int
	HealthCheckName string
}

// Key returns the string value of the health check key as it lives in the
// cache.
func (key *HealthStatusKey) String() string {
	return path.Join(key.ServiceID, string(key.InstanceID), key.HealthCheckName)
}

// HealthStatusItem is an item stored in the health status cache.
type HealthStatusItem struct {
	value   *HealthStatus
	expires time.Time
}

// Value returns the HealthStatus data
func (item *HealthStatusItem) Value() *HealthStatus {
	return item.value
}

// Expired returns true when a health status item has expired in the cache.
func (item *HealthStatusItem) Expired() bool {
	return time.Now().After(item.expires)
}

// HealthStatusCache keeps track of health status items in memory.
type HealthStatusCache struct {
	mu   *sync.Mutex
	data map[string]HealthStatusItem
}

func Initialize(cancel <-chan interface{}, autoPurge time.Duration) *HealthStatusCache {
	cache := &HealthStatusCache{
		mu:   &sync.Mutex{},
		data: make(map[string]HealthStatusItem),
	}

	// start the auto cache cleanup
	go func() {
		timer := time.NewTimer(autoPurge)
		defer timer.Stop()
		for {
			select {
			case <-timer.C:
				cache.DeleteExpired()
				timer.Reset(autoPurge)
			case <-cancel:
				return
			}
		}
	}()

	return cache
}

// Get returns an item from the cache if it hasn't yet expired.
func (cache *HealthStatusCache) Get(key HealthStatusKey) (*HealthStatus, bool) {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	item, ok := cache.get(key.String())
	return item.value, ok
}

// get is non-thread-safe
func (cache *HealthStatusCache) get(key string) (item HealthStatusItem, ok bool) {
	if item, ok = cache.data[key]; ok {
		if item.Expired() {
			cache.delete(key)
			return HealthStatusItem{}, false
		}
	}
	return
}

// Set sets an item into the cache
func (cache *HealthStatusCache) Set(key HealthStatusKey, value *HealthStatus, expire time.Duration) {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	cache.set(key.String(), value, time.Now().Add(expire))
}

// set is non-thread-safe
func (cache *HealthStatusCache) set(key string, value *HealthStatus, expires time.Time) {
	cache.data[key] = HealthStatusItem{value, expires}
}

// Delete removes an item from the cache
func (cache *HealthStatusCache) Delete(key HealthStatusKey) {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	k := key.String()
	if _, ok := cache.get(k); ok {
		cache.delete(k)
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
	prefix := path.Join(serviceID, string(instanceID)) + "/"
	for key := range cache.data {
		if strings.HasPrefix(key, prefix) {
			cache.delete(key)
		}
	}
}

// delete is non-thread-safe
func (cache *HealthStatusCache) delete(key string) {
	delete(cache.data, key)
}
