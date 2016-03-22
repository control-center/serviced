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

package health_test

import (
	"testing"
	"time"

	. "github.com/control-center/serviced/health"
	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

var _ = Suite(&HealthStatusCacheTestSuite{})

type HealthStatusCacheTestSuite struct{}

func (s *HealthStatusCacheTestSuite) TestCRUD_NotExists(c *C) {
	// Get an item from the cache that does not exist
	cache := New()
	key := HealthStatusKey{
		ServiceID:       "test-service",
		InstanceID:      0,
		HealthCheckName: "test-health-0",
	}
	stat, ok := cache.Get(key)
	c.Assert(stat, DeepEquals, HealthStatus{})
	c.Assert(ok, Equals, false)
}

func (s *HealthStatusCacheTestSuite) TestCRUD_Exists(c *C) {
	// Get an item from the cache that does exist
	cache := New()
	key := HealthStatusKey{
		ServiceID:       "test-service",
		InstanceID:      0,
		HealthCheckName: "test-health-0",
	}
	value := HealthStatus{
		Status:    "ok",
		StartedAt: time.Now().Add(-5 * time.Minute),
		Duration:  30 * time.Second,
	}
	cache.Set(key, value, 1*time.Minute)
	stat, ok := cache.Get(key)
	c.Assert(stat, DeepEquals, value)
	c.Assert(ok, Equals, true)
}

func (s *HealthStatusCacheTestSuite) TestCRUD_Update(c *C) {
	// Update an item in the cache
	cache := New()
	key := HealthStatusKey{
		ServiceID:       "test-service",
		InstanceID:      0,
		HealthCheckName: "test-health-0",
	}
	value := HealthStatus{
		Status:    "ok",
		StartedAt: time.Now().Add(-5 * time.Minute),
		Duration:  30 * time.Second,
	}
	cache.Set(key, value, 1*time.Minute)
	stat, ok := cache.Get(key)
	c.Assert(stat, DeepEquals, value)
	c.Assert(ok, Equals, true)
	value = HealthStatus{
		Status:    "ok1",
		StartedAt: time.Now().Add(-5 * time.Minute),
		Duration:  30 * time.Second,
	}
	cache.Set(key, value, 1*time.Minute)
	stat, ok = cache.Get(key)
	c.Assert(stat, DeepEquals, value)
	c.Assert(ok, Equals, true)
}

func (s *HealthStatusCacheTestSuite) TestCRUD_Expired(c *C) {
	// Get an item from the cache that is expired
	cache := New()
	key := HealthStatusKey{
		ServiceID:       "test-service",
		InstanceID:      0,
		HealthCheckName: "test-health-0",
	}
	value := HealthStatus{
		Status:    "ok",
		StartedAt: time.Now().Add(-5 * time.Minute),
		Duration:  30 * time.Second,
	}
	cache.Set(key, value, 500*time.Millisecond)
	timer := time.After(500 * time.Millisecond)
	stat, ok := cache.Get(key)
	c.Assert(stat, DeepEquals, value)
	c.Assert(ok, Equals, true)
	<-timer
	stat, ok = cache.Get(key)
	c.Assert(stat, DeepEquals, HealthStatus{})
	c.Assert(ok, Equals, false)
}

func (s *HealthStatusCacheTestSuite) TestCRUD_Delete(c *C) {
	// Delete an item from the cache
	cache := New()
	key := HealthStatusKey{
		ServiceID:       "test-service",
		InstanceID:      0,
		HealthCheckName: "test-health-0",
	}
	value := HealthStatus{
		Status:    "ok",
		StartedAt: time.Now().Add(-5 * time.Minute),
		Duration:  30 * time.Second,
	}
	cache.Set(key, value, time.Minute)
	stat, ok := cache.Get(key)
	c.Assert(stat, DeepEquals, value)
	c.Assert(ok, Equals, true)
	cache.Delete(key)
	stat, ok = cache.Get(key)
	c.Assert(stat, DeepEquals, HealthStatus{})
	c.Assert(ok, Equals, false)
}

func (s *HealthStatusCacheTestSuite) TestCRUD_DeleteExpired(c *C) {
	// Delete expired items from the cache
	cache := New()
	key := HealthStatusKey{
		ServiceID:       "test-service",
		InstanceID:      0,
		HealthCheckName: "test-health-0",
	}
	value := HealthStatus{
		Status:    "ok1",
		StartedAt: time.Now().Add(-5 * time.Minute),
		Duration:  30 * time.Second,
	}
	cache.Set(key, value, 500*time.Millisecond)
	stat, ok := cache.Get(key)
	c.Assert(stat, DeepEquals, value)
	c.Assert(ok, Equals, true)
	key = HealthStatusKey{
		ServiceID:       "test-service",
		InstanceID:      0,
		HealthCheckName: "test-health-1",
	}
	value = HealthStatus{
		Status:    "ok2",
		StartedAt: time.Now().Add(-5 * time.Minute),
		Duration:  30 * time.Second,
	}
	cache.Set(key, value, 500*time.Millisecond)
	stat, ok = cache.Get(key)
	c.Assert(stat, DeepEquals, value)
	c.Assert(ok, Equals, true)
	key = HealthStatusKey{
		ServiceID:       "test-service",
		InstanceID:      0,
		HealthCheckName: "test-health-2",
	}
	value = HealthStatus{
		Status:    "ok3",
		StartedAt: time.Now().Add(-5 * time.Minute),
		Duration:  30 * time.Second,
	}
	cache.Set(key, value, time.Minute)
	stat, ok = cache.Get(key)
	c.Assert(stat, DeepEquals, value)
	c.Assert(ok, Equals, true)
	c.Assert(cache.Size(), Equals, 3)
	<-time.After(500 * time.Millisecond)
	cache.DeleteExpired()
	stat, ok = cache.Get(key)
	c.Assert(stat, DeepEquals, value)
	c.Assert(ok, Equals, true)
	c.Assert(cache.Size(), Equals, 1)
}

func (s *HealthStatusCacheTestSuite) TestCRUD_DeleteInstance(c *C) {
	// Delete healthchecks for a single instance
	cache := New()
	key := HealthStatusKey{
		ServiceID:       "test-service",
		InstanceID:      0,
		HealthCheckName: "test-health-0",
	}
	value := HealthStatus{
		Status:    "ok1",
		StartedAt: time.Now().Add(-5 * time.Minute),
		Duration:  30 * time.Second,
	}
	cache.Set(key, value, time.Minute)
	stat, ok := cache.Get(key)
	c.Assert(stat, DeepEquals, value)
	c.Assert(ok, Equals, true)
	key = HealthStatusKey{
		ServiceID:       "test-service",
		InstanceID:      0,
		HealthCheckName: "test-health-1",
	}
	value = HealthStatus{
		Status:    "ok2",
		StartedAt: time.Now().Add(-5 * time.Minute),
		Duration:  30 * time.Second,
	}
	cache.Set(key, value, time.Minute)
	stat, ok = cache.Get(key)
	c.Assert(stat, DeepEquals, value)
	c.Assert(ok, Equals, true)
	key = HealthStatusKey{
		ServiceID:       "test-service",
		InstanceID:      1,
		HealthCheckName: "test-health-2",
	}
	value = HealthStatus{
		Status:    "ok3",
		StartedAt: time.Now().Add(-5 * time.Minute),
		Duration:  30 * time.Second,
	}
	cache.Set(key, value, time.Minute)
	stat, ok = cache.Get(key)
	c.Assert(stat, DeepEquals, value)
	c.Assert(ok, Equals, true)
	c.Assert(cache.Size(), Equals, 3)
	cache.DeleteInstance("test-service", 0)
	stat, ok = cache.Get(key)
	c.Assert(stat, DeepEquals, value)
	c.Assert(ok, Equals, true)
	c.Assert(cache.Size(), Equals, 1)
}

func (s *HealthStatusCacheTestSuite) TestSetPurgeFrequency(c *C) {
	// Verifying purge frequency is working
	cache := New()
	key := HealthStatusKey{
		ServiceID:       "test-service",
		InstanceID:      0,
		HealthCheckName: "test-health-0",
	}
	value := HealthStatus{
		Status:    "ok1",
		StartedAt: time.Now().Add(-5 * time.Minute),
		Duration:  30 * time.Second,
	}
	cache.Set(key, value, 0)
	c.Assert(cache.Size(), Equals, 1)
	cache.SetPurgeFrequency(500 * time.Millisecond)
	<-time.After(time.Second)
	c.Assert(cache.Size(), Equals, 0)
	cache.SetPurgeFrequency(0)
	cache.Set(key, value, 500*time.Millisecond)
	<-time.After(time.Second)
	c.Assert(cache.Size(), Equals, 1)
}
