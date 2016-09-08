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
	"net/http"
	"net/url"

	"github.com/control-center/serviced/domain/service"
	"github.com/zenoss/go-json-rest"
)

// getPools returns the list of details requested.  This call supports paging.
func getChildServiceDetails(w *rest.ResponseWriter, r *rest.Request, ctx *requestContext) {
	serviceID, err := url.QueryUnescape(r.PathParam("serviceId"))
	if err != nil {
		writeJSON(w, err, http.StatusBadRequest)
		return
	} else if len(serviceID) == 0 {
		writeJSON(w, "serviceId must be specified", http.StatusBadRequest)
		return
	}

	facade := ctx.getFacade()
	dataCtx := ctx.getDatastoreContext()

	details, err := facade.GetChildServiceDetails(dataCtx, serviceID)
	if err != nil {
		restServerError(w, err)
		return
	}

	response := childServiceDetailsResponse{
		Results: details,
		Total:   len(details),
		Links: []APILink{APILink{
			Rel:    "self",
			HRef:   r.URL.Path,
			Method: "GET",
		}},
	}

	w.WriteJson(response)
}

type childServiceDetailsResponse struct {
	Results []service.ServiceDetails `json:"results"`
	Total   int                      `json:"total"`
	Links   []APILink                `json:"links"`
}
