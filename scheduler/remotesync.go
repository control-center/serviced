// Copyright 2014 The Serviced Authors.
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
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/domain/pool"
	"github.com/control-center/serviced/domain/service"
)

func (s *scheduler) GetResourcePools() ([]pool.ResourcePool, error) {
	return s.facade.GetResourcePools(datastore.Get())
}

func (s *scheduler) AddUpdateResourcePool(pool *pool.ResourcePool) error {
	if p, err := s.facade.GetResourcePool(datastore.Get(), pool.ID); err != nil {
		return err
	} else if p == nil {
		return s.facade.AddResourcePool(datastore.Get(), pool)
	}

	return s.facade.UpdateResourcePool(datastore.Get(), pool)
}

func (s *scheduler) RemoveResourcePool(id string) error {
	return s.facade.RemoveResourcePool(datastore.Get(), id)
}

func (s *scheduler) GetServicesByPool(id string) ([]service.Service, error) {
	return s.facade.GetServicesByPool(datastore.Get(), id)
}

func (s *scheduler) AddUpdateService(svc *service.Service) error {
	if sv, err := s.facade.GetService(datastore.Get(), svc.ID); err != nil {
		return err
	} else if sv == nil {
		return s.facade.AddService(datastore.Get(), *svc)
	}

	return s.facade.UpdateService(datastore.Get(), *svc)
}

func (s *scheduler) RemoveService(id string) error {
	return s.facade.RemoveService(datastore.Get(), id)
}

func (s *scheduler) GetHostsByPool(id string) ([]host.Host, error) {
	return s.facade.FindHostsInPool(datastore.Get(), id)
}

func (s *scheduler) AddUpdateHost(host *host.Host) error {
	if h, err := s.facade.GetHost(datastore.Get(), host.ID); err != nil {
		return err
	} else if h == nil {
		return s.facade.AddHost(datastore.Get(), host)
	}

	return s.facade.UpdateHost(datastore.Get(), host)
}

func (s *scheduler) RemoveHost(id string) error {
	return s.facade.RemoveHost(datastore.Get(), id)
}
