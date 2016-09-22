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

	"github.com/control-center/serviced/domain/service"
	"github.com/zenoss/go-json-rest"
)

func getAllServiceDetails(w *rest.ResponseWriter, r *rest.Request, c *requestContext) {
	ctx := c.getDatastoreContext()

	var err error
	var details []service.ServiceDetails

	if _, ok := r.URL.Query()["tenants"]; ok {
		details, err = c.getFacade().GetServiceDetailsByParentID(ctx, "")
	} else {
		details, err = c.getFacade().GetAllServiceDetails(ctx)
	}

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

	details, err := c.getFacade().GetServiceDetails(ctx, serviceID)
	if err != nil {
		restServerError(w, err)
		return
	}

	if details == nil {
		msg := fmt.Sprintf("Service %v Not Found", serviceID)
		writeJSON(w, msg, http.StatusNotFound)
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

	details, err := c.getFacade().GetServiceDetailsByParentID(ctx, serviceID)
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
	svc.Instances = payload.Instances
	svc.Startup = payload.Startup
	svc.RAMCommitment = payload.RAMCommitment

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
