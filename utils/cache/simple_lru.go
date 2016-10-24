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
package cache

import (
	lru "github.com/hashicorp/golang-lru"
	"time"
)

type SimpleLRUCache struct {
	maxItems        int
	expiration      time.Duration
	cleanupInterval time.Duration

	// Keep the cache implementation private
	lruCache *lru.Cache
}

// assert the LRUCache interface is implemented by SimpleLRUCache
var _ LRUCache = &SimpleLRUCache{}

var timeFunc func() time.Time = time.Now

// At sets a fake time, executes the provided
// function, then restores the default time getter,
// making it possible to test time-sensitive stuff
func At(t time.Time, f func()) {
	defer func() {
		timeFunc = time.Now
	}()

	timeFunc = func() time.Time {
		return t
	}

	f()
}

// An internal struct to track the expiration time for each item in the cache
type cacheItem struct {
	key     string
	value   interface{}
	expires time.Time
}

//
// Creates a new instance of SimpleLRUCache that supports both a maximum cache size and a maximum time-to-live for each
// item in the cache.
// If on Set(), the number of active items exceeds maxItems, the oldest item will be removed to make room for the
// the new item.  Additionally, the cache tracks the age of item and automatically removes items that have aged past
// their expiration time.
//
// Parameters:
// maxItems        - the maximum number of items in the cache
// expiration      - the expiration time for each item added to the cache. Items older than this value will automatically
//                   be removed.
// cleanupInterval - the interval at which the cleaner runs to remove expired items from the cache.
//                   If 0, then the cleaner is not started.  The minimum non-zero value is one second.
// shutdown        - a channel that can be used to shutdown the cleanup timer
//
func NewSimpleLRUCache(maxItems int, expiration time.Duration, cleanupInterval time.Duration, shutdown chan struct{}) (*SimpleLRUCache, error) {

	//
	if cleanupInterval > 0 && cleanupInterval < time.Second {
		cleanupInterval = time.Second
	}

	simpleLRUCache := SimpleLRUCache{
		maxItems:        maxItems,
		expiration:      expiration,
		cleanupInterval: cleanupInterval,
	}

	var err error
	simpleLRUCache.lruCache, err = lru.New(maxItems)
	if err != nil {
		return nil, err
	}

	if cleanupInterval > 0 {
		simpleLRUCache.startCleaner(shutdown)
	}
	return &simpleLRUCache, nil
}

// Gets the value for key; returns the value and true if found, nil and false otherwise.
func (c *SimpleLRUCache) Get(key string) (interface{}, bool) {
	data, ok := c.lruCache.Get(key)
	if data != nil && ok {
		var item cacheItem
		item, _ = data.(cacheItem)
		item.expires = timeFunc().Add(c.expiration) // update the expiration
		c.lruCache.Add(key, item)
		return item.value, true
	}
	return nil, false
}

// Add a value to the cache for the specified key.
// If the key already exists in the cache, it's value will be replaced.
func (c *SimpleLRUCache) Set(key string, value interface{}) {
	item := cacheItem{
		key:     key,
		value:   value,
		expires: timeFunc().Add(c.expiration),
	}

	c.lruCache.Add(key, item)
}

// Returns the maximum size of the cache.
func (c *SimpleLRUCache) GetMaxSize() int {
	return c.maxItems
}

// Returns the current number of items in the cache.
func (c *SimpleLRUCache) GetCurrentSize() int {
	return c.lruCache.Len()
}

func (c *SimpleLRUCache) GetExpiration() time.Duration {
	return c.expiration
}

func (c *SimpleLRUCache) GetCleanupInterval() time.Duration {
	return c.cleanupInterval
}

func (c *SimpleLRUCache) startCleaner(shutdown chan struct{}) {
	timer := time.NewTicker(c.cleanupInterval)
	go (func() {
		for {
			select {
			case <-timer.C:
				c.cleanup()
			case <-shutdown:
				timer.Stop()
				return
			}
		}
	})()
}

func (c *SimpleLRUCache) cleanup() {
	now := timeFunc()
	for _, key := range c.lruCache.Keys() {
		data, ok := c.lruCache.Get(key)
		if data != nil && ok {
			var item cacheItem
			item, _ = data.(cacheItem)
			if item.expires.Before(now) {
				c.lruCache.Remove(key)
			}
		}
	}
}
