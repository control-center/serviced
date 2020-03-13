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
	"github.com/control-center/serviced/domain/addressassignment"
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/domain/hostkey"
	"github.com/control-center/serviced/domain/logfilter"
	"github.com/control-center/serviced/domain/pool"
	"github.com/control-center/serviced/domain/properties"
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

// Facade is an entrypoint to available controlplane methods
type Facade struct {
	addressassignmentStore addressassignment.Store
	configfileStore        serviceconfigfile.Store
	hostStore              host.Store
	hostkeyStore           hostkey.Store
	logfilterStore         logfilter.Store
	poolStore              pool.Store
	propertyStore          properties.Store
	registryStore          registry.Store
	serviceStore           service.Store
	templateStore          servicetemplate.Store
	userStore              user.Store

	auditLogger   audit.Logger
	zzk           ZZK
	dfs           dfs.DFS
	hcache        *health.HealthStatusCache
	metricsClient metrics.Client
	serviceCache  *serviceCache
	poolCache     PoolCache
	hostRegistry  auth.HostExpirationRegistryInterface
	deployments   *PendingDeploymentMgr
	ssm           servicestatemanager.ServiceStateManager
	isvcsPath     string

	rollingRestartTimeout time.Duration
}

// instantiate the package logger
var plog = logging.PackageLogger()

// assert Facade implements the API (interface)
var _ API = &Facade{}

// New creates new Facade instance
func New() *Facade {
	return &Facade{
		addressassignmentStore: addressassignment.NewStore(),
		configfileStore:        serviceconfigfile.NewStore(),
		hostStore:              host.NewStore(),
		hostkeyStore:           hostkey.NewStore(),
		logfilterStore:         logfilter.NewStore(),
		poolStore:              pool.NewStore(),
		propertyStore:          properties.NewStore(),
		registryStore:          registry.NewStore(),
		serviceStore:           service.NewStore(),
		templateStore:          servicetemplate.NewStore(),
		userStore:              user.NewStore(),

		auditLogger:  audit.NewLogger(),
		serviceCache: NewServiceCache(),
		poolCache:    NewPoolCache(),
		hostRegistry: auth.NewHostExpirationRegistry(),
		deployments:  NewPendingDeploymentMgr(),
		zzk:          getZZK(),
	}
}

// SetAddressAssignmentStore sets a addressassignment.Store object on the facade.
func (f *Facade) SetAddressAssignmentStore(store addressassignment.Store) {
	f.addressassignmentStore = store
}

// SetHostStore sets a host.Store object on the facade.
func (f *Facade) SetHostStore(store host.Store) { f.hostStore = store }

// SetHostKeyStore sets a hostkey.Store object on the facade.
func (f *Facade) SetHostKeyStore(store hostkey.Store) { f.hostkeyStore = store }

// SetLogFilterStore sets a logfilter.Store object on the facade.
func (f *Facade) SetLogFilterStore(store logfilter.Store) { f.logfilterStore = store }

// SetPoolStore sets a pool.Store object on the facade.
func (f *Facade) SetPoolStore(store pool.Store) {
	f.poolStore = store
	f.poolCache.SetDirty()
}

// SetPropertyStore sets a properties.Store object on the facade.
func (f *Facade) SetPropertyStore(store properties.Store) { f.propertyStore = store }

// SetRegistryStore sets a registry.Store object on the facade.
func (f *Facade) SetRegistryStore(store registry.Store) { f.registryStore = store }

// SetServiceStore sets a service.Store object on the facade.
func (f *Facade) SetServiceStore(store service.Store) {
	f.serviceStore = store
	f.poolCache.SetDirty()
}

// SetServiceConfigFileStore sets a serviceconfigfile.Store object on the facade.
func (f *Facade) SetServiceConfigFileStore(store serviceconfigfile.Store) { f.configfileStore = store }

// SetServiceTemplateStore sets a servicetemplate.Store object on the facade.
func (f *Facade) SetServiceTemplateStore(store servicetemplate.Store) { f.templateStore = store }

// SetUserStore sets a user.Store object on the facade.
func (f *Facade) SetUserStore(store user.Store) { f.userStore = store }

// SetAuditLogger sets AuditLogger
func (f *Facade) SetAuditLogger(logger audit.Logger) { f.auditLogger = logger }

// SetZZK sets ZZK
func (f *Facade) SetZZK(zzk ZZK) { f.zzk = zzk }

// SetDFS sets DFS
func (f *Facade) SetDFS(dfs dfs.DFS) { f.dfs = dfs }

// SetServiceStateManager sets ServiceStateManager
func (f *Facade) SetServiceStateManager(ssm servicestatemanager.ServiceStateManager) { f.ssm = ssm }

// SetHealthCache sets HealthCache
func (f *Facade) SetHealthCache(hcache *health.HealthStatusCache) { f.hcache = hcache }

// SetMetricsClient sets MetricsClient
func (f *Facade) SetMetricsClient(client metrics.Client) { f.metricsClient = client }

// SetIsvcsPath sets ISVCS path
func (f *Facade) SetIsvcsPath(path string) { f.isvcsPath = path }

// SetHostExpirationRegistry sets HostExpirationRegistry
func (f *Facade) SetHostExpirationRegistry(hostRegistry auth.HostExpirationRegistryInterface) {
	f.hostRegistry = hostRegistry
}

// SetDeploymentMgr sets the PendingDeploymentMgr
func (f *Facade) SetDeploymentMgr(mgr *PendingDeploymentMgr) { f.deployments = mgr }

// SetRollingRestartTimeout sets the duration between restarts of instances.
func (f *Facade) SetRollingRestartTimeout(t time.Duration) { f.rollingRestartTimeout = t }
