package web

import (
	"github.com/control-center/serviced/domain/host"
	"github.com/zenoss/go-json-rest"
)

// getPools returns the list of pools requested.  This call supports paging.
func getHosts(w *rest.ResponseWriter, r *rest.Request, ctx *requestContext) {
	facade := ctx.getFacade()
	dataCtx := ctx.getDatastoreContext()

	hosts, err := facade.GetReadHosts(dataCtx)
	if err != nil {
		restServerError(w, err)
		return
	}

	response := hostsResponse{
		Results: hosts,
		Total:   len(hosts),
		Links: []APILink{APILink{
			Rel:    "self",
			HRef:   r.URL.Path,
			Method: "GET",
		}},
	}

	w.WriteJson(response)
}

type hostsResponse struct {
	Results []host.ReadHost `json:"results"`
	Total   int             `json:"total"`
	Links   []APILink       `json:"links"`
}
