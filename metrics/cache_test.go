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

// +build unit

package metrics

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/control-center/serviced/utils"
)

func TestCache(t *testing.T) {

	clock := utils.NewTestClock()

	cache := MemoryUsageCache{
		Usages: make(map[string]*MemoryUsageItem),
		TTL:    time.Minute,
		Clock:  clock,
	}

	memusage1 := []MemoryUsageStats{
		MemoryUsageStats{ServiceID: "memusage1"},
	}
	memusage2 := []MemoryUsageStats{
		MemoryUsageStats{ServiceID: "memusage2"},
	}

	getter1 := func() ([]MemoryUsageStats, error) {
		return memusage1, nil
	}

	getter2 := func() ([]MemoryUsageStats, error) {
		return memusage2, nil
	}

	errgetter := func() ([]MemoryUsageStats, error) {
		return nil, errors.New("")
	}

	// Cache is empty, test error propagation in getter
	x, err := cache.Get("first", errgetter)
	if err == nil {
		t.Errorf("Cache did not propagate error in getter properly")
	}

	// Cache still empty, test getter
	x, err = cache.Get("first", getter1)
	if !reflect.DeepEqual(x, memusage1) {
		t.Errorf("Empty cache did not return a new item when a key did not yet exist")
	}

	// Cache has value for key, try error getter again
	x, err = cache.Get("first", errgetter)
	if err != nil {
		t.Errorf("Cache returned an error when it should have returned a cached value")
	}

	// Cache has value for key, try a new key with a different getter
	x, err = cache.Get("second", getter2)
	if !reflect.DeepEqual(x, memusage2) {
		t.Errorf("Non-empty cache did not return a new item when a key did not yet exist")
	}

	// Cache has a value for key, try different getter
	x, err = cache.Get("first", getter2)
	if !reflect.DeepEqual(x, memusage1) {
		t.Errorf("Cache did not return the correct item")
	}

	// Force expiration
	// I know this seems crazy, but go clock.Fire() may not trigger before
	// cache.Get is called.
	done := make(chan struct{})
	go func() {
		close(done)
		clock.Fire()
	}()
	<-done

	// Cache should no longer have a value for key, try with different getter
	x, err = cache.Get("first", getter2)
	if !reflect.DeepEqual(x, memusage2) {
		t.Errorf("Cache returned a value that should have expired")
	}
}
