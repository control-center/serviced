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

package service

import (
	"errors"
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/datastore/elastic"
	"github.com/control-center/serviced/domain/servicedefinition"
	"github.com/control-center/serviced/validation"
	"strings"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/control-center/serviced/logging"
)

var (
	kind = "service"
	plog = logging.PackageLogger()
)

type volatileService struct {
	ID           string
	DesiredState int
	CurrentState string
	UpdatedAt    time.Time // Time when the cached entry was changed, not when elastic was changed
}

// Store type for interacting with Service persistent storage
type Store interface {
	// Put adds or updates a Service
	Put(ctx datastore.Context, svc *Service) error

	// Get a Service by id. Return ErrNoSuchEntity if not found
	Get(ctx datastore.Context, id string) (*Service, error)

	// Delete removes the a Service if it exists
	Delete(ctx datastore.Context, id string) error

	// Update the DesiredState in volatile memory for the service
	UpdateDesiredState(ctx datastore.Context, serviceID string, desiredState int) error

	// Update the CurrentState in volatile memory for the service
	UpdateCurrentState(ctx datastore.Context, serviceID string, currentState string) error

	// GetServices returns all services
	GetServices(ctx datastore.Context) ([]Service, error)

	// GetUpdatedServices returns all services updated since "since" time.Duration ago
	GetUpdatedServices(ctx datastore.Context, since time.Duration) ([]Service, error)

	// GetTaggedServices returns services with the given tags
	GetTaggedServices(ctx datastore.Context, tags ...string) ([]Service, error)

	// GetServicesByPool returns services with the given pool id
	GetServicesByPool(ctx datastore.Context, poolID string) ([]Service, error)

	// GetServiceCountByImage returns a count of services using a given imageid
	GetServiceCountByImage(ctx datastore.Context, imageID string) (int, error)

	// GetServicesByDeployment returns services with the given deployment id
	GetServicesByDeployment(ctx datastore.Context, deploymentID string) ([]Service, error)

	// GetChildServices returns services that are children of the given parent service id
	GetChildServices(ctx datastore.Context, parentID string) ([]Service, error)

	FindChildService(ctx datastore.Context, deploymentID, parentID, serviceName string) (*Service, error)

	// FindTenantByDeployment returns the tenant service for a given deployment id and service name
	FindTenantByDeploymentID(ctx datastore.Context, deploymentID, name string) (*Service, error)

	// GetServiceDetails returns the details for the given service
	GetServiceDetails(ctx datastore.Context, serviceID string) (*ServiceDetails, error)

	// GetChildServiceDetails returns the details for the child service of the given parent
	GetServiceDetailsByParentID(ctx datastore.Context, parentID string, since time.Duration) ([]ServiceDetails, error)

	// GetAllServiceHealth returns all service health
	GetAllServiceHealth(ctx datastore.Context) ([]ServiceHealth, error)

	// GetServiceHealth returns a service health by service id
	GetServiceHealth(ctx datastore.Context, serviceID string) (*ServiceHealth, error)

	// GetAllPublicEndpoints returns all public endpoints in the system
	GetAllPublicEndpoints(ctx datastore.Context) ([]PublicEndpoint, error)

	// GetAllExportedEndpoints returns all the exported endpoints in the system
	GetAllExportedEndpoints(ctx datastore.Context) ([]ExportedEndpoint, error)

	// GetAllIPAssignments returns all IP assignments in the system, including those that may not have address assignments
	GetAllIPAssignments(ctx datastore.Context) ([]BaseIPAssignment, error)

	// GetServiceDetailsByIDOrName returns the service details for any services
	// whose serviceID matches the query exactly or whose names contain the
	// query as a substring or string suffix
	GetServiceDetailsByIDOrName(ctx datastore.Context, query string, noprefix bool) ([]ServiceDetails, error)

	Query(ctx datastore.Context, query Query) ([]ServiceDetails, error)
}

// NewStore creates a Service store
func NewStore() Store {
	return &storeImpl{
		serviceCacheLock: &sync.RWMutex{},
		serviceCache:     map[string]volatileService{},
	}
}

type storeImpl struct {
	ds               datastore.DataStore
	serviceCacheLock *sync.RWMutex
	serviceCache     map[string]volatileService
}

