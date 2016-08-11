package web

import (
	"github.com/control-center/serviced/domain/pool"
	"github.com/control-center/serviced/domain/read"
	"github.com/zenoss/go-json-rest"
)

// getPools returns the list of pools requested.  This call supports paging.
func getPools(w *rest.ResponseWriter, r *rest.Request, ctx *requestContext) {
	facade := ctx.getFacade()
	dataCtx := ctx.getDatastoreContext()

	pools, err := facade.GetResourcePools(dataCtx)
	if err != nil {
		restServerError(w, err)
		return
	}

	httpResponse := poolsResponse{
		Results: toReadPools(pools),
		Total:   len(pools),
		Links: []Link{Link{
			Rel:    "self",
			HRef:   r.URL.Path,
			Method: "GET",
		}},
	}

	w.WriteJson(httpResponse)
}

type poolsResponse struct {
	Results []read.Pool `json:"results"`
	Total   int         `json:"total"`
	Links   []Link      `json:"links"`
}

type Link struct {
	Rel    string `json:"rel"`
	HRef   string `json:"href"`
	Method string `json:"method"`
}

func toReadPools(resourcePools []pool.ResourcePool) []read.Pool {
	readPools := []read.Pool{}

	for _, resourcePool := range resourcePools {
		readPools = append(readPools, read.Pool{
			ID:                resourcePool.ID,
			Description:       resourcePool.Description,
			CreatedAt:         resourcePool.CreatedAt,
			UpdatedAt:         resourcePool.UpdatedAt,
			CoreCapacity:      resourcePool.CoreCapacity,
			MemoryCapacity:    resourcePool.MemoryCapacity,
			MemoryCommitment:  resourcePool.MemoryCommitment,
			ConnectionTimeout: resourcePool.ConnectionTimeout,
		})
	}

	return readPools
}
