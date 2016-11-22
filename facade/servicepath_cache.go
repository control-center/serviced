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
	"path"
	"strings"
	"sync"

	log "github.com/Sirupsen/logrus"
	"github.com/control-center/serviced/domain/service"
)

type serviceCache struct {
	mutex sync.RWMutex
	paths map[string]servicePath
}

type servicePath struct {
	serviceID   string
	tenantID    string
	parentID    string
	servicePath string
}

func NewServiceCache() *serviceCache {
	return &serviceCache{
		mutex: sync.RWMutex{},
		paths: make(map[string]servicePath),
	}
}

type GetServiceDetails func(servicedID string) (*service.ServiceDetails, error)

// GetTenantID returns the tenant ID for the specified service from its cache. If the specified service
// is not in the cache, it uses getServiceFunc to populate the cache (assuming serviceID really exists in the DB).
func (sc *serviceCache) GetTenantID(serviceID string, getServiceFunc GetServiceDetails) (string, error) {
	if cachedSvc, found := sc.lookUpService(serviceID); found {
		return cachedSvc.tenantID, nil
	}

	cachedSvc, err := sc.updateCache(serviceID, getServiceFunc)
	if err != nil {
		return "", err
	}
	return cachedSvc.tenantID, nil
}

// GetServicePath returns the tenant ID and service path for the specified service from the.
func (sc *serviceCache) GetServicePath(serviceID string, getServiceFunc GetServiceDetails) (string, string, error) {
	if cachedSvc, found := sc.lookUpService(serviceID); found {
		return cachedSvc.tenantID, cachedSvc.servicePath, nil
	}

	cachedSvc, err := sc.updateCache(serviceID, getServiceFunc)
	if err != nil {
		return "", "", err
	}
	return cachedSvc.tenantID, cachedSvc.servicePath, nil
}

// RemoveIfParentChanged will remove all entries from the cache for this service and its children if the
// specified service's parentID is different from the cached value.
// Returns true if one or more entries was removed from the cache; false otherwise.
func (sc *serviceCache) RemoveIfParentChanged(serviceID string, parentID string) bool {
	cachedSvc, ok := sc.lookUpService(serviceID)
	if !ok || cachedSvc.parentID == parentID {
		return false
	}

	sc.mutex.Lock()
	defer sc.mutex.Unlock()
	for key, value := range sc.paths {
		if strings.HasPrefix(value.servicePath, cachedSvc.servicePath) {
			delete(sc.paths, key)
		}
	}
	return true
}

// Reset clears the cache.
func (sc *serviceCache) Reset() {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()
	sc.paths = make(map[string]servicePath)
}

func (sc *serviceCache) lookUpService(svcID string) (servicePath, bool) {
	sc.mutex.RLock()
	defer sc.mutex.RUnlock()
	cachedSvc, found := sc.paths[svcID]
	return cachedSvc, found
}

func (sc *serviceCache) updateCache(serviceID string, getServiceFunc GetServiceDetails) (servicePath, error) {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()

	svcPaths := make([]servicePath, 0)
	cachedSvc, err := sc.buildServicePath(serviceID, &svcPaths, getServiceFunc)
	if err != nil {
		return servicePath{}, err
	}

	for _, path := range svcPaths {
		sc.paths[path.serviceID] = path
	}
	return cachedSvc, nil
}

func (sc *serviceCache) buildServicePath(serviceID string, svcPaths *[]servicePath, getServiceFunc GetServiceDetails) (svcPath servicePath, err error) {
	logger := plog.WithFields(log.Fields{
		"serviceid": serviceID,
	})

	// FIXME: This getServiceFunc method should be replaced with something much lighter, such as GetServiceDetails.
	//        In fact, there's probably no need at all for the getServiceFunc to be passed into this method.
	svc, err := getServiceFunc(serviceID)
	if err != nil {
		logger.WithError(err).Error("Could not find service")
		return servicePath{}, err
	}
	if svc.ParentServiceID == "" {
		svcPath = servicePath{
			serviceID:   serviceID,
			tenantID:    serviceID,
			servicePath: "/" + serviceID,
		}
		*svcPaths = append(*svcPaths, svcPath)
		return svcPath, nil
	}

	var parent servicePath
	var parentCached bool
	if parent, parentCached = sc.paths[svc.ParentServiceID]; !parentCached {
		parent, err = sc.buildServicePath(svc.ParentServiceID, svcPaths, getServiceFunc)
		if err != nil {
			return servicePath{}, err
		}
	}

	svcPath = servicePath{
		serviceID:   svc.ID,
		tenantID:    parent.tenantID,
		parentID:    svc.ParentServiceID,
		servicePath: path.Join(parent.servicePath, svc.ID),
	}
	*svcPaths = append(*svcPaths, svcPath)
	return svcPath, nil
}
