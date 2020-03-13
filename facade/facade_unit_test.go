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

	auditmocks "github.com/control-center/serviced/audit/mocks"
	"github.com/control-center/serviced/auth"
	authmocks "github.com/control-center/serviced/auth/mocks"
	datastoremocks "github.com/control-center/serviced/datastore/mocks"
	dfsmocks "github.com/control-center/serviced/dfs/mocks"
	hostmocks "github.com/control-center/serviced/domain/host/mocks"
	keymocks "github.com/control-center/serviced/domain/hostkey/mocks"
	logfiltermocks "github.com/control-center/serviced/domain/logfilter/mocks"
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
	configfileStore  *configmocks.Store
	templateStore    *templatemocks.Store
	logfilterStore   *logfiltermocks.Store
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

	mockLogger := &auditmocks.Logger{}
	mockLogger.On("Message", mock.AnythingOfType("*datastore.context"), mock.AnythingOfType("string")).Return(mockLogger)
	mockLogger.On("Message", mock.AnythingOfType("*mocks.Context"), mock.AnythingOfType("string")).Return(mockLogger)
	mockLogger.On("Action", mock.AnythingOfType("string")).Return(mockLogger)
	mockLogger.On("Type", mock.AnythingOfType("string")).Return(mockLogger)
	mockLogger.On("ID", mock.AnythingOfType("string")).Return(mockLogger)
	mockLogger.On("Entity", mock.AnythingOfType("*pool.ResourcePool")).Return(mockLogger)
	mockLogger.On("Entity", mock.AnythingOfType("*service.Service")).Return(mockLogger)
	mockLogger.On("Entity", mock.AnythingOfType("*host.Host")).Return(mockLogger)
	mockLogger.On("WithField", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(mockLogger)
	mockLogger.On("WithFields", mock.AnythingOfType("logrus.Fields")).Return(mockLogger)
	mockLogger.On("Error", mock.Anything)
	mockLogger.On("Succeeded", mock.Anything)
	mockLogger.On("SucceededIf", mock.AnythingOfType("bool"))
	mockLogger.On("Failed", mock.Anything)
	ft.Facade.SetAuditLogger(mockLogger)

	ft.dfs = &dfsmocks.DFS{}
	ft.Facade.SetDFS(ft.dfs)

	ft.hostStore = &hostmocks.Store{}
	ft.Facade.SetHostStore(ft.hostStore)

	ft.hostkeyStore = &keymocks.Store{}
	ft.Facade.SetHostKeyStore(ft.hostkeyStore)

	ft.poolStore = &poolmocks.Store{}
	ft.Facade.SetPoolStore(ft.poolStore)

	ft.registryStore = &registrymocks.ImageRegistryStore{}
	ft.Facade.SetRegistryStore(ft.registryStore)

	ft.serviceStore = &servicemocks.Store{}
	ft.Facade.SetServiceStore(ft.serviceStore)

	ft.configfileStore = &configmocks.Store{}
	ft.Facade.SetServiceConfigFileStore(ft.configfileStore)

	ft.templateStore = &templatemocks.Store{}
	ft.Facade.SetServiceTemplateStore(ft.templateStore)

	ft.logfilterStore = &logfiltermocks.Store{}
	ft.Facade.SetLogFilterStore(ft.logfilterStore)

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
