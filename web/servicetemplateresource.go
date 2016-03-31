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
	"bytes"
	"io"
	"net/http"
	"net/url"

	"github.com/zenoss/glog"
	"github.com/zenoss/go-json-rest"

	"github.com/control-center/serviced/domain/addressassignment"
	"github.com/control-center/serviced/domain/servicetemplate"
)


func restGetAppTemplates(w *rest.ResponseWriter, r *rest.Request, ctx *requestContext) {
	templatesMap, err := ctx.getFacade().GetServiceTemplates(ctx.getDatastoreContext())
	if err != nil {
		restServerError(w, err)
		return
	}

	w.WriteJson(&templatesMap)
}

func restAddAppTemplate(w *rest.ResponseWriter, r *rest.Request, ctx *requestContext) {
	// read uploaded file
	file, _, err := r.FormFile("tpl")
	if err != nil {
		restBadRequest(w, err)
		return
	}
	defer file.Close()

	var b bytes.Buffer
	_, err = io.Copy(&b, file)
	template, err := servicetemplate.FromJSON(b.String())
	if err != nil {
		restServerError(w, err)
		return
	}

	templateID, err := ctx.getFacade().AddServiceTemplate(ctx.getDatastoreContext(), *template)
	if err != nil {
		restServerError(w, err)
		return
	}

	w.WriteJson(&simpleResponse{templateID, servicesLinks()})
}

func restRemoveAppTemplate(w *rest.ResponseWriter, r *rest.Request, ctx *requestContext) {
	templateID, err := url.QueryUnescape(r.PathParam("templateId"))
	if err != nil {
		restBadRequest(w, err)
		return
	}

	err = ctx.getFacade().RemoveServiceTemplate(ctx.getDatastoreContext(), templateID)
	if err != nil {
		restServerError(w, err)
		return
	}

	w.WriteJson(&simpleResponse{templateID, servicesLinks()})
}

func restDeployAppTemplate(w *rest.ResponseWriter, r *rest.Request, ctx *requestContext) {
	var payload servicetemplate.ServiceTemplateDeploymentRequest
	err := r.DecodeJsonPayload(&payload)
	if err != nil {
		glog.V(1).Info("Could not decode deployment payload: ", err)
		restBadRequest(w, err)
		return
	}
	tenantIDs, err := ctx.getFacade().DeployTemplate(ctx.getDatastoreContext(), payload.PoolID, payload.TemplateID, payload.DeploymentID)
	if err != nil {
		glog.Error("Could not deploy template: ", err)
		restServerError(w, err)
		return
	}
	glog.V(0).Info("Deployed template ", payload)

	// FIXME: This REST implementation isn't compatible with the Facade implementation - if the Facade ever returns
	//        more than one value, then the code below will only return a result for the first value.
	//        When can the Facade return more than one value?
	for _, tenantID := range tenantIDs {
		// FIXME: Business logic like assigning IPs does NOT belong in the REST tier.
		//        This logic should be moved into the Facade.
		assignmentRequest := addressassignment.AssignmentRequest{tenantID, "", true}
		if err := ctx.getFacade().AssignIPs(ctx.getDatastoreContext(), assignmentRequest); err != nil {
			// FIXME: This error is never reported to the client
			glog.Errorf("Could not automatically assign IPs: %v", err)
			continue
		}
		glog.Infof("Automatically assigned IP addresses to service: %v", tenantID)
		// end of automatic IP assignment
		w.WriteJson(&simpleResponse{tenantID, servicesLinks()})
	}
}

func restDeployAppTemplateStatus(w *rest.ResponseWriter, r *rest.Request, ctx *requestContext) {
	var payload servicetemplate.ServiceTemplateDeploymentRequest
	err := r.DecodeJsonPayload(&payload)
	if err != nil {
		glog.V(1).Info("Could not decode deployment payload: ", err)
		restBadRequest(w, err)
		return
	}

	status, err := ctx.getFacade().DeployTemplateStatus(payload.DeploymentID)
	if err != nil {
		glog.Errorf("Unexpected error during template status: %v", err)
		writeJSON(w, &simpleResponse{err.Error(), homeLink()}, http.StatusInternalServerError)
		return
	}
	w.WriteJson(&simpleResponse{status, servicesLinks()})
}

func restDeployAppTemplateActive(w *rest.ResponseWriter, r *rest.Request, ctx *requestContext) {
	active, err := ctx.getFacade().DeployTemplateActive()
	if err != nil {
		glog.Errorf("Unexpected error during template status: %v", err)
		writeJSON(w, &simpleResponse{err.Error(), homeLink()}, http.StatusInternalServerError)
		return
	}
	w.WriteJson(&active)
}
