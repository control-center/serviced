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
		rest.Route{"POST", "/hosts/:hostId", AuthorizedClient(RestUpdateHost)},
		rest.Route{"GET", "/hosts/:hostId/running", AuthorizedClient(RestGetRunning)},
		// Pools
		rest.Route{"POST", "/pools/add", AuthorizedClient(RestAddPool)},
		rest.Route{"GET", "/pools/:poolId/hosts", AuthorizedClient(RestGetHostsForResourcePool)},
		rest.Route{"DELETE", "/pools/:poolId", AuthorizedClient(RestRemovePool)},
		rest.Route{"POST", "/pools/:poolId", AuthorizedClient(RestUpdatePool)},
		rest.Route{"GET", "/pools", AuthorizedClient(RestGetPools)},
		// Services (Apps)
		rest.Route{"GET", "/services", AuthorizedClient(RestGetAllServices)},
		rest.Route{"POST", "/services/add", AuthorizedClient(RestAddService)},
		rest.Route{"DELETE", "/services/:serviceId", AuthorizedClient(RestRemoveService)},
		rest.Route{"POST", "/services/:serviceId", AuthorizedClient(RestUpdateService)},
		// Service templates (App templates)
		rest.Route{"GET", "/templates", AuthorizedClient(RestGetAppTemplates)},
		// Login
		rest.Route{"POST", "/login", RestLogin},
		rest.Route{"DELETE", "/login", RestLogout},
		// "Top" stuff
		rest.Route{"GET", "/top/services", AuthorizedClient(RestGetTopServices)},
		// Generic static data
		rest.Route{"GET", "/favicon.ico", FavIcon},
		rest.Route{"GET", "/static*resource", StaticData},
	)
	http.ListenAndServe(":8787", &handler)
}

