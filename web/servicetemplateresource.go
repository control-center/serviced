// Copyright 2014 The Serviced Authors.
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

	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/domain/servicetemplate"
	"github.com/control-center/serviced/node"
)


func restGetAppTemplates(w *rest.ResponseWriter, r *rest.Request, client *node.ControlClient) {
	var unused int
	var templatesMap map[string]servicetemplate.ServiceTemplate
	client.GetServiceTemplates(unused, &templatesMap)
	w.WriteJson(&templatesMap)
}

func restAddAppTemplate(w *rest.ResponseWriter, r *rest.Request, client *node.ControlClient) {
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

	var templateId string
	err = client.AddServiceTemplate(*template, &templateId)
	if err != nil {
		restServerError(w, err)
		return
	}

	w.WriteJson(&simpleResponse{templateId, servicesLinks()})
}

func restRemoveAppTemplate(w *rest.ResponseWriter, r *rest.Request, client *node.ControlClient) {
	templateID, err := url.QueryUnescape(r.PathParam("templateId"))
	var unused int

	if err != nil {
		restBadRequest(w, err)
		return
	}

	err = client.RemoveServiceTemplate(templateID, &unused)

	if err != nil {
		restServerError(w, err)
		return
	}

	w.WriteJson(&simpleResponse{templateID, servicesLinks()})
}

func restDeployAppTemplate(w *rest.ResponseWriter, r *rest.Request, client *node.ControlClient) {
	var payload dao.ServiceTemplateDeploymentRequest
	err := r.DecodeJsonPayload(&payload)
	if err != nil {
		glog.V(1).Info("Could not decode deployment payload: ", err)
		restBadRequest(w, err)
		return
	}
	var tenantIDs []string
	err = client.DeployTemplate(payload, &tenantIDs)
	if err != nil {
		glog.Error("Could not deploy template: ", err)
		restServerError(w, err)
		return
	}
	glog.V(0).Info("Deployed template ", payload)

	for _, tenantID := range tenantIDs {
		assignmentRequest := dao.AssignmentRequest{tenantID, "", true}
		if err := client.AssignIPs(assignmentRequest, nil); err != nil {
			glog.Errorf("Could not automatically assign IPs: %v", err)
			continue
		}
		glog.Infof("Automatically assigned IP addresses to service: %v", tenantID)
		// end of automatic IP assignment
		w.WriteJson(&simpleResponse{tenantID, servicesLinks()})
	}
}

func restDeployAppTemplateStatus(w *rest.ResponseWriter, r *rest.Request, client *node.ControlClient) {
	var payload dao.ServiceTemplateDeploymentRequest
	err := r.DecodeJsonPayload(&payload)
	if err != nil {
		glog.V(1).Info("Could not decode deployment payload: ", err)
		restBadRequest(w, err)
		return
	}
	status := ""

	err = client.DeployTemplateStatus(payload, &status)
	if err != nil {
		glog.Errorf("Unexpected error during template status: %v", err)
		writeJSON(w, &simpleResponse{err.Error(), homeLink()}, http.StatusInternalServerError)
		return
	}
	w.WriteJson(&simpleResponse{status, servicesLinks()})
}

func restDeployAppTemplateActive(w *rest.ResponseWriter, r *rest.Request, client *node.ControlClient) {
	var active []map[string]string

	err := client.DeployTemplateActive("", &active)
	if err != nil {
		glog.Errorf("Unexpected error during template status: %v", err)
		writeJSON(w, &simpleResponse{err.Error(), homeLink()}, http.StatusInternalServerError)
		return
	}
	w.WriteJson(&active)
}
