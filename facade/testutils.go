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
	"fmt"
	"time"

	auditmocks "github.com/control-center/serviced/audit/mocks"
	"github.com/control-center/serviced/auth"
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
	"github.com/control-center/serviced/scheduler/servicestatemanager"
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
	ssm    *servicestatemanager.BatchServiceStateManager
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
	mockLogger := &auditmocks.Logger{}
	mockLogger.On("Message", mock.AnythingOfType("*datastore.context"), mock.AnythingOfType("string")).Return(mockLogger)
	mockLogger.On("Message", mock.AnythingOfType("*mocks.Context"), mock.AnythingOfType("string")).Return(mockLogger)
	mockLogger.On("Action", mock.AnythingOfType("string")).Return(mockLogger)
	mockLogger.On("Type", mock.AnythingOfType("string")).Return(mockLogger)
	mockLogger.On("ID", mock.AnythingOfType("string")).Return(mockLogger)
	mockLogger.On("Entity", mock.AnythingOfType("*pool.ResourcePool")).Return(mockLogger)
	mockLogger.On("Entity", mock.AnythingOfType("*service.Service")).Return(mockLogger)
	mockLogger.On("Entity", mock.AnythingOfType("*servicestatemanager.CancellableService")).Return(mockLogger)
	mockLogger.On("Entity", mock.AnythingOfType("*host.Host")).Return(mockLogger)
	mockLogger.On("Entity", mock.AnythingOfType("*servicetemplate.ServiceTemplate")).Return(mockLogger)
	mockLogger.On("Entity", mock.AnythingOfType("*addressassignment.AddressAssignment")).Return(mockLogger)
	mockLogger.On("WithField", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(mockLogger)
	mockLogger.On("WithFields", mock.AnythingOfType("logrus.Fields")).Return(mockLogger)
	mockLogger.On("Error", mock.Anything)
	mockLogger.On("Succeeded", mock.Anything)
	mockLogger.On("SucceededIf", mock.AnythingOfType("bool"))
	mockLogger.On("Failed", mock.Anything)

	ft.Facade.SetAuditLogger(mockLogger)
	ft.dfs = &dfsmocks.DFS{}
	ft.Facade.SetDFS(ft.dfs)
	ft.setupMockDFS()

	// Create a master key pair
	pub, priv, _ := auth.GenerateRSAKeyPairPEM(nil)
	auth.LoadMasterKeysFromPEM(pub, priv)
}

func (ft *FacadeIntegrationTest) SetUpTest(c *gocheck.C) {
	ft.ElasticTest.SetUpTest(c)
	ft.zzk = &zzkmocks.ZZK{}
	ft.Facade.SetZZK(ft.zzk)
	ft.dfs = &dfsmocks.DFS{}
	ft.Facade.SetDFS(ft.dfs)
	ft.setupMockZZK(c)
	ft.setupMockDFS()
	ft.ssm = servicestatemanager.NewBatchServiceStateManager(ft.Facade, ft.CTX, 10*time.Second)
	ft.ssm.Start()
	ft.Facade.SetServiceStateManager(ft.ssm)
}

func (ft *FacadeIntegrationTest) setupMockZZK(c *gocheck.C) {
	ft.zzk.On("AddResourcePool", mock.AnythingOfType("*pool.ResourcePool")).Return(nil)
	ft.zzk.On("UpdateResourcePool", mock.AnythingOfType("*pool.ResourcePool")).Return(nil)
	ft.zzk.On("RemoveResourcePool", mock.AnythingOfType("string")).Return(nil)
	ft.zzk.On("AddVirtualIP", mock.AnythingOfType("*pool.VirtualIP")).Return(nil)
	ft.zzk.On("RemoveVirtualIP", mock.AnythingOfType("*pool.VirtualIP")).Return(nil)
	ft.zzk.On("AddHost", mock.AnythingOfType("*host.Host")).Return(nil)
	ft.zzk.On("UpdateHost", mock.AnythingOfType("*host.Host")).Return(nil)
	ft.zzk.On("RemoveHost", mock.AnythingOfType("*host.Host")).Return(nil)
	ft.zzk.On("UpdateService", mock.AnythingOfType("*datastore.context"), mock.AnythingOfType("string"), mock.AnythingOfType("*service.Service"), mock.AnythingOfType("bool"), mock.AnythingOfType("bool")).Return(nil)
	ft.zzk.On("UpdateServices", mock.AnythingOfType("*datastore.context"), mock.AnythingOfType("string"), mock.AnythingOfType("[]*service.Service"), mock.AnythingOfType("bool"), mock.AnythingOfType("bool")).Return(nil)
	ft.zzk.On("RemoveService", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
	ft.zzk.On("RemoveServiceEndpoints", mock.AnythingOfType("string")).Return(nil)
	ft.zzk.On("RemoveTenantExports", mock.AnythingOfType("string")).Return(nil)
	ft.zzk.On("SetRegistryImage", mock.AnythingOfType("*registry.Image")).Return(nil)
	ft.zzk.On("DeleteRegistryImage", mock.AnythingOfType("string")).Return(nil)
	ft.zzk.On("DeleteRegistryLibrary", mock.AnythingOfType("string")).Return(nil)
	ft.zzk.On("LockServices", ft.CTX, mock.AnythingOfType("[]service.ServiceDetails")).Return(nil)
	ft.zzk.On("UnlockServices", ft.CTX, mock.AnythingOfType("[]service.ServiceDetails")).Return(nil)
	ft.zzk.On("UnregisterDfsClients", mock.AnythingOfType("[]host.Host")).Return(nil)
	ft.zzk.On("UpdateInstanceCurrentState", ft.CTX, mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("int"), mock.AnythingOfType("service.InstanceCurrentState")).Return(nil)

	ft.zzk.On("WaitService", mock.AnythingOfType("*service.Service"), mock.AnythingOfType("service.DesiredState"),
		mock.AnythingOfType("<-chan interface {}")).Return(nil)
}

func (ft *FacadeIntegrationTest) setupMockDFS() {
	ft.dfs.On("Destroy", mock.AnythingOfType("string")).Return(nil)
}

func (ft *FacadeIntegrationTest) TearDownTest(c *gocheck.C) {
	ft.ssm.Shutdown()
}

func (ft *FacadeIntegrationTest) BeforeTest(suiteName, testName string) {
	fmt.Printf("Starting test %s\n", testName)
}

func (ft *FacadeIntegrationTest) Dfs() *dfsmocks.DFS {
	return ft.dfs
}
