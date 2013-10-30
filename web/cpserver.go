package web

import (
	"github.com/ant0ine/go-json-rest"
	"net/http"
)

func Serve() {

	handler := rest.ResourceHandler{
		EnableRelaxedContentType: true,
        }
	handler.SetRoutes(
		rest.Route{"GET", "/", MainPage},
		rest.Route{"GET", "/test", TestPage},
		// Hosts
		rest.Route{"GET", "/hosts", AuthorizedClient(RestGetHosts)},
		rest.Route{"POST", "/hosts/add", AuthorizedClient(RestAddHost)},
		rest.Route{"DELETE", "/hosts/:hostId", AuthorizedClient(RestRemoveHost)},
		rest.Route{"PUT", "/hosts/:hostId", AuthorizedClient(RestUpdateHost)},
		rest.Route{"GET", "/hosts/:hostId/running", AuthorizedClient(RestGetRunningForHost)},
		// Pools
		rest.Route{"POST", "/pools/add", AuthorizedClient(RestAddPool)},
		rest.Route{"GET", "/pools/:poolId/hosts", AuthorizedClient(RestGetHostsForResourcePool)},
		rest.Route{"DELETE", "/pools/:poolId", AuthorizedClient(RestRemovePool)},
		rest.Route{"PUT", "/pools/:poolId", AuthorizedClient(RestUpdatePool)},
		rest.Route{"GET", "/pools", AuthorizedClient(RestGetPools)},
		// Services (Apps)
		rest.Route{"GET", "/services", AuthorizedClient(RestGetAllServices)},
		rest.Route{"GET", "/services/:serviceId/running", AuthorizedClient(RestGetRunningForService)},
		rest.Route{"POST", "/services/add", AuthorizedClient(RestAddService)},
		rest.Route{"DELETE", "/services/:serviceId", AuthorizedClient(RestRemoveService)},
		rest.Route{"GET", "/services/:serviceId/logs", AuthorizedClient(RestGetServiceLogs)},
		rest.Route{"PUT", "/services/:serviceId", AuthorizedClient(RestUpdateService)},
		// Service templates (App templates)
		rest.Route{"GET", "/templates", AuthorizedClient(RestGetAppTemplates)},
		rest.Route{"POST", "/templates/deploy", AuthorizedClient(RestDeployAppTemplate)},
		// Login
		rest.Route{"POST", "/login", RestLogin},
		rest.Route{"DELETE", "/login", RestLogout},
		// "Misc" stuff
		rest.Route{"GET", "/top/services", AuthorizedClient(RestGetTopServices)},
		rest.Route{"GET", "/running/:serviceStateId/logs", AuthorizedClient(RestGetServiceStateLogs)},
		rest.Route{"GET", "/running", AuthorizedClient(RestGetAllRunning)},
		rest.Route{"DELETE", "/running/:serviceStateId", AuthorizedClient(RestKillRunning)},
		// Generic static data
		rest.Route{"GET", "/favicon.ico", FavIcon},
		rest.Route{"GET", "/static*resource", StaticData},
	)
	http.ListenAndServe(":8787", &handler)
}

