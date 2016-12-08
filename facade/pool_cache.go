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

type poolCache struct {
	mutex sync.RWMutex
	dirty bool
	pools []pool.ReadPool
}

type GetPoolsFunc func() ([]pool.ReadPool, error)

func NewPoolCache() *poolCache {
	return &poolCache{
		mutex: sync.RWMutex{},
		dirty: true,
		pools: make([]pool.ReadPool, 0),
	}
}

func (pc *poolCache) GetPools(getPoolsFunc GetPoolsFunc) ([]pool.ReadPool, error) {
	var err error
	if pc.dirty {
		pc.mutex.Lock()
		defer pc.mutex.Unlock()
		pc.pools, err = getPoolsFunc()
		if err == nil {
			pc.dirty = false
		}
	}
	return pc.pools, err
}

func (pc *poolCache) SetDirty() {
	pc.mutex.Lock()
	defer pc.mutex.Unlock()
	pc.dirty = true
}
