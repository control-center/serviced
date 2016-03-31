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

import "github.com/zenoss/go-json-rest"

//getRoutes returns all registered rest routes
func (sc *ServiceConfig) getRoutes() []rest.Route {

	gz := gzipHandler

	routes := []rest.Route{
		rest.Route{"GET", "/", gz(mainPage)},

		// Backups
		rest.Route{"GET", "/backup/create", gz(sc.authorizedClient(RestBackupCreate))},
		rest.Route{"GET", "/backup/restore", gz(sc.authorizedClient(RestBackupRestore))},
		rest.Route{"GET", "/backup/list", gz(sc.authorizedClient(RestBackupFileList))},
		rest.Route{"GET", "/backup/status", gz(sc.authorizedClient(RestBackupStatus))},
		rest.Route{"GET", "/backup/restore/status", gz(sc.authorizedClient(RestRestoreStatus))},

		// Hosts
		rest.Route{"GET", "/hosts", gz(sc.checkAuth(restGetHosts))},
		rest.Route{"GET", "/hosts/running", gz(sc.checkAuth(restGetActiveHostIDs))},
		rest.Route{"GET", "/hosts/defaultHostAlias", gz(sc.checkAuth(restGetDefaultHostAlias))},
		rest.Route{"GET", "/hosts/:hostId", gz(sc.checkAuth(restGetHost))},
		rest.Route{"POST", "/hosts/add", gz(sc.checkAuth(restAddHost))},
		rest.Route{"DELETE", "/hosts/:hostId", gz(sc.checkAuth(restRemoveHost))},
		rest.Route{"PUT", "/hosts/:hostId", gz(sc.checkAuth(restUpdateHost))},
		rest.Route{"GET", "/hosts/:hostId/running", gz(sc.authorizedClient(restGetRunningForHost))},
		rest.Route{"DELETE", "/hosts/:hostId/:serviceStateId", gz(sc.authorizedClient(restKillRunning))},

		// Pools
		rest.Route{"GET", "/pools/:poolId", gz(sc.checkAuth(restGetPool))},
		rest.Route{"DELETE", "/pools/:poolId", gz(sc.checkAuth(restRemovePool))},
		rest.Route{"PUT", "/pools/:poolId", gz(sc.checkAuth(restUpdatePool))},
		rest.Route{"POST", "/pools/add", gz(sc.checkAuth(restAddPool))},
		rest.Route{"GET", "/pools", gz(sc.checkAuth(restGetPools))},
		rest.Route{"GET", "/pools/:poolId/hosts", gz(sc.checkAuth(restGetHostsForResourcePool))},

		// Pools (VirtualIP)
		rest.Route{"PUT", "/pools/:poolId/virtualip", gz(sc.checkAuth(restAddPoolVirtualIP))},
		rest.Route{"DELETE", "/pools/:poolId/virtualip/*ip", gz(sc.checkAuth(restRemovePoolVirtualIP))},

		// Pools (IPs)
		rest.Route{"GET", "/pools/:poolId/ips", gz(sc.checkAuth(restGetPoolIps))},

		// Services (Apps)
		rest.Route{"GET", "/services", gz(sc.authorizedClient(restGetAllServices))},
		rest.Route{"GET", "/servicehealth", gz(sc.authorizedClient(restGetServicesHealth))},
		rest.Route{"GET", "/services/:serviceId", gz(sc.authorizedClient(restGetService))},
		rest.Route{"GET", "/services/:serviceId/running", gz(sc.authorizedClient(restGetRunningForService))},
		rest.Route{"GET", "/services/:serviceId/status", gz(sc.authorizedClient(restGetStatusForService))},
		rest.Route{"GET", "/services/:serviceId/running/:serviceStateId", gz(sc.authorizedClient(restGetRunningService))},
		rest.Route{"GET", "/services/:serviceId/:serviceStateId/logs", gz(sc.authorizedClient(restGetServiceStateLogs))},
		rest.Route{"GET", "/services/:serviceId/:serviceStateId/logs/download", gz(sc.authorizedClient(downloadServiceStateLogs))},
		rest.Route{"POST", "/services/add", gz(sc.authorizedClient(restAddService))},
		rest.Route{"POST", "/services/deploy", gz(sc.authorizedClient(restDeployService))},
		rest.Route{"DELETE", "/services/:serviceId", gz(sc.authorizedClient(restRemoveService))},
		rest.Route{"GET", "/services/:serviceId/logs", gz(sc.authorizedClient(restGetServiceLogs))},
		rest.Route{"PUT", "/services/:serviceId", gz(sc.authorizedClient(restUpdateService))},
		rest.Route{"GET", "/services/:serviceId/snapshot", gz(sc.authorizedClient(restSnapshotService))},
		rest.Route{"PUT", "/services/:serviceId/restartService", gz(sc.authorizedClient(restRestartService))},
		rest.Route{"PUT", "/services/:serviceId/startService", gz(sc.authorizedClient(restStartService))},
		rest.Route{"PUT", "/services/:serviceId/stopService", gz(sc.authorizedClient(restStopService))},
		rest.Route{"POST", "/services/:serviceId/migrate", sc.authorizedClient(restPostServicesForMigration)},

		// Services (Virtual Host)
		rest.Route{"GET", "/services/vhosts", gz(sc.authorizedClient(restGetVirtualHosts))},
		rest.Route{"PUT", "/services/:serviceId/endpoint/:application/vhosts/*name", gz(sc.authorizedClient(restAddVirtualHost))},
		rest.Route{"DELETE", "/services/:serviceId/endpoint/:application/vhosts/*name", gz(sc.authorizedClient(restRemoveVirtualHost))},
		rest.Route{"POST", "/services/:serviceId/endpoint/:application/vhosts/*name", gz(sc.authorizedClient(restVirtualHostEnable))},

		// Services (Endpoint Ports)
		rest.Route{"PUT", "/services/:serviceId/endpoint/:application/ports/*portname", gz(sc.authorizedClient(restAddPort))},
		rest.Route{"DELETE", "/services/:serviceId/endpoint/:application/ports/*portname", gz(sc.authorizedClient(restRemovePort))},
		rest.Route{"POST", "/services/:serviceId/endpoint/:application/ports/*portname", gz(sc.authorizedClient(restPortEnable))},

		// Services (IP)
		rest.Route{"PUT", "/services/:serviceId/ip", gz(sc.authorizedClient(restServiceAutomaticAssignIP))},
		rest.Route{"PUT", "/services/:serviceId/ip/*ip", gz(sc.authorizedClient(restServiceManualAssignIP))},

		// Service templates (App templates)
		rest.Route{"GET", "/templates", gz(sc.checkAuth(restGetAppTemplates))},
		rest.Route{"POST", "/templates/add", gz(sc.checkAuth(restAddAppTemplate))},
		rest.Route{"DELETE", "/templates/:templateId", gz(sc.checkAuth(restRemoveAppTemplate))},
		rest.Route{"POST", "/templates/deploy", gz(sc.checkAuth(restDeployAppTemplate))},
		rest.Route{"POST", "/templates/deploy/status", gz(sc.checkAuth(restDeployAppTemplateStatus))},
		rest.Route{"GET", "/templates/deploy/active", gz(sc.checkAuth(restDeployAppTemplateActive))},

		// Login
		rest.Route{"POST", "/login", gz(sc.unAuthorizedClient(restLogin))},
		rest.Route{"DELETE", "/login", gz(restLogout)},

		// "Misc" stuff
		rest.Route{"GET", "/top/services", gz(sc.authorizedClient(restGetTopServices))},
		rest.Route{"GET", "/running", gz(sc.authorizedClient(restGetAllRunning))},
		rest.Route{"GET", "/config", gz(sc.authorizedClient(restGetUIConfig))},
		rest.Route{"GET", "/servicestatus", gz(sc.authorizedClient(restGetConciseServiceStatus))},

		// Generic static data
		rest.Route{"GET", "/favicon.ico", gz(favIcon)},
		rest.Route{"GET", "/static/logview/*resource", gz(sc.checkAuth(getProtectedLogViewData))},
		rest.Route{"GET", "/static/*resource", gz(staticData)},
		rest.Route{"GET", "/licenses.html", gz(licenses)},

		// Info about serviced itself
		rest.Route{"GET", "/dockerIsLoggedIn", gz(sc.authorizedClient(restDockerIsLoggedIn))},
		rest.Route{"GET", "/stats", gz(sc.isCollectingStats())},
		rest.Route{"GET", "/version", gz(sc.authorizedClient(restGetServicedVersion))},
		rest.Route{"GET", "/storage", gz(sc.authorizedClient(restGetStorage))},
	}

	// Hardcoding these target URLs for now.
	// TODO: When internal services are allowed to run on other hosts, look that up.
	// All API calls require authentication
	routes = routeToInternalServiceProxy("/api/controlplane/elastic", "http://127.0.0.1:9100/", true, routes)
	routes = routeToInternalServiceProxy("/metrics/api", "http://127.0.0.1:8888/api", true, routes)

	// Allow static assets for metrics data to be loaded without authentication since they are
	// included in index.html by default.
	routes = routeToInternalServiceProxy("/metrics/static", "http://127.0.0.1:8888/static", false, routes)

	return routes
}