// Put adds or updates a Service
func (s *storeImpl) Put(ctx datastore.Context, svc *Service) error {
	//No need to store ConfigFiles
	defer ctx.Metrics().Stop(ctx.Metrics().Start("ServiceStore.Put"))
	svc.ConfigFiles = make(map[string]servicedefinition.ConfigFile)

	err := s.ds.Put(ctx, Key(svc.ID), svc)
	if err == nil {
		s.updateVolatileInfo(svc.ID, svc.DesiredState, svc.UpdatedAt) // Uses Mutex Lock
	}
	return err
}

// UpdateDesiredState updates the DesiredState for the service by saving the information in volatile storage.
func (s *storeImpl) UpdateDesiredState(ctx datastore.Context, serviceID string, desiredState int) error {
	plog.WithFields(log.Fields{
		"serviceID":    serviceID,
		"desiredState": desiredState,
	}).Debug("Storing desiredState")
	s.updateDesiredState(serviceID, desiredState, time.Now())
	return nil
}

// UpdateCurrentState updates the CurrentState for the service by saving the information in volatile storage.
func (s *storeImpl) UpdateCurrentState(ctx datastore.Context, serviceID string, currentState string) error {
	plog.WithFields(log.Fields{
		"serviceID":    serviceID,
		"currentState": currentState,
	}).Debug("Storing currentState")
	s.updateCurrentState(serviceID, currentState, time.Now())
	return nil
}

// Get a Service by id. Return ErrNoSuchEntity if not found
func (s *storeImpl) Get(ctx datastore.Context, id string) (*Service, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("ServiceStore.Get"))
	// Get the service from elastic and fill additional info into the
	// service object.
	svc, err := s.get(ctx, id)
	if err == nil {
		s.fillAdditionalInfo(svc) // Mutex RLock
	}
	return svc, err
}

// Get the service from elastic (without modifications)
func (s *storeImpl) get(ctx datastore.Context, id string) (*Service, error) {
	svc := &Service{}
	if err := s.ds.Get(ctx, Key(id), svc); err != nil {
		return nil, err
	}
	return svc, nil
}

// Delete removes the a Service if it exists
func (s *storeImpl) Delete(ctx datastore.Context, id string) error {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("ServiceStore.Delete"))
	err := s.ds.Delete(ctx, Key(id))
	if err == nil {
		s.removeVolatileInfo(id) // Uses Mutex RLock
	}
	return err
}

// GetServices returns all services
func (s *storeImpl) GetServices(ctx datastore.Context) ([]Service, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("ServiceStore.GetServices"))
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []map[string]interface{}{
					{"exists": map[string]string{"field": "ID"}},
					{"term": map[string]string{"type": kind}},
				},
			},
		},
	}
	return s.query(ctx, query)
}

// GetUpdatedServices returns all services updated since "since" time.Duration ago
func (s *storeImpl) GetUpdatedServices(ctx datastore.Context, since time.Duration) ([]Service, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("ServiceStore.GetUpdatedServices"))
	q := datastore.NewQuery(ctx)
	t0 := time.Now().Add(-since)
	t0s := t0.Format(time.RFC3339)
	//elasticQuery := search.Query().Range(search.Range().Field("UpdatedAt").From(t0s)).Search("_exists_:ID")
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []map[string]interface{}{
					{"range": map[string]interface{}{"UpdatedAt": map[string]interface{}{"gte": t0s}}},
					{"exists": map[string]string{"field": "ID"}},
					{"term": map[string]string{"type": kind}},
				},
			},
		},
	}

	search, err := elastic.BuildSearchRequest(query, "controlplane")
	if err != nil {
		return nil, err
	}
	results, err := q.Execute(search)
	if err != nil {
		return nil, err
	}
	// First get the list of updated services from Elastic.
	svcs, err := s.convert(results)
	if err != nil {
		return nil, err
	}
	// Then add updated services from the cache
	return s.addUpdatedServicesFromCache(ctx, svcs, t0)
}

// GetTaggedServices returns services with the given tags
func (s *storeImpl) GetTaggedServices(ctx datastore.Context, tags ...string) ([]Service, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("ServiceStore.GetTaggedServices"))
	if len(tags) == 0 {
		return nil, errors.New("empty tags not allowed")
	}
	qs := strings.Join(tags, " AND ")
	return s.query(ctx, qs)
}

