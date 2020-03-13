// Copyright 2016 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package facade

import (
	"sync"

	"github.com/control-center/serviced/domain/pool"
)

// PoolCache is an interface accessing the pool cache data
type PoolCache interface {
	GetPools(getPoolsFunc GetPoolsFunc) ([]pool.ReadPool, error)
	SetDirty()
}

// poolCache is a simple in-memory cache for storing ReadPools.
// The dirty flag should be set when changes are made that affect a ReadPool.
type poolCache struct {
	mutex sync.RWMutex
	dirty bool
	pools []pool.ReadPool
}

// GetPoolsFunc should return an up-to-date slice of ReadPools.
type GetPoolsFunc func() ([]pool.ReadPool, error)

// NewPoolCache returns a new object that implements the PoolCache interface
func NewPoolCache() PoolCache {
	return &poolCache{
		mutex: sync.RWMutex{},
		dirty: true,
		pools: make([]pool.ReadPool, 0),
	}
}

// GetPools caches the result of getPoolsFunc if the cache is dirty
// then returns the cached ReadPools.
func (pc *poolCache) GetPools(getPoolsFunc GetPoolsFunc) ([]pool.ReadPool, error) {
	pc.mutex.Lock()
	defer pc.mutex.Unlock()

	var err error
	if pc.dirty {
		pc.pools, err = getPoolsFunc()
		if err == nil {
			pc.dirty = false
		}
	}
	return pc.pools, err
}

// SetDirty sets poolCache.dirty to true, meaning it will call a GetPoolsFunc
// next time GetPools is called.
func (pc *poolCache) SetDirty() {
	pc.mutex.Lock()
	defer pc.mutex.Unlock()
	pc.dirty = true
}
