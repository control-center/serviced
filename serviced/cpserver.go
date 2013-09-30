package main

import (
	"github.com/ant0ine/go-json-rest"
	"github.com/zenoss/serviced/web"
	"net/http"
)

func websvc() {

	handler := rest.ResourceHandler{
		EnableRelaxedContentType: true,
        }
	handler.SetRoutes(
		rest.Route{"GET", "/", web.MainPage},
		rest.Route{"GET", "/test", web.TestPage},
		rest.Route{"GET", "/hosts", web.AuthorizedClient(web.RestGetHosts)},
		rest.Route{"POST", "/hosts/add", web.AuthorizedClient(web.RestAddHost)},
		rest.Route{"POST", "/pools/add", web.AuthorizedClient(web.RestAddPool)},
		rest.Route{"POST", "/services/add", web.AuthorizedClient(web.RestAddService)},
		rest.Route{"GET", "/pools/:poolId/hosts", web.AuthorizedClient(web.RestGetHostsForResourcePool)},
		rest.Route{"DELETE", "/pools/:poolId", web.AuthorizedClient(web.RestRemovePool)},
		rest.Route{"DELETE", "/hosts/:hostId", web.AuthorizedClient(web.RestRemoveHost)},
		rest.Route{"DELETE", "/services/:serviceId", web.AuthorizedClient(web.RestRemoveService)},
		rest.Route{"POST", "/pools/:poolId", web.AuthorizedClient(web.RestUpdatePool)},
		rest.Route{"POST", "/hosts/:hostId", web.AuthorizedClient(web.RestUpdateHost)},
		rest.Route{"POST", "/services/:serviceId", web.AuthorizedClient(web.RestUpdateService)},
		rest.Route{"GET", "/pools", web.AuthorizedClient(web.RestGetPools)},
		rest.Route{"GET", "/services", web.AuthorizedClient(web.RestGetServices)},
		rest.Route{"POST", "/login", web.RestLogin},
		rest.Route{"DELETE", "/login", web.RestLogout},
		rest.Route{"GET", "/favicon.ico", web.FavIcon},
		rest.Route{"GET", "/static*resource", web.StaticData},
	)
	http.ListenAndServe(":8787", &handler)
}