// GetServicesByPool returns services with the given pool id
func (s *storeImpl) GetServicesByPool(ctx datastore.Context, poolID string) ([]Service, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("ServiceStore.GetServicesByPool"))
	id := strings.TrimSpace(poolID)
	if id == "" {
		return nil, errors.New("empty poolID not allowed")
	}
	q := datastore.NewQuery(ctx)

	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []map[string]interface{}{
					{"term": map[string]string{"PoolID": id}},
					{"term": map[string]string{"type": kind}},
				},
			},
		},
	}

	search, err := elastic.BuildSearchRequest(query, "controlplane")
	if err != nil {
		return nil, err
	}

	results, err := q.Execute(search)
	if err != nil {
		return nil, err
	}
	return s.convert(results)
}

// GetServiceCountByImage returns a count of services using a given imageid
func (s *storeImpl) GetServiceCountByImage(ctx datastore.Context, imageID string) (int, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("ServiceStore.GetServiceCountByImage"))
	id := strings.TrimSpace(imageID)
	if id == "" {
		return 0, errors.New("empty imageID not allowed")
	}
	q := datastore.NewQuery(ctx)

	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []map[string]interface{}{
					{"term": map[string]string{"ImageID": id}},
					{"term": map[string]string{"type": kind}},
				},
			},
		},
	}

	search, err := elastic.BuildSearchRequest(query, "controlplane")
	if err != nil {
		return 0, err
	}

	results, err := q.Execute(search)
	if err != nil {
		return 0, err
	}
	return results.Len(), nil
}

// GetServicesByDeployment returns services with the given deployment id
func (s *storeImpl) GetServicesByDeployment(ctx datastore.Context, deploymentID string) ([]Service, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("ServiceStore.GetServicesByDeployment"))
	id := strings.TrimSpace(deploymentID)
	if id == "" {
		return nil, errors.New("empty deploymentID not allowed")
	}
	q := datastore.NewQuery(ctx)

	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []map[string]interface{}{
					{"term": map[string]string{"DeploymentID": id}},
					{"term": map[string]string{"type": kind}},
				},
			},
		},
	}

	search, err := elastic.BuildSearchRequest(query, "controlplane")
	if err != nil {
		return nil, err
	}

	results, err := q.Execute(search)
	if err != nil {
		return nil, err
	}
	return s.convert(results)
}

// GetChildServices returns services that are children of the given parent service id
func (s *storeImpl) GetChildServices(ctx datastore.Context, parentID string) ([]Service, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("ServiceStore.GetChildServices"))
	id := strings.TrimSpace(parentID)
	if id == "" {
		return nil, errors.New("empty parent service id not allowed")
	}
	q := datastore.NewQuery(ctx)

	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []map[string]interface{}{
					{"term": map[string]string{"ParentServiceID": id}},
					{"term": map[string]string{"type": kind}},
				},
			},
		},
	}

	search, err := elastic.BuildSearchRequest(query, "controlplane")
	if err != nil {
		return nil, err
	}
	results, err := q.Execute(search)
	if err != nil {
		return nil, err
	}
	return s.convert(results)
}

func (s *storeImpl) FindChildService(ctx datastore.Context, deploymentID, parentID, serviceName string) (*Service, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("ServiceStore.FindChildService"))
	parentID = strings.TrimSpace(parentID)

	if deploymentID = strings.TrimSpace(deploymentID); deploymentID == "" {
		return nil, errors.New("empty deployment ID not allowed")
	} else if serviceName = strings.TrimSpace(serviceName); serviceName == "" {
		return nil, errors.New("empty service name not allowed")
	}
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []map[string]interface{}{
					{"term": map[string]string{"DeploymentID": deploymentID}},
					{"term": map[string]string{"ParentServiceID": parentID}},
					{"term": map[string]string{"Name": serviceName}},
					{"term": map[string]string{"type": kind}},
				},
			},
		},
	}

	search, err := elastic.BuildSearchRequest(query, "controlplane")
	if err != nil {
		return nil, err
	}
	q := datastore.NewQuery(ctx)
	results, err := q.Execute(search)
	if err != nil {
		return nil, err
	}

	if results.Len() == 0 {
		return nil, nil
	} else if svcs, err := s.convert(results); err != nil {
		return nil, err
	} else {
		return &svcs[0], nil
	}
}

