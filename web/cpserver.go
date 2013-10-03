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
		// Pools
		rest.Route{"POST", "/pools/add", AuthorizedClient(RestAddPool)},
		rest.Route{"GET", "/pools/:poolId/hosts", AuthorizedClient(RestGetHostsForResourcePool)},
		rest.Route{"DELETE", "/pools/:poolId", AuthorizedClient(RestRemovePool)},
		rest.Route{"POST", "/pools/:poolId", AuthorizedClient(RestUpdatePool)},
		rest.Route{"GET", "/pools", AuthorizedClient(RestGetPools)},
		// Services (Apps)
		rest.Route{"GET", "/services/all", AuthorizedClient(RestGetAllServices)},
		rest.Route{"POST", "/services/add", AuthorizedClient(RestAddService)},
		rest.Route{"GET", "/services/top", AuthorizedClient(RestGetTopServices)},
		rest.Route{"DELETE", "/service/:serviceId", AuthorizedClient(RestRemoveService)},
		rest.Route{"POST", "/service/:serviceId", AuthorizedClient(RestUpdateService)},
		// Service templates (App templates)
		rest.Route{"GET", "/templates", AuthorizedClient(RestGetAppTemplates)},
		// Login
		rest.Route{"POST", "/login", RestLogin},
		rest.Route{"DELETE", "/login", RestLogout},
		// Generic static data
		rest.Route{"GET", "/favicon.ico", FavIcon},
		rest.Route{"GET", "/static*resource", StaticData},
	)
	http.ListenAndServe(":8787", &handler)
}

