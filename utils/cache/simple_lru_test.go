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

// Package agent implements a service that runs on a serviced node. It is
// responsible for ensuring that a particular node is running the correct services
// and reporting the state and health of those services back to the master
// serviced.

// +build unit

package cache

import (
	"fmt"
	"testing"
	"time"

	. "gopkg.in/check.v1"
)

var (
	maxCacheSize = 4
	itemTimeToLive = time.Minute * 2
	cleanupInterval = time.Second * 30
)

func TestSimpleLRU(t *testing.T) { TestingT(t) }

type  TestSimpleLRUSuite struct{
	cache *SimpleLRUCache
	shutdown chan struct{}
}

var _ = Suite(&TestSimpleLRUSuite{})


type testItem struct {
	Name  string
	ID    int
}

func (s * TestSimpleLRUSuite) SetUpTest(c *C) {
	s.shutdown = make(chan struct{})
	s.cache, _ = NewSimpleLRUCache(maxCacheSize, itemTimeToLive, cleanupInterval, s.shutdown)
}

func (s * TestSimpleLRUSuite) TearDownTest(c *C) {
	close(s.shutdown)
}

func (s * TestSimpleLRUSuite) TestConstructor(c *C) {
	shutdown := make(chan struct{})
	defer close(shutdown)
	cache, err := NewSimpleLRUCache(maxCacheSize, itemTimeToLive, cleanupInterval, shutdown)

	c.Assert(err, IsNil)
	c.Assert(cache, NotNil)
	c.Assert(cache.GetMaxSize(), Equals, maxCacheSize)
	c.Assert(cache.GetExpiration(), Equals, itemTimeToLive)
	c.Assert(cache.GetCleanupInterval(), Equals, cleanupInterval)
	c.Assert(cache.GetCurrentSize(), Equals, 0)
}


func (s * TestSimpleLRUSuite) TestConstructorFails(c *C) {
	shutdown := make(chan struct{})
	defer close(shutdown)
	cache, err := NewSimpleLRUCache(0, itemTimeToLive, cleanupInterval, shutdown)
	c.Assert(err, NotNil)
	c.Assert(cache, IsNil)

	cache, err = NewSimpleLRUCache(-1, itemTimeToLive, cleanupInterval, shutdown)
	c.Assert(err, NotNil)
	c.Assert(cache, IsNil)
}

func (s * TestSimpleLRUSuite) TestGetOnEmptyCache(c *C) {
	result, isFound := s.cache.Get("somekey")

	c.Assert(isFound, Equals, false)
	c.Assert(result, IsNil)
}

func (s * TestSimpleLRUSuite) TestSimpleSetAndGet(c *C) {
	item := testItem{
		Name: "something",
		ID:   21,
	}
	s.cache.Set(item.Name, item)

	c.Assert(s.cache.GetCurrentSize(), Equals, 1)

	result, isFound := s.cache.Get(item.Name)
	c.Assert(isFound, Equals, true)
	c.Assert(result, Equals, item)

	result, isFound = s.cache.Get("some unknown key")
	c.Assert(isFound, Equals, false)
	c.Assert(result, IsNil)
}

