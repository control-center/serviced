package main

import (
	"github.com/ant0ine/go-json-rest"
	"github.com/zenoss/serviced/web"
	"net/http"
)

func main() {

	handler := rest.ResourceHandler{
		EnableRelaxedContentType: true,
        }
	handler.SetRoutes(
		rest.Route{"GET", "/", web.MainIndex},
		rest.Route{"GET", "/login", web.ContentLoginPage},
		rest.Route{"GET", "/hosts", web.AuthorizedClient(web.RestGetHosts)},
		rest.Route{"POST", "/hosts/add", web.AuthorizedClient(web.RestAddHost)},
		rest.Route{"POST", "/pools/add", web.AuthorizedClient(web.RestAddPool)},
		rest.Route{"GET", "/pools/:poolId/hosts", web.AuthorizedClient(web.RestGetHostsForResourcePool)},
		rest.Route{"DELETE", "/pools/:poolId", web.AuthorizedClient(web.RestRemovePool)},
		rest.Route{"DELETE", "/hosts/:hostId", web.AuthorizedClient(web.RestRemoveHost)},
		rest.Route{"POST", "/pools/:poolId", web.AuthorizedClient(web.RestUpdatePool)},
		rest.Route{"POST", "/hosts/:hostId", web.AuthorizedClient(web.RestUpdateHost)},
		rest.Route{"GET", "/pools", web.AuthorizedClient(web.RestGetPools)},
		rest.Route{"POST", "/start", web.AuthorizedOnly(web.RestStartServer)},
		rest.Route{"POST", "/login", web.RestLogin},
		rest.Route{"DELETE", "/login", web.RestLogout},
		rest.Route{"GET", "/user", web.AuthorizedOnly(web.RestUser)},
		rest.Route{"GET", "/favicon.ico", web.FavIcon},
		rest.Route{"GET", "/static*resource", web.StaticData},
	)
	http.ListenAndServe(":8787", &handler)
}