// FindTenantByDeployment returns the tenant service for a given deployment id and service name
func (s *storeImpl) FindTenantByDeploymentID(ctx datastore.Context, deploymentID, name string) (*Service, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("ServiceStore.FindTenantByDeploymentID"))
	if deploymentID = strings.TrimSpace(deploymentID); deploymentID == "" {
		return nil, errors.New("empty deployment ID not allowed")
	} else if name = strings.TrimSpace(name); name == "" {
		return nil, errors.New("empty service name not allowed")
	}

	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []map[string]interface{}{
					{"term": map[string]string{"DeploymentID": deploymentID}},
					{"term": map[string]string{"ParentServiceID": ""}},
					{"term": map[string]string{"Name": name}},
					{"term": map[string]string{"type": kind}},
				},
			},
		},
	}

	search, err := elastic.BuildSearchRequest(query, "controlplane")
	if err != nil {
		return nil, err
	}
	q := datastore.NewQuery(ctx)
	results, err := q.Execute(search)
	if err != nil {
		return nil, err
	}

	if results.Len() == 0 {
		return nil, nil
	} else if svcs, err := s.convert(results); err != nil {
		return nil, err
	} else {
		return &svcs[0], nil
	}
}

func (s *storeImpl) query(ctx datastore.Context, query interface{}) ([]Service, error) {
	q := datastore.NewQuery(ctx)

	search, err := elastic.BuildSearchRequest(query, "controlplane")
	if err != nil {
		return nil, err
	}

	results, err := q.Execute(search)
	if err != nil {
		return nil, err
	}
	return s.convert(results)
}

// fillAdditionalInfo fills the service object with additional information
// that amends or overrides what was retrieved from elastic
func (s *storeImpl) fillAdditionalInfo(svc *Service) {
	plog.WithFields(log.Fields{
		"serviceId":   svc.ID,
		"serviceName": svc.Name,
	}).Debug("Adding additional info to Elastic result")
	s.fillConfig(svc)

	// Update the service from volatile cached data.
	cacheEntry, ok := s.getVolatileInfo(svc.ID) // Uses Mutex RLock
	if ok {
		s.updateServiceFromVolatileService(svc, cacheEntry)
	} else {
		// If there's no ZK data, make sure the service is stopped.
		svc.DesiredState = int(SVCStop)
	}
}

// fillConfig fills in the ConfigFiles values
func (s *storeImpl) fillConfig(svc *Service) {
	svc.ConfigFiles = make(map[string]servicedefinition.ConfigFile)
	for key, val := range svc.OriginalConfigs {
		svc.ConfigFiles[key] = val
	}
}

func (s *storeImpl) convert(results datastore.Results) ([]Service, error) {
	svcs := make([]Service, results.Len())
	for idx := range svcs {
		var svc Service
		err := results.Get(idx, &svc)
		if err != nil {
			return nil, err
		}
		s.fillAdditionalInfo(&svc)
		svcs[idx] = svc
	}
	return svcs, nil
}

//Key creates a Key suitable for getting, putting and deleting Services
func Key(id string) datastore.Key {
	return datastore.NewKey(kind, id)
}

// Take the list of services and append services updated since 'since'
func (s *storeImpl) addUpdatedServicesFromCache(ctx datastore.Context, svcs []Service, since time.Time) ([]Service, error) {
	// Make a map of service ids so we don't duplicate services from our cache.
	svcMap := make(map[string]struct{})
	for _, svc := range svcs {
		svcMap[svc.ID] = struct{}{}
	}

	// If getting these one at a time turns out to be hard on elastic, we can
	// later try batching the elastic queries for sets of N ids until we go
	// through the whole list with a new elastic search.
	for _, cacheEntry := range s.getUpdatedCacheEntries(since) { // single Mutex RLock
		// Don't add services already in the list.
		if _, ok := svcMap[cacheEntry.ID]; !ok {
			// Query the service from elastic.  We already have the cached
			// data, so we save making a mutex lock here for every service
			// by called s.get() and updating the service without needing
			// additional mutex locks.
			if svc, err := s.get(ctx, cacheEntry.ID); err != nil {
				return svcs, err
			} else {
				// Fill additional info without a mutex lock.
				s.fillConfig(svc)
				s.updateServiceFromVolatileService(svc, cacheEntry)
				svcs = append(svcs, *svc)
			}
		} else {
			plog.WithField("serviceid", cacheEntry.ID).Debug("Skipping service because it is already cached")
		}
	}
	return svcs, nil
}

