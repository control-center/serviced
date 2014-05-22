// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package web

import (
	"github.com/zenoss/go-json-rest"
	"github.com/zenoss/serviced/health"
)

//getRoutes returns all registered rest routes
func (sc *ServiceConfig) getRoutes() []rest.Route {

	routes := []rest.Route{
		rest.Route{"GET", "/", mainPage},
		rest.Route{"GET", "/test", testPage},
		rest.Route{"GET", "/stats", sc.isCollectingStats()},
		rest.Route{"GET", "/version", sc.authorizedClient(restGetServicedVersion)},

		// Hosts
		rest.Route{"GET", "/hosts", sc.checkAuth(restGetHosts)},
		rest.Route{"GET", "/hosts/:hostId", sc.checkAuth(restGetHost)},
		rest.Route{"POST", "/hosts/add", sc.checkAuth(restAddHost)},
		rest.Route{"DELETE", "/hosts/:hostId", sc.checkAuth(restRemoveHost)},
		rest.Route{"PUT", "/hosts/:hostId", sc.checkAuth(restUpdateHost)},
		rest.Route{"GET", "/hosts/:hostId/running", sc.authorizedClient(restGetRunningForHost)},
		rest.Route{"DELETE", "/hosts/:hostId/:serviceStateId", sc.authorizedClient(restKillRunning)},

		// Pools
		rest.Route{"GET", "/pools/:poolId", sc.checkAuth(restGetPool)},
		rest.Route{"DELETE", "/pools/:poolId", sc.checkAuth(restRemovePool)},
		rest.Route{"PUT", "/pools/:poolId", sc.checkAuth(restUpdatePool)},
		rest.Route{"POST", "/pools/add", sc.checkAuth(restAddPool)},
		rest.Route{"GET", "/pools", sc.checkAuth(restGetPools)},
		rest.Route{"GET", "/pools/:poolId/hosts", sc.checkAuth(restGetHostsForResourcePool)},

		// Pools (VirtualIP)
		rest.Route{"PUT", "/pools/:poolId/virtualip", sc.authorizedClient(restAddPoolVirtualIP)},
		rest.Route{"DELETE", "/pools/:poolId/virtualip/*id", sc.authorizedClient(restRemovePoolVirtualIP)},

		// Pools (IPs)
		rest.Route{"GET", "/pools/:poolId/ips", sc.checkAuth(restGetPoolIps)},

		// Services (Apps)
		rest.Route{"GET", "/services", sc.authorizedClient(restGetAllServices)},
		rest.Route{"GET", "/servicehealth", sc.authorizedClient(health.RestGetHealthStatus)},
		rest.Route{"GET", "/services/:serviceId", sc.authorizedClient(restGetService)},
		rest.Route{"GET", "/services/:serviceId/running", sc.authorizedClient(restGetRunningForService)},
		rest.Route{"GET", "/services/:serviceId/running/:serviceStateId", sc.authorizedClient(restGetRunningService)},
		rest.Route{"GET", "/services/:serviceId/:serviceStateId/logs", sc.authorizedClient(restGetServiceStateLogs)},
		rest.Route{"POST", "/services/add", sc.authorizedClient(restAddService)},
		rest.Route{"DELETE", "/services/:serviceId", sc.authorizedClient(restRemoveService)},
		rest.Route{"GET", "/services/:serviceId/logs", sc.authorizedClient(restGetServiceLogs)},
		rest.Route{"PUT", "/services/:serviceId", sc.authorizedClient(restUpdateService)},
		rest.Route{"GET", "/services/:serviceId/snapshot", sc.authorizedClient(restSnapshotService)},
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
		rest.Route{"POST", "/templates/deploy", sc.authorizedClient(restDeployAppTemplate)},

		// Login
		rest.Route{"POST", "/login", sc.unAuthorizedClient(restLogin)},
		rest.Route{"DELETE", "/login", restLogout},

		// "Misc" stuff
		rest.Route{"GET", "/top/services", sc.authorizedClient(restGetTopServices)},
		rest.Route{"GET", "/running", sc.authorizedClient(restGetAllRunning)},

		// Generic static data
		rest.Route{"GET", "/favicon.ico", favIcon},
		rest.Route{"GET", "/static*resource", staticData},
	}

	// Hardcoding these target URLs for now.
	// TODO: When internal services are allowed to run on other hosts, look that up.
	routes = routeToInternalServiceProxy("/api/controlplane/elastic", "http://127.0.0.1:9200/", routes)
	routes = routeToInternalServiceProxy("/metrics", "http://127.0.0.1:8888/", routes)

	return routes
}
