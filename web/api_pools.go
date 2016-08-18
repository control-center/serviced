// Copyright 2016 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
		Links: []read.Link{read.Link{
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
	Links   []read.Link `json:"links"`
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
