// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package web

import (
	"github.com/zenoss/go-json-rest"
)

//getRoutes returns all registered rest routes
func (sc *ServiceConfig) getRoutes() []rest.Route {

	routes := []rest.Route{
		rest.Route{"GET", "/", MainPage},
		rest.Route{"GET", "/test", TestPage},
		rest.Route{"GET", "/stats", sc.IsCollectingStats()},
		//Hosts
		rest.Route{"GET", "/hosts", sc.CheckAuth(RestGetHosts)},
		rest.Route{"POST", "/hosts/add", sc.CheckAuth(RestAddHost)},
		rest.Route{"DELETE", "/hosts/:hostId", sc.CheckAuth(RestRemoveHost)},
		rest.Route{"PUT", "/hosts/:hostId", sc.CheckAuth(RestUpdateHost)},
		rest.Route{"GET", "/hosts/:hostId/running", sc.AuthorizedClient(RestGetRunningForHost)},
		rest.Route{"DELETE", "/hosts/:hostId/:serviceStateId", sc.AuthorizedClient(RestKillRunning)},
		//Pools
		rest.Route{"POST", "/pools/add", sc.CheckAuth(RestAddPool)},
		rest.Route{"GET", "/pools/:poolId/hosts", sc.CheckAuth(RestGetHostsForResourcePool)},
		rest.Route{"DELETE", "/pools/:poolId", sc.CheckAuth(RestRemovePool)},
		rest.Route{"PUT", "/pools/:poolId", sc.CheckAuth(RestUpdatePool)},
		rest.Route{"GET", "/pools", sc.CheckAuth(RestGetPools)},
		// Services (Apps)
		rest.Route{"GET", "/services", sc.AuthorizedClient(RestGetAllServices)},
		rest.Route{"GET", "/services/:serviceId", sc.AuthorizedClient(RestGetService)},
		rest.Route{"GET", "/services/:serviceId/running", sc.AuthorizedClient(RestGetRunningForService)},
		rest.Route{"GET", "/services/:serviceId/running/:serviceStateId", sc.AuthorizedClient(RestGetRunningService)},
		rest.Route{"GET", "/services/:serviceId/:serviceStateId/logs", sc.AuthorizedClient(RestGetServiceStateLogs)},
		rest.Route{"POST", "/services/add", sc.AuthorizedClient(RestAddService)},
		rest.Route{"DELETE", "/services/:serviceId", sc.AuthorizedClient(RestRemoveService)},
		rest.Route{"GET", "/services/:serviceId/logs", sc.AuthorizedClient(RestGetServiceLogs)},
		rest.Route{"PUT", "/services/:serviceId", sc.AuthorizedClient(RestUpdateService)},
		rest.Route{"GET", "/services/:serviceId/snapshot", sc.AuthorizedClient(RestSnapshotService)},
		rest.Route{"PUT", "/services/:serviceId/startService", sc.AuthorizedClient(RestStartService)},
		rest.Route{"PUT", "/services/:serviceId/stopService", sc.AuthorizedClient(RestStopService)},

		// Services (Virtual Host)
		rest.Route{"GET", "/vhosts", sc.AuthorizedClient(RestGetVirtualHosts)},
		rest.Route{"POST", "/vhosts/:serviceId/:application/:vhostName", sc.AuthorizedClient(RestAddVirtualHost)},
		rest.Route{"DELETE", "/vhosts/:serviceId/:application/:vhostName", sc.AuthorizedClient(RestRemoveVirtualHost)},

		// Service templates (App templates)
		rest.Route{"GET", "/templates", sc.AuthorizedClient(RestGetAppTemplates)},
		rest.Route{"POST", "/templates/deploy", sc.AuthorizedClient(RestDeployAppTemplate)},

		// Login
		rest.Route{"POST", "/login", sc.UnAuthorizedClient(RestLogin)},
		rest.Route{"DELETE", "/login", RestLogout},
		// "Misc" stuff
		rest.Route{"GET", "/top/services", sc.AuthorizedClient(RestGetTopServices)},
		rest.Route{"GET", "/running", sc.AuthorizedClient(RestGetAllRunning)},
		// Generic static data
		rest.Route{"GET", "/favicon.ico", FavIcon},
		rest.Route{"GET", "/static*resource", StaticData},
	}

	// Hardcoding these target URLs for now.
	// TODO: When internal services are allowed to run on other hosts, look that up.
	routes = routeToInternalServiceProxy("/elastic", "http://127.0.0.1:9200/", routes)
	routes = routeToInternalServiceProxy("/metrics", "http://127.0.0.1:8888/", routes)

	return routes
}
