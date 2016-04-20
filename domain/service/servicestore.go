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
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain/servicedefinition"
	"github.com/zenoss/elastigo/search"

	"errors"
	"strings"
	"time"
)

// NewStore creates a Service store
func NewStore() Store {
	return &storeImpl{}
}

// Store type for interacting with Service persistent storage
type Store interface {
	// Put adds or updates a Service
	Put(ctx datastore.Context, svc *Service) error

	// Get a Service by id. Return ErrNoSuchEntity if not found
	Get(ctx datastore.Context, id string) (*Service, error)

	// Delete removes the a Service if it exists
	Delete(ctx datastore.Context, id string) error

	// GetServices returns all services
	GetServices(ctx datastore.Context) ([]Service, error)

	// GetUpdatedServices returns all services updated since "since" time.Duration ago
	GetUpdatedServices(ctx datastore.Context, since time.Duration) ([]Service, error)

	// GetTaggedServices returns services with the given tags
	GetTaggedServices(ctx datastore.Context, tags ...string) ([]Service, error)

	// GetServicesByPool returns services with the given pool id
	GetServicesByPool(ctx datastore.Context, poolID string) ([]Service, error)

	// GetServicesByDeployment returns services with the given deployment id
	GetServicesByDeployment(ctx datastore.Context, deploymentID string) ([]Service, error)

	// GetChildServices returns services that are children of the given parent service id
	GetChildServices(ctx datastore.Context, parentID string) ([]Service, error)

	FindChildService(ctx datastore.Context, deploymentID, parentID, serviceName string) (*Service, error)

	// FindTenantByDeployment returns the tenant service for a given deployment id and service name
	FindTenantByDeploymentID(ctx datastore.Context, deploymentID, name string) (*Service, error)
}

type storeImpl struct {
	ds datastore.DataStore
}

// Put adds or updates a Service
func (s *storeImpl) Put(ctx datastore.Context, svc *Service) error {
	//No need to store ConfigFiles
	svc.ConfigFiles = make(map[string]servicedefinition.ConfigFile)

	return s.ds.Put(ctx, Key(svc.ID), svc)
}

// Get a Service by id. Return ErrNoSuchEntity if not found
func (s *storeImpl) Get(ctx datastore.Context, id string) (*Service, error) {
	svc := &Service{}
	if err := s.ds.Get(ctx, Key(id), svc); err != nil {
		return nil, err
	}

	//Copy original config files
	fillConfig(svc)

	return svc, nil
}

// Delete removes the a Service if it exists
func (s *storeImpl) Delete(ctx datastore.Context, id string) error {
       return s.ds.Delete(ctx, Key(id))
}

// GetServices returns all services
func (s *storeImpl) GetServices(ctx datastore.Context) ([]Service, error) {
	return query(ctx, "_exists_:ID")
}

// GetUpdatedServices returns all services updated since "since" time.Duration ago
func (s *storeImpl) GetUpdatedServices(ctx datastore.Context, since time.Duration) ([]Service, error) {
	q := datastore.NewQuery(ctx)
	t0 := time.Now().Add(-since).Format(time.RFC3339)
	elasticQuery := search.Query().Range(search.Range().Field("UpdatedAt").From(t0)).Search("_exists_:ID")
	search := search.Search("controlplane").Type(kind).Size("50000").Query(elasticQuery)
	results, err := q.Execute(search)
	if err != nil {
		return nil, err
	}
	return convert(results)
}

// GetTaggedServices returns services with the given tags
func (s *storeImpl) GetTaggedServices(ctx datastore.Context, tags ...string) ([]Service, error) {
	if len(tags) == 0 {
		return nil, errors.New("empty tags not allowed")
	}
	qs := strings.Join(tags, " AND ")
	return query(ctx, qs)
}

// GetServicesByPool returns services with the given pool id
func (s *storeImpl) GetServicesByPool(ctx datastore.Context, poolID string) ([]Service, error) {
	id := strings.TrimSpace(poolID)
	if id == "" {
		return nil, errors.New("empty poolID not allowed")
	}
	q := datastore.NewQuery(ctx)
	query := search.Query().Term("PoolID", id)
	search := search.Search("controlplane").Type(kind).Size("50000").Query(query)
	results, err := q.Execute(search)
	if err != nil {
		return nil, err
	}
	return convert(results)
}

