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
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/datastore/elastic"
	dfsmocks "github.com/control-center/serviced/dfs/mocks"
	"github.com/control-center/serviced/domain/addressassignment"
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/domain/pool"
	"github.com/control-center/serviced/domain/registry"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/serviceconfigfile"
	"github.com/control-center/serviced/domain/servicetemplate"
	"github.com/control-center/serviced/domain/user"
	zzkmocks "github.com/control-center/serviced/facade/mocks"
	"github.com/stretchr/testify/mock"
	gocheck "gopkg.in/check.v1"
)

// FacadeIntegrationTest used for running integration tests where a facade type is needed.
type FacadeIntegrationTest struct {
	elastic.ElasticTest
	CTX    datastore.Context
	Facade *Facade
	zzk    *zzkmocks.ZZK
	dfs    *dfsmocks.DFS
}

var _ = gocheck.Suite(&FacadeIntegrationTest{})

//SetUpSuite sets up test suite
func (ft *FacadeIntegrationTest) SetUpSuite(c *gocheck.C) {
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
	ft.Mappings = append(ft.Mappings, registry.MAPPING)

	ft.ElasticTest.SetUpSuite(c)
	datastore.Register(ft.Driver())
	ft.CTX = datastore.Get()

	ft.Facade = New()
	ft.dfs = &dfsmocks.DFS{}
	ft.Facade.SetDFS(ft.dfs)
	ft.setupMockDFS()
}

func (ft *FacadeIntegrationTest) SetUpTest(c *gocheck.C) {
	ft.ElasticTest.SetUpTest(c)
	ft.zzk = &zzkmocks.ZZK{}
	ft.Facade.SetZZK(ft.zzk)
	ft.dfs = &dfsmocks.DFS{}
	ft.Facade.SetDFS(ft.dfs)
	ft.setupMockZZK()
	ft.setupMockDFS()
	LogstashContainerReloader = reloadLogstashContainerStub
}

func (ft *FacadeIntegrationTest) setupMockZZK() {
	ft.zzk.On("AddResourcePool", mock.AnythingOfType("*pool.ResourcePool")).Return(nil)
	ft.zzk.On("UpdateResourcePool", mock.AnythingOfType("*pool.ResourcePool")).Return(nil)
	ft.zzk.On("RemoveResourcePool", mock.AnythingOfType("string")).Return(nil)
	ft.zzk.On("AddVirtualIP", mock.AnythingOfType("*pool.VirtualIP")).Return(nil)
	ft.zzk.On("RemoveVirtualIP", mock.AnythingOfType("*pool.VirtualIP")).Return(nil)
	ft.zzk.On("AddHost", mock.AnythingOfType("*host.Host")).Return(nil)
	ft.zzk.On("UpdateHost", mock.AnythingOfType("*host.Host")).Return(nil)
	ft.zzk.On("RemoveHost", mock.AnythingOfType("*host.Host")).Return(nil)
	ft.zzk.On("UpdateService", mock.AnythingOfType("*service.Service"), mock.AnythingOfType("bool"), mock.AnythingOfType("bool")).Return(nil)
	ft.zzk.On("RemoveService", mock.AnythingOfType("*service.Service")).Return(nil)
	ft.zzk.On("SetRegistryImage", mock.AnythingOfType("*registry.Image")).Return(nil)
	ft.zzk.On("DeleteRegistryImage", mock.AnythingOfType("string")).Return(nil)
	ft.zzk.On("DeleteRegistryLibrary", mock.AnythingOfType("string")).Return(nil)
	ft.zzk.On("LockServices", mock.AnythingOfType("[]service.Service")).Return(nil)
	ft.zzk.On("UnlockServices", mock.AnythingOfType("[]service.Service")).Return(nil)

}

func (ft *FacadeIntegrationTest) setupMockDFS() {
	ft.dfs.On("Destroy", mock.AnythingOfType("string")).Return(nil)
}

func (ft *FacadeIntegrationTest) TearDownTest(c *gocheck.C) {
}

func reloadLogstashContainerStub(_ datastore.Context, _ FacadeInterface) error {
	return nil
}
