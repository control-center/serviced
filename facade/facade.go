// Copyright 2014 The Serviced Authors.
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

package facade

import (
	"sync"

	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/dfs"
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/domain/pool"
	"github.com/control-center/serviced/domain/registry"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/servicetemplate"
)

var tenantLockLock = &sync.Mutex{}
var tenantLockMap = make(map[string]*tenantLock)

// assert interface
var _ FacadeInterface = &Facade{}

func New(dockerRegistryName string) *Facade {
	return &Facade{
		hostStore:     host.NewStore(),
		registryStore: registry.NewStore(),
		poolStore:     pool.NewStore(),
		serviceStore:  service.NewStore(),
		templateStore: servicetemplate.NewStore(),
	}
}

// Facade is an entrypoint to available controlplane methods
type Facade struct {
	hostStore     *host.HostStore
	registryStore *registry.ImageRegistryStore
	poolStore     *pool.Store
	templateStore *servicetemplate.Store
	serviceStore  *service.Store
	dfs           dfs.DFS
}

func (f *Facade) SetDFS(dfs dfs.DFS) {
	f.dfs = dfs
}

type tenantLock struct {
	ctx      datastore.Context
	facade   *Facade
	rwlocker *sync.RWMutex
	tenantID string
}

func getTenantLock(facade *Facade, tenantID string) *tenantLock {
	tenantLockLock.Lock()
	defer tenantLockLock.Unlock()
	tlock := tenantLockMap[tenantID]
	if tlock == nil {
		tlock = &tenantLock{
			ctx:      datastore.Get(),
			rwlocker: &sync.RWMutex{},
			facade:   facade,
			tenantID: tenantID,
		}
		tenantLockMap[tenantID] = tlock
	}
	return tlock
}

func (l *tenantLock) RLock() {
	l.rwlocker.RLock()
}

func (l *tenantLock) RUnlock() {
	l.rwlocker.RUnlock()
}

func (l *tenantLock) Lock() error {
	l.rwlocker.Lock()
	// TODO: call the locker on the facade
	return nil
}

func (l *tenantLock) Unlock() error {
	// TODO: call the unlocker on the facade--how to handle errors???
	l.rwlocker.Unlock()
	return nil
}
