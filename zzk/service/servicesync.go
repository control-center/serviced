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

package service

import (
	"errors"

	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/utils"
)

// ServiceSync is the zookeeper synchronization object for services
type ServiceSync struct {
	conn   client.Connection // the global client connector
	poolID string
}

// SyncServices performs the service synchronization
func SyncServices(conn client.Connection, poolID string, services []service.Service) error {
	servicemap := make(map[string]interface{})
	for i, service := range services {
		servicemap[service.ID] = &services[i]
	}
	return utils.Sync(&ServiceSync{conn, poolID}, servicemap)
}

// IDs returns the current data by id
func (sync *ServiceSync) IDs() ([]string, error) {
	if ids, err := sync.conn.Children(poolpath(sync.poolID, servicepath())); err == client.ErrNoNode {
		return []string{}, nil
	} else {
		return ids, err
	}
}

// Create creates the new object data
func (sync *ServiceSync) Create(data interface{}) error {
	path, service, err := sync.convert(data)
	if err != nil {
		return err
	}
	var node ServiceNode
	if err := sync.conn.Create(path, &node); err != nil {
		return err
	}
	node.Service = service
	return sync.conn.Set(path, &node)
}

// Update updates the existing object data
func (sync *ServiceSync) Update(data interface{}) error {
	path, service, err := sync.convert(data)
	if err != nil {
		return err
	}
	var node ServiceNode
	if err := sync.conn.Get(path, &node); err != nil {
		return err
	}
	node.Service = service
	return sync.conn.Set(path, &node)
}

// Delete deletes an existing service by its id
func (sync *ServiceSync) Delete(id string) error {
	return sync.conn.Delete(poolpath(sync.poolID, servicepath(id)))
}

// convert transforms the generic object data into its proper type
func (sync *ServiceSync) convert(data interface{}) (string, *service.Service, error) {
	if service, ok := data.(*service.Service); ok {
		return poolpath(sync.poolID, servicepath(service.ID)), service, nil
	}
	return "", nil, errors.New("could not convert to a service object")
}