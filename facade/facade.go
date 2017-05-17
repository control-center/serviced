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

package facade

import (
	"time"

	"github.com/control-center/serviced/audit"
	"github.com/control-center/serviced/auth"
	"github.com/control-center/serviced/dfs"
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/domain/hostkey"
	"github.com/control-center/serviced/domain/pool"
	"github.com/control-center/serviced/domain/registry"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/serviceconfigfile"
	"github.com/control-center/serviced/domain/servicetemplate"
	"github.com/control-center/serviced/domain/user"
	"github.com/control-center/serviced/health"
	"github.com/control-center/serviced/logging"
	"github.com/control-center/serviced/metrics"
	"github.com/control-center/serviced/scheduler/servicestatemanager"
)

type MetricsClient interface {
	GetInstanceMemoryStats(time.Time, ...metrics.ServiceInstance) ([]metrics.MemoryUsageStats, error)
	GetAvailableStorage(time.Duration, string, ...string) (*metrics.StorageMetrics, error)
}

// instantiate the package logger
var plog = logging.PackageLogger()

// assert interface
var _ FacadeInterface = &Facade{}

// New creates an initialized Facade instance
func New() *Facade {
	return &Facade{
		auditLogger:   audit.NewLogger(),
		hostStore:     host.NewStore(),
		hostkeyStore:  hostkey.NewStore(),
		registryStore: registry.NewStore(),
		poolStore:     pool.NewStore(),
		serviceStore:  service.NewStore(),
		configStore:   serviceconfigfile.NewStore(),
		templateStore: servicetemplate.NewStore(),
		userStore:     user.NewStore(),
		serviceCache:  NewServiceCache(),
		poolCache:     NewPoolCache(),
		hostRegistry:  auth.NewHostExpirationRegistry(),
		deployments:   NewPendingDeploymentMgr(),
		zzk:           getZZK(),
	}
}

// Facade is an entrypoint to available controlplane methods
type Facade struct {
	hostStore     host.Store
	hostkeyStore  hostkey.Store
	registryStore registry.ImageRegistryStore
	poolStore     pool.Store
	templateStore servicetemplate.Store
	serviceStore  service.Store
	configStore   serviceconfigfile.Store
	userStore     user.Store

	auditLogger   audit.Logger
	zzk           ZZK
	dfs           dfs.DFS
	hcache        *health.HealthStatusCache
	metricsClient MetricsClient
	serviceCache  *serviceCache
	poolCache     *poolCache
	hostRegistry  auth.HostExpirationRegistryInterface
	deployments   *PendingDeploymentMgr
	ssm           servicestatemanager.ServiceStateManager
	isvcsPath     string

	rollingRestartTimeout time.Duration
}

func (f *Facade) SetAuditLogger(logger audit.Logger) { f.auditLogger = logger }

func (f *Facade) SetZZK(zzk ZZK) { f.zzk = zzk }

func (f *Facade) SetDFS(dfs dfs.DFS) { f.dfs = dfs }

func (f *Facade) SetServiceStateManager(ssm servicestatemanager.ServiceStateManager) { f.ssm = ssm }

func (f *Facade) SetHostStore(store host.Store) {
	f.hostStore = store
	f.poolCache.SetDirty()
}

func (f *Facade) SetHostkeyStore(store hostkey.Store) { f.hostkeyStore = store }

func (f *Facade) SetRegistryStore(store registry.ImageRegistryStore) { f.registryStore = store }

func (f *Facade) SetPoolStore(store pool.Store) {
	f.poolStore = store
	f.poolCache.SetDirty()
}

func (f *Facade) SetServiceStore(store service.Store) {
	f.serviceStore = store
	f.poolCache.SetDirty()
}

func (f *Facade) SetConfigStore(store serviceconfigfile.Store) { f.configStore = store }

func (f *Facade) SetUserStore(store user.Store) { f.userStore = store }

func (f *Facade) SetTemplateStore(store servicetemplate.Store) { f.templateStore = store }

func (f *Facade) SetHealthCache(hcache *health.HealthStatusCache) { f.hcache = hcache }

func (f *Facade) SetMetricsClient(client MetricsClient) { f.metricsClient = client }

func (f *Facade) SetIsvcsPath(path string) { f.isvcsPath = path }

func (f *Facade) SetHostExpirationRegistry(hostRegistry auth.HostExpirationRegistryInterface) {
	f.hostRegistry = hostRegistry
}

func (f *Facade) SetDeploymentMgr(mgr *PendingDeploymentMgr) { f.deployments = mgr }

func (f *Facade) SetRollingRestartTimeout(t time.Duration) { f.rollingRestartTimeout = t }
