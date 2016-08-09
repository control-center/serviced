package web

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/control-center/serviced/domain/pool"
	"github.com/control-center/serviced/domain/read"
	"github.com/zenoss/go-json-rest"
)

func buildPoolsQuery(r *rest.Request) (pool.ResourcePoolsQuery, error) {

	query := pool.ResourcePoolsQuery{}
	parameters := r.URL.Query()

	var err error

	var skip int
	if skipValue := parameters.Get("skip"); skipValue == "" {
		skip = 0
	} else {
		skip, err = strconv.Atoi(skipValue)
		if err != nil {
			return query, errors.New("Invalid value for parameter skip.  Must be an integer.")
		} else if skip < 0 {
			return query, errors.New("Invalid value for parameter skip.  Must be greater than zero.")
		}
	}

	var pull int
	if pullValue := parameters.Get("pull"); pullValue == "" {
		pull = 50000
	} else {
		pull, err = strconv.Atoi(pullValue)
		if err != nil {
			return query, errors.New("Invalid value for parameter pull.  Must be an integer.")
		} else if pull < 0 {
			return query, errors.New("Invalid value for parameter pull.  Must be greater than zero.")
		}
	}

	var order string
	if order = parameters.Get("order"); order == "" {
		order = "asc"
	}

	if order != "asc" && order != "desc" {
		return query, errors.New("Order invalid.  Must be 'asc' or 'desc'.")
	}

	sort := r.URL.Query().Get("sort")
	if sort == "" {
		sort = "ID"
	} else if sort != "ID" && sort != "Description" && sort != "CreatedAt" && sort != "UpdatedAt" {
		return query, errors.New("Sort invalid.  Must be ID, Description, CreatedAt or UpdatedAt.")
	}

	return pool.ResourcePoolsQuery{
		Skip:  skip,
		Pull:  pull,
		Sort:  sort,
		Order: order,
	}, nil
}

// getPools returns the list of pools requested.  This call supports paging.
func getPools(w *rest.ResponseWriter, r *rest.Request, ctx *requestContext) {
	query, err := buildPoolsQuery(r)
	if err != nil {
		writeJSON(w, err, http.StatusBadRequest)
		return
	}

	facade := ctx.getFacade()
	dataCtx := ctx.getDatastoreContext()

	response, err := facade.GetResourcePoolsByPage(dataCtx, query)
	if err != nil {
		restServerError(w, err)
		return
	}

	httpResponse := PoolsResponse{
		Results: toReadPools(response.Results),
		Total:   response.Total,
		Links: []Link{Link{
			Rel:    "self",
			HRef:   r.RequestURI,
			Method: "GET",
		}},
	}

	w.WriteJson(httpResponse)
}

type PoolsResponse struct {
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