// Update all properties of the service with data from our volatile structure. No
// mutex lock needed.
func (s *storeImpl) updateServiceFromVolatileService(svc *Service, cacheEntry volatileService) {
	svc.DesiredState = cacheEntry.DesiredState
	svc.CurrentState = cacheEntry.CurrentState
}

// updateVolatileInfo updates the local cache for volatile information
func (s *storeImpl) updateVolatileInfo(serviceID string, desiredState int, updatedAt time.Time) error {
	// Only update desired state.  Current state should only be set explicitly
	return s.updateDesiredState(serviceID, desiredState, updatedAt)
}

// updateDesiredState updates the local cache for desired state
func (s *storeImpl) updateDesiredState(serviceID string, desiredState int, updatedAt time.Time) error {
	// Validate desired state
	if err := validation.IntIn(desiredState, int(SVCRun), int(SVCStop), int(SVCPause)); err != nil {
		plog.WithFields(log.Fields{
			"serviceID":    serviceID,
			"desiredState": desiredState,
		}).Debug("Invalid Desired State")
		return err
	}

	plog.WithFields(log.Fields{
		"serviceID":    serviceID,
		"desiredState": desiredState,
		"updatedAt":    updatedAt,
	}).Debug("Saving desired state in cache")

	s.serviceCacheLock.Lock()
	defer s.serviceCacheLock.Unlock()
	var cacheEntry volatileService

	cacheEntry, ok := s.serviceCache[serviceID]
	if !ok {
		cacheEntry = volatileService{
			ID:           serviceID,
			DesiredState: desiredState,
			UpdatedAt:    updatedAt,
		}
	} else {
		cacheEntry.DesiredState = desiredState
		cacheEntry.UpdatedAt = updatedAt
	}

	s.serviceCache[serviceID] = cacheEntry

	return nil
}

// updateDesiredState updates the local cache for current state
func (s *storeImpl) updateCurrentState(serviceID string, currentState string, updatedAt time.Time) error {
	// Validate desired state
	if err := ServiceCurrentState(currentState).Validate(); err != nil {
		plog.WithFields(log.Fields{
			"serviceID":    serviceID,
			"currentState": currentState,
		}).Debug("Invalid Current State")
		return err
	}

	plog.WithFields(log.Fields{
		"serviceID":    serviceID,
		"currentState": currentState,
		"updatedAt":    updatedAt,
	}).Debug("Saving current state in cache")

	s.serviceCacheLock.Lock()
	defer s.serviceCacheLock.Unlock()
	var cacheEntry volatileService

	cacheEntry, ok := s.serviceCache[serviceID]
	if !ok {
		cacheEntry = volatileService{
			ID:           serviceID,
			CurrentState: currentState,
			UpdatedAt:    updatedAt,
		}
	} else {
		cacheEntry.CurrentState = currentState
		cacheEntry.UpdatedAt = updatedAt
	}

	s.serviceCache[serviceID] = cacheEntry

	return nil
}

// removeVolatileInfo removes the service's information from the local cache
func (s *storeImpl) removeVolatileInfo(serviceID string) {
	s.serviceCacheLock.Lock()
	defer s.serviceCacheLock.Unlock()
	delete(s.serviceCache, serviceID)
}

// Returns the volatile data for a service id.
func (s *storeImpl) getVolatileInfo(serviceID string) (volatileService, bool) {
	s.serviceCacheLock.RLock()
	defer s.serviceCacheLock.RUnlock()
	cacheEntry, ok := s.serviceCache[serviceID]
	if ok && cacheEntry.CurrentState == "" {
		cacheEntry.CurrentState = string(SVCCSUnknown)
	}
	return cacheEntry, ok
}

// Returns the list of cache entries updated since the given time.
func (s *storeImpl) getUpdatedCacheEntries(since time.Time) []volatileService {
	s.serviceCacheLock.RLock()
	defer s.serviceCacheLock.RUnlock()

	logger := plog.WithFields(log.Fields{
		"since": since,
	})
	logger.Debug("Querying services updated since")

	cacheEntries := []volatileService{}
	for _, cacheEntry := range s.serviceCache {
		if cacheEntry.UpdatedAt.After(since) {
			logger.WithField("serviceid", cacheEntry.ID).Debug("Adding cache entry")
			cacheEntries = append(cacheEntries, cacheEntry)
		}
	}
	logger.WithField("count", len(cacheEntries)).Debug("Returning cached entries")
	return cacheEntries
}
