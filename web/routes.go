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

package web

import (
	"github.com/control-center/serviced/health"
	"github.com/zenoss/go-json-rest"
)

//getRoutes returns all registered rest routes
func (sc *ServiceConfig) getRoutes() []rest.Route {

	routes := []rest.Route{
		rest.Route{"GET", "/", mainPage},
		rest.Route{"GET", "/test", testPage},
		rest.Route{"GET", "/stats", sc.isCollectingStats()},
		rest.Route{"GET", "/version", sc.authorizedClient(restGetServicedVersion)},
		rest.Route{"GET", "/backup/create", sc.authorizedClient(RestBackupCreate)},
		rest.Route{"GET", "/backup/restore", sc.authorizedClient(RestBackupRestore)},
		rest.Route{"GET", "/backup/list", sc.checkAuth(RestBackupFileList)},
		rest.Route{"GET", "/backup/status", sc.authorizedClient(RestBackupStatus)},
		rest.Route{"GET", "/backup/restore/status", sc.authorizedClient(RestRestoreStatus)},
		// Hosts
		rest.Route{"GET", "/hosts", sc.checkAuth(restGetHosts)},
		rest.Route{"GET", "/hosts/running", sc.checkAuth(restGetActiveHostIDs)},
		rest.Route{"GET", "/hosts/defaultHostAlias", sc.checkAuth(restGetDefaultHostAlias)},
		rest.Route{"GET", "/hosts/:hostId", sc.checkAuth(restGetHost)},
		rest.Route{"POST", "/hosts/add", sc.checkAuth(restAddHost)},
		rest.Route{"DELETE", "/hosts/:hostId", sc.checkAuth(restRemoveHost)},
		rest.Route{"PUT", "/hosts/:hostId", sc.checkAuth(restUpdateHost)},
		rest.Route{"GET", "/hosts/:hostId/running", sc.authorizedClient(restGetRunningForHost)},
		rest.Route{"DELETE", "/hosts/:hostId/:serviceStateId", sc.authorizedClient(restKillRunning)},

		// Mux
		rest.Route{"GET", "/mux/connections", sc.checkAuth(restGetMuxConnections)},

		// Pools
		rest.Route{"GET", "/pools/:poolId", sc.checkAuth(restGetPool)},
		rest.Route{"DELETE", "/pools/:poolId", sc.checkAuth(restRemovePool)},
		rest.Route{"PUT", "/pools/:poolId", sc.checkAuth(restUpdatePool)},
		rest.Route{"POST", "/pools/add", sc.checkAuth(restAddPool)},
		rest.Route{"GET", "/pools", sc.checkAuth(restGetPools)},
		rest.Route{"GET", "/pools/:poolId/hosts", sc.checkAuth(restGetHostsForResourcePool)},

		// Pools (VirtualIP)
		rest.Route{"PUT", "/pools/:poolId/virtualip", sc.checkAuth(restAddPoolVirtualIP)},
		rest.Route{"DELETE", "/pools/:poolId/virtualip/*ip", sc.checkAuth(restRemovePoolVirtualIP)},

		// Pools (IPs)
		rest.Route{"GET", "/pools/:poolId/ips", sc.checkAuth(restGetPoolIps)},

		// Services (Apps)
		rest.Route{"GET", "/services", sc.authorizedClient(restGetAllServices)},
		rest.Route{"GET", "/servicehealth", sc.authorizedClient(health.RestGetHealthStatus)},
		rest.Route{"GET", "/services/:serviceId", sc.authorizedClient(restGetService)},
		rest.Route{"GET", "/services/:serviceId/running", sc.authorizedClient(restGetRunningForService)},
		rest.Route{"GET", "/services/:serviceId/status", sc.authorizedClient(restGetStatusForService)},
		rest.Route{"GET", "/services/:serviceId/running/:serviceStateId", sc.authorizedClient(restGetRunningService)},
		rest.Route{"GET", "/services/:serviceId/:serviceStateId/logs", sc.authorizedClient(restGetServiceStateLogs)},
		rest.Route{"POST", "/services/add", sc.authorizedClient(restAddService)},
		rest.Route{"POST", "/services/deploy", sc.authorizedClient(restDeployService)},
		rest.Route{"DELETE", "/services/:serviceId", sc.authorizedClient(restRemoveService)},
		rest.Route{"GET", "/services/:serviceId/logs", sc.authorizedClient(restGetServiceLogs)},
		rest.Route{"PUT", "/services/:serviceId", sc.authorizedClient(restUpdateService)},
		rest.Route{"GET", "/services/:serviceId/snapshot", sc.authorizedClient(restSnapshotService)},
		rest.Route{"PUT", "/services/:serviceId/restartService", sc.authorizedClient(restRestartService)},
		rest.Route{"PUT", "/services/:serviceId/startService", sc.authorizedClient(restStartService)},
		rest.Route{"PUT", "/services/:serviceId/stopService", sc.authorizedClient(restStopService)},

		// Services (Virtual Host)
		rest.Route{"GET", "/services/vhosts", sc.authorizedClient(restGetVirtualHosts)},
		rest.Route{"PUT", "/services/:serviceId/endpoint/:application/vhosts/*name", sc.authorizedClient(restAddVirtualHost)},
		rest.Route{"DELETE", "/services/:serviceId/endpoint/:application/vhosts/*name", sc.authorizedClient(restRemoveVirtualHost)},

		// Services (IP)
		rest.Route{"PUT", "/services/:serviceId/ip", sc.authorizedClient(restServiceAutomaticAssignIP)},
		rest.Route{"PUT", "/services/:serviceId/ip/*ip", sc.authorizedClient(restServiceManualAssignIP)},

		// Service templates (App templates)
		rest.Route{"GET", "/templates", sc.authorizedClient(restGetAppTemplates)},
		rest.Route{"POST", "/templates/add", sc.authorizedClient(restAddAppTemplate)},
		rest.Route{"DELETE", "/templates/:templateId", sc.authorizedClient(restRemoveAppTemplate)},
		rest.Route{"POST", "/templates/deploy", sc.authorizedClient(restDeployAppTemplate)},
		rest.Route{"POST", "/templates/deploy/status", sc.authorizedClient(restDeployAppTemplateStatus)},
		rest.Route{"GET", "/templates/deploy/active", sc.authorizedClient(restDeployAppTemplateActive)},

		// Login
		rest.Route{"POST", "/login", sc.unAuthorizedClient(restLogin)},
		rest.Route{"DELETE", "/login", restLogout},

		// DockerLogin
		rest.Route{"GET", "/dockerIsLoggedIn", sc.authorizedClient(restDockerIsLoggedIn)},

		// "Misc" stuff
		rest.Route{"GET", "/top/services", sc.authorizedClient(restGetTopServices)},
		rest.Route{"GET", "/running", sc.authorizedClient(restGetAllRunning)},

		// Generic static data
		rest.Route{"GET", "/favicon.ico", favIcon},
		rest.Route{"GET", "/static*resource", staticData},
	}

	// Hardcoding these target URLs for now.
	// TODO: When internal services are allowed to run on other hosts, look that up.
	routes = routeToInternalServiceProxy("/api/controlplane/elastic", "http://127.0.0.1:9100/", routes)
	routes = routeToInternalServiceProxy("/metrics", "http://127.0.0.1:8888/", routes)

	return routes
}
