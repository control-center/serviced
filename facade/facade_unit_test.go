// Copyright 2016 The Serviced Authors.
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

// +build unit

package facade_test

import (
	datastoremocks "github.com/control-center/serviced/datastore/mocks"
	dfsmocks "github.com/control-center/serviced/dfs/mocks"
	hostmocks "github.com/control-center/serviced/domain/host/mocks"
	poolmocks "github.com/control-center/serviced/domain/pool/mocks"
	registrymocks "github.com/control-center/serviced/domain/registry/mocks"
	servicemocks "github.com/control-center/serviced/domain/service/mocks"
	templatemocks "github.com/control-center/serviced/domain/servicetemplate/mocks"
	zzkmocks "github.com/control-center/serviced/facade/mocks"
	"github.com/control-center/serviced/facade"
	"github.com/stretchr/testify/mock"
	. "gopkg.in/check.v1"
)

var _ = Suite(&FacadeUnitTest{})

type FacadeUnitTest struct {
	Facade        *facade.Facade
	ctx           *datastoremocks.Context
	zzk           *zzkmocks.ZZK
	dfs           *dfsmocks.DFS
	hostStore     *hostmocks.Store
	poolStore     *poolmocks.Store
	registryStore *registrymocks.ImageRegistryStore
	serviceStore  *servicemocks.Store
	templateStore *templatemocks.Store
}

func (ft *FacadeUnitTest) SetUpSuite(c *C) {
	ft.Facade = facade.New()
}

func (ft *FacadeUnitTest) SetUpTest(c *C) {
	ft.ctx = &datastoremocks.Context{}

	ft.dfs = &dfsmocks.DFS{}
	ft.Facade.SetDFS(ft.dfs)

	ft.hostStore = &hostmocks.Store{}
	ft.Facade.SetHostStore(ft.hostStore)

	ft.poolStore = &poolmocks.Store{}
	ft.Facade.SetPoolStore(ft.poolStore)

	ft.registryStore = &registrymocks.ImageRegistryStore{}
	ft.Facade.SetRegistryStore(ft.registryStore)

	ft.serviceStore = &servicemocks.Store{}
	ft.Facade.SetServiceStore(ft.serviceStore)

	ft.templateStore = &templatemocks.Store{}
	ft.Facade.SetTemplateStore(ft.templateStore)

	ft.zzk = &zzkmocks.ZZK{}
	ft.Facade.SetZZK(ft.zzk)
}

// Mock all DFS locking operations into no-ops
func (ft *FacadeUnitTest) setupMockDFSLocking() {
	ft.dfs.On("Lock", mock.AnythingOfType("string")).Return()
	ft.dfs.On("LockWithTimeout", mock.AnythingOfType("string"), mock.AnythingOfType("time.Duration")).Return(nil)
	ft.dfs.On("Unlock").Return()
}
