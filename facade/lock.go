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

package facade

import (
	"errors"
	"sync"
	"time"

	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain/service"
	"github.com/zenoss/glog"
)

// TenantLocker keeps track of locks per tenant
type TenantLocker struct {
	sync.Locker
	tenants map[string]*sync.RWMutex
}

// tlock is the global list of tenant locks
var tlock = &TenantLocker{Locker: &sync.Mutex{}, tenants: make(map[string]*sync.RWMutex)}

// getTenantLock returns the locker for a given tenant
func getTenantLock(tenantID string) (mutex *sync.RWMutex) {
	tlock.Lock()
	mutex, ok := tlock.tenants[tenantID]
	if !ok {
		tlock.tenants[tenantID] = &sync.RWMutex{}
		mutex = tlock.tenants[tenantID]
	}
	tlock.Unlock()
	return
}

// lockTenant sets the write lock for a given tenant and locks all services for
// that tenant
func (f *Facade) lockTenant(ctx datastore.Context, tenantID string) (err error) {
	mutex := getTenantLock(tenantID)
	mutex.Lock()
	defer func() {
		if err != nil {
			mutex.Unlock()
		}
	}()
	var svcs []service.Service
	if svcs, err = f.GetServices(ctx, dao.ServiceRequest{TenantID: tenantID}); err != nil {
		glog.Errorf("Could not get services for tenant %s: %s", tenantID, err)
		return
	}
	if err = f.zzk.LockServices(svcs); err != nil {
		glog.Errorf("Could not lock services for tenant %s: %s", tenantID, err)
		return
	}
	return
}

// unlockTenant unsets the write lock for a given tenant and unlocks all
// services for that tenant
func (f *Facade) unlockTenant(ctx datastore.Context, tenantID string) (err error) {
	mutex := getTenantLock(tenantID)
	var svcs []service.Service
	if svcs, err = f.GetServices(ctx, dao.ServiceRequest{TenantID: tenantID}); err != nil {
		glog.Errorf("Could not get services for tenant %s: %s", tenantID, err)
		return
	}
	if err = f.zzk.UnlockServices(svcs); err != nil {
		glog.Errorf("Could not unlock services for tenant %s: %s", tenantID, err)
		return
	}
	mutex.Unlock()
	return
}

// retryUnlockTenant is a persistent unlock for a given tenant
func (f *Facade) retryUnlockTenant(ctx datastore.Context, tenantID string, cancel <-chan time.Time, interval time.Duration) error {
	for {
		if err := f.unlockTenant(ctx, tenantID); err == nil {
			return nil
		}
		glog.Warningf("Could not unlock, retrying in %s", interval)
		select {
		case <-time.After(interval):
		case <-cancel:
			return errors.New("operation cancelled")
		}
	}
}