// GetServicesByDeployment returns services with the given deployment id
func (s *storeImpl) GetServicesByDeployment(ctx datastore.Context, deploymentID string) ([]Service, error) {
	id := strings.TrimSpace(deploymentID)
	if id == "" {
		return nil, errors.New("empty deploymentID not allowed")
	}
	q := datastore.NewQuery(ctx)
	query := search.Query().Term("DeploymentID", id)
	search := search.Search("controlplane").Type(kind).Size("50000").Query(query)
	results, err := q.Execute(search)
	if err != nil {
		return nil, err
	}
	return convert(results)
}

// GetChildServices returns services that are children of the given parent service id
func (s *storeImpl) GetChildServices(ctx datastore.Context, parentID string) ([]Service, error) {
	id := strings.TrimSpace(parentID)
	if id == "" {
		return nil, errors.New("empty parent service id not allowed")
	}
	q := datastore.NewQuery(ctx)
	query := search.Query().Term("ParentServiceID", id)
	search := search.Search("controlplane").Type(kind).Size("50000").Query(query)
	results, err := q.Execute(search)
	if err != nil {
		return nil, err
	}
	return convert(results)
}

func (s *storeImpl) FindChildService(ctx datastore.Context, deploymentID, parentID, serviceName string) (*Service, error) {
	parentID = strings.TrimSpace(parentID)

	if deploymentID = strings.TrimSpace(deploymentID); deploymentID == "" {
		return nil, errors.New("empty deployment ID not allowed")
	} else if serviceName = strings.TrimSpace(serviceName); serviceName == "" {
		return nil, errors.New("empty service name not allowed")
	}

	search := search.Search("controlplane").Type(kind).Filter(
		"and",
		search.Filter().Terms("DeploymentID", deploymentID),
		search.Filter().Terms("ParentServiceID", parentID),
		search.Filter().Terms("Name", serviceName),
	)

	q := datastore.NewQuery(ctx)
	results, err := q.Execute(search)
	if err != nil {
		return nil, err
	}

	if results.Len() == 0 {
		return nil, nil
	} else if svcs, err := convert(results); err != nil {
		return nil, err
	} else {
		return &svcs[0], nil
	}
}

// FindTenantByDeployment returns the tenant service for a given deployment id and service name
func (s *storeImpl) FindTenantByDeploymentID(ctx datastore.Context, deploymentID, name string) (*Service, error) {
	if deploymentID = strings.TrimSpace(deploymentID); deploymentID == "" {
		return nil, errors.New("empty deployment ID not allowed")
	} else if name = strings.TrimSpace(name); name == "" {
		return nil, errors.New("empty service name not allowed")
	}

	search := search.Search("controlplane").Type(kind).Filter(
		"and",
		search.Filter().Terms("DeploymentID", deploymentID),
		search.Filter().Terms("Name", name),
		search.Filter().Terms("ParentServiceID", ""),
	)

	q := datastore.NewQuery(ctx)
	results, err := q.Execute(search)
	if err != nil {
		return nil, err
	}

	if results.Len() == 0 {
		return nil, nil
	} else if svcs, err := convert(results); err != nil {
		return nil, err
	} else {
		return &svcs[0], nil
	}
}

func query(ctx datastore.Context, query string) ([]Service, error) {
	q := datastore.NewQuery(ctx)
	elasticQuery := search.Query().Search(query)
	search := search.Search("controlplane").Type(kind).Size("50000").Query(elasticQuery)
	results, err := q.Execute(search)
	if err != nil {
		return nil, err
	}
	return convert(results)
}

func fillConfig(svc *Service) {
	svc.ConfigFiles = make(map[string]servicedefinition.ConfigFile)
	for key, val := range svc.OriginalConfigs {
		svc.ConfigFiles[key] = val
	}
}

func convert(results datastore.Results) ([]Service, error) {
	svcs := make([]Service, results.Len())
	for idx := range svcs {
		var svc Service
		err := results.Get(idx, &svc)
		if err != nil {
			return nil, err
		}
		fillConfig(&svc)
		svcs[idx] = svc
	}
	return svcs, nil
}

//Key creates a Key suitable for getting, putting and deleting Services
func Key(id string) datastore.Key {
	return datastore.NewKey(kind, id)
}

//confFileKey creates a Key suitable for getting, putting and deleting svcConfigFile
func confFileKey(id string) datastore.Key {
	return datastore.NewKey(confKind, id)
}

var (
	kind     = "service"
	confKind = "serviceconfig"
)
