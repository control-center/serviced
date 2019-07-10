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
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain/service"
	"github.com/zenoss/go-json-rest"
)

func getAllServiceDetails(w *rest.ResponseWriter, r *rest.Request, c *requestContext) {
	ctx := c.getDatastoreContext()

	query, err := buildQuery(r)
	if err != nil {
		writeJSON(w, err, http.StatusBadRequest)
		return
	}

	details, err := c.getFacade().QueryServiceDetails(ctx, query)
	if err != nil {
		restServerError(w, err)
		return
	}

	w.WriteJson(details)
}

func getServiceDetails(w *rest.ResponseWriter, r *rest.Request, c *requestContext) {
	serviceID, err := url.QueryUnescape(r.PathParam("serviceId"))
	if err != nil {
		writeJSON(w, err, http.StatusBadRequest)
		return
	} else if len(serviceID) == 0 {
		writeJSON(w, "serviceId must be specified", http.StatusBadRequest)
		return
	}

	ctx := c.getDatastoreContext()

	var details *service.ServiceDetails
	if _, ancestors := r.URL.Query()["ancestors"]; ancestors {
		details, err = c.getFacade().GetServiceDetailsAncestry(ctx, serviceID)
	} else {
		details, err = c.getFacade().GetServiceDetails(ctx, serviceID)
	}

	if datastore.IsErrNoSuchEntity(err) {
		msg := fmt.Sprintf("Service %v Not Found", serviceID)
		writeJSON(w, msg, http.StatusNotFound)
		return
	} else if err != nil {
		restServerError(w, err)
		return
	}

	w.WriteJson(*details)
}

func getChildServiceDetails(w *rest.ResponseWriter, r *rest.Request, c *requestContext) {
	serviceID, err := url.QueryUnescape(r.PathParam("serviceId"))
	if err != nil {
		writeJSON(w, err, http.StatusBadRequest)
		return
	} else if len(serviceID) == 0 {
		writeJSON(w, "serviceId must be specified", http.StatusBadRequest)
		return
	}

	ctx := c.getDatastoreContext()

	tsince, err := getSinceParameter(r)
	if err != nil {
		restServerError(w, err)
		return
	}

	details, err := c.getFacade().GetServiceDetailsByParentID(ctx, serviceID, tsince)
	if err != nil {
		restServerError(w, err)
		return
	}

	w.WriteJson(details)
}

func getServiceContext(w *rest.ResponseWriter, r *rest.Request, c *requestContext) {
	serviceID, err := url.QueryUnescape(r.PathParam("serviceId"))
	if err != nil {
		writeJSON(w, err, http.StatusBadRequest)
		return
	} else if len(serviceID) == 0 {
		writeJSON(w, "serviceId must be specified", http.StatusBadRequest)
		return
	}

	ctx := c.getDatastoreContext()

	service, err := c.getFacade().GetService(ctx, serviceID)
	if err != nil {
		restServerError(w, err)
		return
	}

	w.WriteJson(service.Context)

}

func putServiceDetails(w *rest.ResponseWriter, r *rest.Request, c *requestContext) {
	ctx := c.getDatastoreContext()
	f := c.getFacade()

	serviceID, err := url.QueryUnescape(r.PathParam("serviceId"))
	if err != nil {
		writeJSON(w, err, http.StatusBadRequest)
		return
	}

	var payload service.ServiceDetails
	err = r.DecodeJsonPayload(&payload)
	if err != nil {
		writeJSON(w, err, http.StatusBadRequest)
		return
	}

	payload.ID = serviceID
	if invalid := payload.ValidEntity(); invalid != nil {
		writeJSON(w, err, http.StatusBadRequest)
	}

	svc, e := f.GetService(ctx, serviceID)
	if e != nil {
		restServerError(w, e)
		return
	}

	svc.Name = payload.Name
	svc.Description = payload.Description
	svc.PoolID = payload.PoolID
	svc.Launch = payload.Launch
	svc.Instances = payload.Instances
	svc.Startup = payload.Startup
	svc.RAMCommitment = payload.RAMCommitment
	svc.RAMThreshold = payload.RAMThreshold

	err = f.UpdateService(ctx, *svc)
	if err != nil {
		restServerError(w, err)
		return
	}

	writeJSON(w, "Service Updated.", http.StatusOK)
}

func putServiceContext(w *rest.ResponseWriter, r *rest.Request, c *requestContext) {
	ctx := c.getDatastoreContext()
	f := c.getFacade()

	serviceID, err := url.QueryUnescape(r.PathParam("serviceId"))
	if err != nil {
		writeJSON(w, err, http.StatusBadRequest)
		return
	}

	var payload map[string]interface{}
	err = r.DecodeJsonPayload(&payload)
	if err != nil {
		writeJSON(w, err, http.StatusBadRequest)
		return
	}

	svc, e := f.GetService(ctx, serviceID)
	if e != nil {
		restServerError(w, e)
		return
	}

	svc.Context = payload

	err = f.UpdateService(ctx, *svc)
	if err != nil {
		restServerError(w, err)
		return
	}

	writeJSON(w, "Service Context Updated.", http.StatusOK)
}

func getSinceParameter(r *rest.Request) (time.Duration, error) {
	var tsince time.Duration
	since := r.URL.Query().Get("since")
	if since == "" {
		tsince = time.Duration(0)
	} else {
		tint, err := strconv.ParseInt(since, 10, 64)
		if err != nil {
			return 0, err
		}
		tsince = time.Duration(tint) * time.Millisecond
	}
	return tsince, nil
}

func buildQuery(r *rest.Request) (service.Query, error) {
	query := service.Query{
		Name: r.URL.Query().Get("name"),
	}

	if r.URL.Query().Get("tags") != "" {
		query.Tags = strings.Split(r.URL.Query().Get("tags"), ",")
	} else {
		query.Tags = []string{}
	}

	if _, ok := r.URL.Query()["tenants"]; ok {
		query.Tenants = true
	}

	since := r.URL.Query().Get("since")
	if since != "" {
		i, err := strconv.ParseInt(since, 10, 64)
		if err != nil {
			return service.Query{}, err
		}
		query.Since = time.Duration(i) * time.Millisecond
	}

	return query, nil
}
