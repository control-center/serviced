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
	"strconv"
	"time"

	"github.com/zenoss/go-json-rest"
)

// getPools returns the list of pools requested.
func getHosts(w *rest.ResponseWriter, r *rest.Request, ctx *requestContext) {
	facade := ctx.getFacade()
	dataCtx := ctx.getDatastoreContext()

	hosts, err := facade.GetReadHosts(dataCtx)
	if err != nil {
		restServerError(w, err)
		return
	}

	w.WriteJson(hosts)
}

// getHostsForPool returns the list of hosts for a pool.
func getHostsForPool(w *rest.ResponseWriter, r *rest.Request, ctx *requestContext) {
	poolID, err := url.QueryUnescape(r.PathParam("poolId"))
	if err != nil {
		writeJSON(w, err, http.StatusBadRequest)
		return
	} else if len(poolID) == 0 {
		writeJSON(w, "poolId must be specified", http.StatusBadRequest)
		return
	}

	facade := ctx.getFacade()
	dataCtx := ctx.getDatastoreContext()

	hosts, err := facade.FindReadHostsInPool(dataCtx, poolID)
	if err != nil {
		restServerError(w, err)
		return
	}

	w.WriteJson(hosts)
}

// getHostStatus return status information for hosts.  This includes the memory usage and
// whether or not the host is active.
func getHostStatuses(w *rest.ResponseWriter, r *rest.Request, ctx *requestContext) {
	facade := ctx.getFacade()
	dataCtx := ctx.getDatastoreContext()

	values := r.URL.Query()

	var hostIDs []string
	if _, ok := values["hostId"]; ok {
		hostIDs = values["hostId"]
	} else {
		hosts, err := facade.GetReadHosts(dataCtx)
		if err != nil {
			restServerError(w, err)
			return
		}

		hostIDs = make([]string, len(hosts))
		for i, host := range hosts {
			hostIDs[i] = host.ID
		}
	}

	since := values.Get("since")
	var duration time.Duration
	if since == "" {
		duration = time.Hour
	} else {
		tint, err := strconv.ParseInt(since, 10, 64)
		if err != nil {
			restServerError(w, err)
			return
		}
		duration = time.Duration(tint) * time.Millisecond
	}

	statuses, err := facade.GetHostStatuses(dataCtx, hostIDs, time.Now().Add(-duration))
	if err != nil {
		restServerError(w, err)
		return
	}

	w.WriteJson(statuses)
}
