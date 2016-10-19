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
	"time"

	"github.com/control-center/serviced/auth"
	authmocks "github.com/control-center/serviced/auth/mocks"
	datastoremocks "github.com/control-center/serviced/datastore/mocks"
	dfsmocks "github.com/control-center/serviced/dfs/mocks"
	hostmocks "github.com/control-center/serviced/domain/host/mocks"
	keymocks "github.com/control-center/serviced/domain/hostkey/mocks"
	poolmocks "github.com/control-center/serviced/domain/pool/mocks"
	registrymocks "github.com/control-center/serviced/domain/registry/mocks"
	servicemocks "github.com/control-center/serviced/domain/service/mocks"
	configmocks "github.com/control-center/serviced/domain/serviceconfigfile/mocks"
	templatemocks "github.com/control-center/serviced/domain/servicetemplate/mocks"
	"github.com/control-center/serviced/facade"
	zzkmocks "github.com/control-center/serviced/facade/mocks"
	"github.com/control-center/serviced/metrics"
	"github.com/stretchr/testify/mock"
	. "gopkg.in/check.v1"
)

var _ = Suite(&FacadeUnitTest{})

type FacadeUnitTest struct {
	Facade           *facade.Facade
	ctx              *datastoremocks.Context
	zzk              *zzkmocks.ZZK
	dfs              *dfsmocks.DFS
	hostStore        *hostmocks.Store
	poolStore        *poolmocks.Store
	hostkeyStore     *keymocks.Store
	registryStore    *registrymocks.ImageRegistryStore
	serviceStore     *servicemocks.Store
	configStore      *configmocks.Store
	templateStore    *templatemocks.Store
	metricsClient    *zzkmocks.MetricsClient
	hostauthregistry *authmocks.HostExpirationRegistryInterface
}

func (ft *FacadeUnitTest) SetUpSuite(c *C) {
	ft.Facade = facade.New()

	// Create a master key pair
	pub, priv, _ := auth.GenerateRSAKeyPairPEM(nil)
	auth.LoadMasterKeysFromPEM(pub, priv)
}

func (ft *FacadeUnitTest) SetUpTest(c *C) {
	ft.ctx = &datastoremocks.Context{}

	ft.dfs = &dfsmocks.DFS{}
	ft.Facade.SetDFS(ft.dfs)

	ft.hostStore = &hostmocks.Store{}
	ft.Facade.SetHostStore(ft.hostStore)

	ft.hostkeyStore = &keymocks.Store{}
	ft.Facade.SetHostkeyStore(ft.hostkeyStore)

	ft.poolStore = &poolmocks.Store{}
	ft.Facade.SetPoolStore(ft.poolStore)

	ft.registryStore = &registrymocks.ImageRegistryStore{}
	ft.Facade.SetRegistryStore(ft.registryStore)

	ft.serviceStore = &servicemocks.Store{}
	ft.Facade.SetServiceStore(ft.serviceStore)

	ft.configStore = &configmocks.Store{}
	ft.Facade.SetConfigStore(ft.configStore)

	ft.templateStore = &templatemocks.Store{}
	ft.Facade.SetTemplateStore(ft.templateStore)

	ft.zzk = &zzkmocks.ZZK{}
	ft.Facade.SetZZK(ft.zzk)

	ft.metricsClient = &zzkmocks.MetricsClient{}
	ft.Facade.SetMetricsClient(ft.metricsClient)

	ft.hostauthregistry = &authmocks.HostExpirationRegistryInterface{}
	ft.Facade.SetHostExpirationRegistry(ft.hostauthregistry)

	ft.hostauthregistry.On("Remove", mock.AnythingOfType("string")).Return()

	ft.ctx.On("Metrics").Return(metrics.NewMetrics())
}

// Mock all DFS locking operations into no-ops
func (ft *FacadeUnitTest) setupMockDFSLocking() {
	ft.dfs.On("Lock", mock.AnythingOfType("string")).Return()
	ft.dfs.On("LockWithTimeout", mock.AnythingOfType("string"), mock.AnythingOfType("time.Duration")).Return(nil)
	ft.dfs.On("Unlock").Return()
}

type timeChecker struct {
	*CheckerInfo
}

func (c *timeChecker) Check(params []interface{}, names []string) (result bool, error string) {
	var ok bool
	var first, second time.Time

	first, ok = params[0].(time.Time)
	if !ok {
		return false, "First parameter is not a Time"
	}
	second, ok = params[1].(time.Time)
	if !ok {
		return false, "Second parameter is not an Time"
	}
	return first.Equal(second), ""
}

var TimeEqual Checker = &timeChecker{&CheckerInfo{Name: "TimeEqual", Params: []string{"Obtained", "Expected"}}}
