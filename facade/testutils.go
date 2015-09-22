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

// +build integration

package facade

import (
	"github.com/control-center/serviced/commons/docker/test"
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/datastore/elastic"
	"github.com/control-center/serviced/domain/addressassignment"
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/domain/pool"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/serviceconfigfile"
	"github.com/control-center/serviced/domain/servicestate"
	"github.com/control-center/serviced/domain/servicetemplate"
	"github.com/control-center/serviced/domain/user"
	gocheck "gopkg.in/check.v1"
)

//FacadeTest used for running tests where a facade type is needed.
type FacadeTest struct {
	elastic.ElasticTest
	CTX    datastore.Context
	Facade *Facade
	mockRegistry *test.MockDockerRegistry
}

//SetUpSuite sets up test suite
func (ft *FacadeTest) SetUpSuite(c *gocheck.C) {
	//set up index and mappings before setting up elastic
	ft.Index = "controlplane"
	if ft.Mappings == nil {
		ft.Mappings = make([]elastic.Mapping, 0)
	}
	ft.Mappings = append(ft.Mappings, host.MAPPING)
	ft.Mappings = append(ft.Mappings, pool.MAPPING)
	ft.Mappings = append(ft.Mappings, service.MAPPING)
	ft.Mappings = append(ft.Mappings, servicetemplate.MAPPING)
	ft.Mappings = append(ft.Mappings, addressassignment.MAPPING)
	ft.Mappings = append(ft.Mappings, serviceconfigfile.MAPPING)
	ft.Mappings = append(ft.Mappings, user.MAPPING)

	ft.ElasticTest.SetUpSuite(c)
	datastore.Register(ft.Driver())
	ft.CTX = datastore.Get()

	ft.Facade = New("localhost:5000")

	//mock out ZK calls to no ops
	zkAPI = func(f *Facade) zkfuncs { return &zkMock{} }
}

func (ft *FacadeTest) SetUpTest(c *gocheck.C) {
	ft.ElasticTest.SetUpTest(c)
}

func (ft *FacadeTest) TearDownTest(c *gocheck.C) {
	if ft.mockRegistry != nil {
		ft.mockRegistry = nil
		ft.Facade.registry = nil
	}
}

func (ft *FacadeTest) setupMockRegistry() {
	ft.mockRegistry = &test.MockDockerRegistry{}
	ft.Facade.registry = ft.mockRegistry
}

type zkMock struct {
}

func (z *zkMock) UpdateService(svc *service.Service) error {
	return nil
}

func (z *zkMock) RemoveService(svc *service.Service) error {
	return nil
}

func (z *zkMock) GetServiceStates(poolID string, serviceStates *[]servicestate.ServiceState, serviceIds ...string) error {
	return nil
}

func (z *zkMock) StopServiceInstance(poolID, hostID, stateID string) error {
	return nil
}

func (z *zkMock) AddHost(h *host.Host) error {
	return nil
}

func (z *zkMock) UpdateHost(h *host.Host) error {
	return nil
}

func (z *zkMock) RemoveHost(h *host.Host) error {
	return nil
}

func (z *zkMock) GetActiveHosts(poolID string, hosts *[]string) error {
	return nil
}

func (z *zkMock) AddVirtualIP(vip *pool.VirtualIP) error {
	return nil
}

func (z *zkMock) RemoveVirtualIP(vip *pool.VirtualIP) error {
	return nil
}

func (z *zkMock) AddResourcePool(pool *pool.ResourcePool) error {
	return nil
}

func (z *zkMock) UpdateResourcePool(pool *pool.ResourcePool) error {
	return nil
}

func (z *zkMock) RemoveResourcePool(poolID string) error {
	return nil
}

func (z *zkMock) CheckRunningVHost(vhostName, serviceID string) error {
	return nil
}

func (z *zkMock) WaitService(service *service.Service, state service.DesiredState, cancel <-chan interface{}) error {
	return nil
}
