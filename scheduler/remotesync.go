// Copyright 2015 The Serviced Authors.
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

package scheduler

import (
	"errors"
	"time"

	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/utils"
	"github.com/zenoss/glog"
)

// RemoteSyncDatastore contains the primary datastore information from which
// to sync.
type RemoteSyncDatastore interface {
	// GetUpdatedServices gets the list of updated serviced for the remote pool
	GetUpdatedServices(poolID string) ([]string, []service.Service, error)
}

// RemoteSyncInterface is the endpoint where the data is synced
type RemoteSyncInterface interface {
	// GetServices returns all the services within a pool
	GetServices(poolID string) ([]service.Service, error)
	// AddService creates  a new service
	AddService(svc *service.Service) error
	// UpdateService updates an existing service
	UpdateService(svc *service.Service) error
	// DeleteService deletes a service
	RemoveService(serviceID string) error
}

// RemoteServiceSync performs service synchronizations for services that
// originate in remote pools.
type RemoteServiceSync struct {
	ds     RemoteSyncDatastore
	iface  RemoteSyncInterface
	poolID string
}

// Purge looks up the remote data and performs the synchronization.
// Implements utils.TTL
func (sync *RemoteServiceSync) Purge(age time.Duration) (time.Duration, error) {
	// look up the services
	ids, svcs, err := sync.ds.GetUpdatedServices(sync.poolID)
	if err != nil {
		glog.Errorf("Could not look up remote services for pool %s: %s", sync.poolID, err)
		return 0, err
	}

	// build the datamap
	datamap := make(map[string]interface{})
	for _, id := range ids {
		datamap[id] = nil
	}
	for i, svc := range svcs {
		datamap[svc.ID] = &svcs[i]
	}

	// perform the sync
	if err := utils.Sync(sync, datamap); err != nil {
		glog.Errorf("Could not sync services for remote pool %s: %s", sync.poolID, err)
		return 0, err
	}
	return age, nil
}

// IDs return the complete list of service ids.
// Implements utils.Synchronizer
func (sync *RemoteServiceSync) IDs() ([]string, error) {
	svcs, err := sync.iface.GetServices(sync.poolID)
	if err != nil {
		glog.Errorf("Could not look up services in pool %s: %s", sync.poolID, err)
		return nil, err
	}

	ids := make([]string, len(svcs))
	for i, svc := range svcs {
		ids[i] = svc.ID
	}
	return ids, nil
}

// Create creates a new service.
// Implements utils.Synchronizer
func (sync *RemoteServiceSync) Create(data interface{}) error {
	svc, err := sync.convert(data)
	if err != nil {
		return err
	}
	return sync.iface.AddService(svc)
}

// Update updates existing service.
// Implements utils.Synchronizer
func (sync *RemoteServiceSync) Update(data interface{}) error {
	svc, err := sync.convert(data)
	if err != nil {
		return err
	}
	return sync.iface.UpdateService(svc)
}

// Delete deletes an existing service
// Implements utils.Synchronizer
func (sync *RemoteServiceSync) Delete(id string) error {
	return sync.iface.RemoveService(id)
}

// convert transforms an interface into domain service object
func (sync *RemoteServiceSync) convert(data interface{}) (*service.Service, error) {
	if svc, ok := data.(*service.Service); ok {
		return svc, nil
	}
	return nil, errors.New("invalid type")
}