func (s * TestSimpleLRUSuite) TestMaxItems(c *C) {
	// Fill the cache with exactly the max number of items
	for i := 1; i <= maxCacheSize; i++ {
		item := testItem{
			Name: fmt.Sprintf("key %d", i),
			ID:   i,
		}
		s.cache.Set(item.Name, item)
	}

	// Verify we have all of the items we expect
	c.Assert(s.cache.GetCurrentSize(), Equals, maxCacheSize)
	for i := 1; i <= maxCacheSize; i++ {
		key := fmt.Sprintf("key %d", i)
		result, isFound := s.cache.Get(key)
		c.Assert(isFound, Equals, true)
		c.Assert(result.(testItem).ID, Equals, i)
	}

	// Add one more item, which should push the oldest item out of the cache (item #1)
	newItem := testItem{
		Name: "something",
		ID:   99,
	}
	s.cache.Set(newItem.Name, newItem)

	// Verify it's there
	c.Assert(s.cache.GetCurrentSize(), Equals, maxCacheSize)
	result, isFound := s.cache.Get(newItem.Name)
	c.Assert(isFound, Equals, true)
	c.Assert(result, Equals, newItem)

	// Verify the oldest item is NOT there
	key := fmt.Sprintf("key %d", 1)
	result, isFound = s.cache.Get(key)
	c.Assert(isFound, Equals, false)
	c.Assert(result, IsNil)

	// Verify we have all of other initial items we expect
	c.Assert(s.cache.GetCurrentSize(), Equals, maxCacheSize)
	for i := 2; i <= maxCacheSize; i++ {
		key := fmt.Sprintf("key %d", i)
		result, isFound := s.cache.Get(key)
		c.Assert(isFound, Equals, true)
		c.Assert(result.(testItem).ID, Equals, i)
	}

	// Get the next oldest item (#2), which should make item #3 the oldest
	key = fmt.Sprintf("key %d", 2)
	result, isFound = s.cache.Get(key)
	c.Assert(isFound, Equals, true)
	c.Assert(result, NotNil)
	c.Assert(result.(testItem).ID, Equals, 2)

	// Do a couple of gets on newItem so it's more recently used
	s.cache.Get(newItem.Name)
	s.cache.Get(newItem.Name)

	// Add one more item, which should push the oldest item out of the cache (item #3)
	item := testItem{
		Name: "something new",
		ID:   100,
	}
	s.cache.Set(item.Name, item)
	c.Assert(s.cache.GetCurrentSize(), Equals, maxCacheSize)
	result, isFound = s.cache.Get(item.Name)
	c.Assert(isFound, Equals, true)

	// Verify #3 has been remove from the cache
	key = fmt.Sprintf("key %d", 3)
	result, isFound = s.cache.Get(key)
	c.Assert(isFound, Equals, false)
	c.Assert(result, IsNil)
}

func (s * TestSimpleLRUSuite) TestCacheEmptyAfterTTLExpires(c *C) {
	// Use a shorter values than the test suite defaults so we don't hold up the entire test suite waiting
	// for the cache to clear
	shutdown := make(chan struct{})
	defer close(shutdown)
	cache, _ := NewSimpleLRUCache(maxCacheSize, time.Second*10, time.Second*2, shutdown)

	// Fill the cache with exactly the max number of items
	for i := 1; i <= maxCacheSize; i++ {
		item := testItem{
			Name: fmt.Sprintf("key %d", i),
			ID:   i,
		}
		cache.Set(item.Name, item)
	}

	// Sleep long enough for the cache cleanup to run
	time.Sleep(cache.GetExpiration() + cache.GetCleanupInterval())

	c.Assert(cache.GetCurrentSize(), Equals, 0)
}

func (s * TestSimpleLRUSuite) TestUsedItemsRemainInCache(c *C) {
	// Use a shorter values than the test suite defaults so we don't hold up the entire test suite waiting
	// for the cache to clear
	shutdown := make(chan struct{})
	defer close(shutdown)
	cache, _ := NewSimpleLRUCache(maxCacheSize, time.Second*10, time.Second*2, shutdown)

	// Fill the cache with exactly the max number of items
	for i := 1; i <= maxCacheSize; i++ {
		item := testItem{
			Name: fmt.Sprintf("key %d", i),
			ID:   i,
		}
		cache.Set(item.Name, item)
	}

	time.Sleep(time.Second*5)
	for i := 1; i <= 2; i++ {
		_, ok := cache.Get(fmt.Sprintf("key %d", i))
		c.Assert(ok, Equals, true)
	}

	// Sleep long enough for the cache cleanup to run
	time.Sleep(time.Second*5 + cache.GetCleanupInterval())

	c.Assert(cache.GetCurrentSize(), Equals, 2)
}
