// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package web

import (
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/zenoss/glog"
	"github.com/zenoss/go-json-rest"

	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/servicetemplate"
	"github.com/control-center/serviced/isvcs"
	"github.com/control-center/serviced/node"
	"github.com/control-center/serviced/servicedversion"
)

var empty interface{}

type handlerFunc func(w *rest.ResponseWriter, r *rest.Request)
type handlerClientFunc func(w *rest.ResponseWriter, r *rest.Request, client *node.ControlClient)

func restGetAppTemplates(w *rest.ResponseWriter, r *rest.Request, client *node.ControlClient) {
	var unused int
	var templatesMap map[string]*servicetemplate.ServiceTemplate
	client.GetServiceTemplates(unused, &templatesMap)
	w.WriteJson(&templatesMap)
}

func restDeployAppTemplate(w *rest.ResponseWriter, r *rest.Request, client *node.ControlClient) {
	var payload dao.ServiceTemplateDeploymentRequest
	err := r.DecodeJsonPayload(&payload)
	if err != nil {
		glog.V(1).Info("Could not decode deployment payload: ", err)
		restBadRequest(w, err)
		return
	}
	var tenantID string
	err = client.DeployTemplate(payload, &tenantID)
	if err != nil {
		glog.Error("Could not deploy template: ", err)
		restServerError(w, err)
		return
	}
	glog.V(0).Info("Deployed template ", payload)

	assignmentRequest := dao.AssignmentRequest{tenantID, "", true}
	if err := client.AssignIPs(assignmentRequest, nil); err != nil {
		glog.Error("Could not automatically assign IPs: %v", err)
		return
	}

	glog.Infof("Automatically assigned IP addresses to service: %v", tenantID)
	// end of automatic IP assignment

	w.WriteJson(&simpleResponse{tenantID, servicesLinks()})
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

func filterByNameRegex(nmregex string, services []*service.Service) ([]*service.Service, error) {
	r, err := regexp.Compile(nmregex)
	if err != nil {
		glog.Errorf("Bad name regexp :%s", nmregex)
		return nil, err
	}

	matches := []*service.Service{}
	for _, service := range services {
		if r.MatchString(service.Name) {
			matches = append(matches, service)
		}
	}
	glog.V(2).Infof("Returning %d named services", len(matches))
	return matches, nil
}

func getTaggedServices(client *node.ControlClient, tags, nmregex string) ([]*service.Service, error) {
	services := []*service.Service{}
	var ts interface{}
	ts = strings.Split(tags, ",")
	if err := client.GetTaggedServices(&ts, &services); err != nil {
		glog.Errorf("Could not get tagged services: %v", err)
		return nil, err
	}

	if nmregex != "" {
		return filterByNameRegex(nmregex, services)
	}
	glog.V(2).Infof("Returning %d tagged services", len(services))
	return services, nil
}

func getNamedServices(client *node.ControlClient, nmregex string) ([]*service.Service, error) {
	services := []*service.Service{}
	if err := client.GetServices(&empty, &services); err != nil {
		glog.Errorf("Could not get named services: %v", err)
		return nil, err
	}

	return filterByNameRegex(nmregex, services)
}

func getServices(client *node.ControlClient) ([]*service.Service, error) {
	services := []*service.Service{}
	if err := client.GetServices(&empty, &services); err != nil {
		glog.Errorf("Could not get services: %v", err)
		return nil, err
	}

	glog.V(2).Infof("Returning %d services", len(services))
	return services, nil
}

func getISVCS() []*service.Service {
	services := []*service.Service{}
	services = append(services, &isvcs.InternalServicesISVC)
	services = append(services, &isvcs.ElasticsearchISVC)
	services = append(services, &isvcs.ZookeeperISVC)
	services = append(services, &isvcs.LogstashISVC)
	services = append(services, &isvcs.OpentsdbISVC)
	services = append(services, &isvcs.CeleryISVC)
	services = append(services, &isvcs.DockerRegistryISVC)
	return services
}

func restGetAllServices(w *rest.ResponseWriter, r *rest.Request, client *node.ControlClient) {
	if tags := r.URL.Query().Get("tags"); tags != "" {
		nmregex := r.URL.Query().Get("name")
		result, err := getTaggedServices(client, tags, nmregex)
		if err != nil {
			restServerError(w, err)
			return
		}

		w.WriteJson(&result)
		return
	}

	if nmregex := r.URL.Query().Get("name"); nmregex != "" {
		result, err := getNamedServices(client, nmregex)
		if err != nil {
			restServerError(w, err)
			return
		}

		w.WriteJson(&result)
		return
	}

	result, err := getServices(client)
	result = append(result, getISVCS()...)
	if err != nil {
		restServerError(w, err)
		return
	}

	w.WriteJson(&result)
}

func restGetRunningForHost(w *rest.ResponseWriter, r *rest.Request, client *node.ControlClient) {
	hostID, err := url.QueryUnescape(r.PathParam("hostId"))
	if err != nil {
		restBadRequest(w, err)
		return
	}
	var services []*dao.RunningService
	err = client.GetRunningServicesForHost(hostID, &services)
	if err != nil {
		glog.Errorf("Could not get services: %v", err)
		restServerError(w, err)
		return
	}
	if services == nil {
		glog.V(3).Info("Running services was nil, returning empty list instead")
		services = []*dao.RunningService{}
	}
	glog.V(2).Infof("Returning %d running services for host %s", len(services), hostID)
	w.WriteJson(&services)
}

func restGetRunningForService(w *rest.ResponseWriter, r *rest.Request, client *node.ControlClient) {
	serviceID, err := url.QueryUnescape(r.PathParam("serviceId"))
	if strings.Contains(serviceID, "isvc-") {
		w.WriteJson([]*dao.RunningService{})
		return
	}
	if err != nil {
		restBadRequest(w, err)
		return
	}
	var services []*dao.RunningService
	err = client.GetRunningServicesForService(serviceID, &services)
	if err != nil {
		glog.Errorf("Could not get services: %v", err)
		restServerError(w, err)
		return
	}
	if services == nil {
		glog.V(3).Info("Running services was nil, returning empty list instead")
		services = []*dao.RunningService{}
	}
	glog.V(2).Infof("Returning %d running services for service %s", len(services), serviceID)
	w.WriteJson(&services)
}

func restGetAllRunning(w *rest.ResponseWriter, r *rest.Request, client *node.ControlClient) {
	var services []*dao.RunningService
	err := client.GetRunningServices(&empty, &services)
	if err != nil {
		glog.Errorf("Could not get services: %v", err)
		restServerError(w, err)
		return
	}
	if services == nil {
		glog.V(3).Info("Services was nil, returning empty list instead")
		services = []*dao.RunningService{}
	}
	glog.V(2).Infof("Return %d running services", len(services))
	w.WriteJson(&services)
}

func restKillRunning(w *rest.ResponseWriter, r *rest.Request, client *node.ControlClient) {
	serviceStateID, err := url.QueryUnescape(r.PathParam("serviceStateId"))
	if err != nil {
		restBadRequest(w, err)
		return
	}
	hostID, err := url.QueryUnescape(r.PathParam("hostId"))
	if err != nil {
		restBadRequest(w, err)
		return
	}
	request := dao.HostServiceRequest{hostID, serviceStateID}
	glog.V(1).Info("Received request to kill ", request)

	var unused int
	err = client.StopRunningInstance(request, &unused)
	if err != nil {
		glog.Errorf("Unable to stop service: %v", err)
		restServerError(w, err)
		return
	}

	w.WriteJson(&simpleResponse{"Marked for death", servicesLinks()})
}

func restGetTopServices(w *rest.ResponseWriter, r *rest.Request, client *node.ControlClient) {
	var allServices []*service.Service
	topServices := []*service.Service{}

	err := client.GetServices(&empty, &allServices)
	if err != nil {
		glog.Errorf("Could not get services: %v", err)
		restServerError(w, err)
		return
	}
	for _, service := range allServices {
		if len(service.ParentServiceID) == 0 {
			topServices = append(topServices, service)
		}
	}
	topServices = append(topServices, &isvcs.InternalServicesISVC)
	glog.V(2).Infof("Returning %d services as top services", len(topServices))
	w.WriteJson(&topServices)
}

func restGetService(w *rest.ResponseWriter, r *rest.Request, client *node.ControlClient) {
	sid, err := url.QueryUnescape(r.PathParam("serviceId"))
	if err != nil {
		restBadRequest(w, err)
		return
	}

	if strings.Contains(sid, "isvc-") {
		w.WriteJson(isvcs.ISVCSMap[sid])
		return
	}

	var allServices []*service.Service

	if err := client.GetServices(&empty, &allServices); err != nil {
		glog.Errorf("Could not get services: %v", err)
		restServerError(w, err)
		return
	}

	for _, service := range allServices {
		if service.ID == sid {
			w.WriteJson(&service)
			return
		}
	}

	glog.Errorf("No such service [%v]", sid)
	restServerError(w, err)
}

func restAddService(w *rest.ResponseWriter, r *rest.Request, client *node.ControlClient) {
	var payload service.Service
	var serviceID string
	err := r.DecodeJsonPayload(&payload)
	if err != nil {
		glog.V(1).Info("Could not decode service payload: ", err)
		restBadRequest(w, err)
		return
	}
	svc, err := service.NewService()
	if err != nil {
		glog.Errorf("Could not create service: %v", err)
		restServerError(w, err)
		return
	}
	now := time.Now()
	svc.Name = payload.Name
	svc.Description = payload.Description
	svc.Context = payload.Context
	svc.Tags = payload.Tags
	svc.PoolID = payload.PoolID
	svc.ImageID = payload.ImageID
	svc.Startup = payload.Startup
	svc.Instances = payload.Instances
	svc.ParentServiceID = payload.ParentServiceID
	svc.DesiredState = payload.DesiredState
	svc.Launch = payload.Launch
	svc.Endpoints = payload.Endpoints
	svc.ConfigFiles = payload.ConfigFiles
	svc.OriginalConfigs = payload.OriginalConfigs
	svc.Volumes = payload.Volumes
	svc.CreatedAt = now
	svc.UpdatedAt = now

	//for each endpoint, evaluate it's EndpointTemplates
	getSvc := func(svcID string) (service.Service, error) {
		svc := service.Service{}
		err := client.GetService(svcID, &svc)
		return svc, err
	}
	findChild := func(svcID, childName string) (service.Service, error) {
		svc := service.Service{}
		err := client.FindChildService(dao.FindChildRequest{svcID, childName}, &svc)
		return svc, err
	}
	if err = svc.EvaluateEndpointTemplates(getSvc, findChild); err != nil {
		glog.Errorf("Unable to evaluate service endpoints: %v", err)
		restServerError(w, err)
		return
	}

	//add the service to the data store
	err = client.AddService(*svc, &serviceID)
	if err != nil {
		glog.Errorf("Unable to add service: %v", err)
		restServerError(w, err)
		return
	}

	//automatically assign virtual ips to new service
	request := dao.AssignmentRequest{ServiceID: svc.ID, IPAddress: "", AutoAssignment: true}
	if err := client.AssignIPs(request, nil); err != nil {
		glog.Error("Failed to automatically assign IPs: %+v -> %v", request, err)
		restServerError(w, err)
		return
	}

	glog.V(0).Info("Added service ", serviceID)
	w.WriteJson(&simpleResponse{"Added service", serviceLinks(serviceID)})
}

func restDeployService(w *rest.ResponseWriter, r *rest.Request, client *node.ControlClient) {
	var payload dao.ServiceDeploymentRequest
	err := r.DecodeJsonPayload(&payload)
	if err != nil {
		glog.V(1).Info("Could not decode service payload: ", err)
		restBadRequest(w, err)
		return
	}

	var serviceID string
	err = client.DeployService(payload, &serviceID)
	if err != nil {
		glog.Errorf("Unable to deploy service: %v", err)
		restServerError(w, err)
		return
	}

	glog.V(0).Info("Deployed service ", serviceID)
	w.WriteJson(&simpleResponse{"Deployed service", serviceLinks(serviceID)})
}

func restUpdateService(w *rest.ResponseWriter, r *rest.Request, client *node.ControlClient) {
	serviceID, err := url.QueryUnescape(r.PathParam("serviceId"))
	glog.V(3).Infof("Received update request for %s", serviceID)
	if err != nil {
		restBadRequest(w, err)
		return
	}
	var payload service.Service
	var unused int
	err = r.DecodeJsonPayload(&payload)
	if err != nil {
		glog.V(1).Info("Could not decode service payload: ", err)
		restBadRequest(w, err)
		return
	}
	err = client.UpdateService(payload, &unused)
	if err != nil {
		glog.Errorf("Unable to update service %s: %v", serviceID, err)
		restServerError(w, err)
		return
	}
	glog.V(1).Info("Updated service ", serviceID)
	w.WriteJson(&simpleResponse{"Updated service", serviceLinks(serviceID)})
}

func restRemoveService(w *rest.ResponseWriter, r *rest.Request, client *node.ControlClient) {
	var unused int
	serviceID, err := url.QueryUnescape(r.PathParam("serviceId"))
	if err != nil {
		restBadRequest(w, err)
		return
	}
	err = client.RemoveService(serviceID, &unused)
	if err != nil {
		glog.Errorf("Could not remove service: %v", err)
		restServerError(w, err)
		return
	}
	glog.V(0).Info("Removed service ", serviceID)
	w.WriteJson(&simpleResponse{"Removed service", servicesLinks()})
}

func restGetServiceLogs(w *rest.ResponseWriter, r *rest.Request, client *node.ControlClient) {
	serviceID, err := url.QueryUnescape(r.PathParam("serviceId"))
	if err != nil {
		restBadRequest(w, err)
		return
	}
	var logs string
	err = client.GetServiceLogs(serviceID, &logs)
	if err != nil {
		glog.Errorf("Unexpected error getting service logs: %v", err)
		restServerError(w, err)
		return
	}
	w.WriteJson(&simpleResponse{logs, serviceLinks(serviceID)})
}

// restStartService starts the service with the given id and all of its children
func restStartService(w *rest.ResponseWriter, r *rest.Request, client *node.ControlClient) {
	serviceID, err := url.QueryUnescape(r.PathParam("serviceId"))
	if err != nil {
		restBadRequest(w, err)
		return
	}
	var i string
	err = client.StartService(serviceID, &i)
	if err != nil {
		glog.Errorf("Unexpected error starting service: %v", err)
		restServerError(w, err)
		return
	}
	w.WriteJson(&simpleResponse{"Started service", serviceLinks(serviceID)})
}

// restStopService stop the service with the given id and all of its children
func restStopService(w *rest.ResponseWriter, r *rest.Request, client *node.ControlClient) {
	serviceID, err := url.QueryUnescape(r.PathParam("serviceId"))
	if err != nil {
		restBadRequest(w, err)
		return
	}
	var i int
	err = client.StopService(serviceID, &i)
	if err != nil {
		glog.Errorf("Unexpected error stopping service: %v", err)
		restServerError(w, err)
		return
	}
	w.WriteJson(&simpleResponse{"Stopped service", serviceLinks(serviceID)})
}

func restSnapshotService(w *rest.ResponseWriter, r *rest.Request, client *node.ControlClient) {
	serviceID, err := url.QueryUnescape(r.PathParam("serviceId"))
	if err != nil {
		restBadRequest(w, err)
		return
	}
	var label string
	err = client.Snapshot(serviceID, &label)
	if err != nil {
		glog.Errorf("Unexpected error snapshotting service: %v", err)
		restServerError(w, err)
		return
	}
	w.WriteJson(&simpleResponse{label, serviceLinks(serviceID)})
}

func restGetRunningService(w *rest.ResponseWriter, r *rest.Request, client *node.ControlClient) {
	serviceStateID, err := url.QueryUnescape(r.PathParam("serviceStateId"))
	if err != nil {
		restBadRequest(w, err)
		return
	}
	serviceID, err := url.QueryUnescape(r.PathParam("serviceId"))
	if err != nil {
		restBadRequest(w, err)
		return
	}
	request := dao.ServiceStateRequest{serviceID, serviceStateID}

	var running dao.RunningService
	err = client.GetRunningService(request, &running)
	if err != nil {
		glog.Errorf("Unexpected error retrieving services: %v", err)
		restServerError(w, err)
		return
	}
	w.WriteJson(running)
}

func restGetServiceStateLogs(w *rest.ResponseWriter, r *rest.Request, client *node.ControlClient) {
	serviceStateID, err := url.QueryUnescape(r.PathParam("serviceStateId"))
	if err != nil {
		restBadRequest(w, err)
		return
	}
	serviceID, err := url.QueryUnescape(r.PathParam("serviceId"))
	if err != nil {
		restBadRequest(w, err)
		return
	}
	request := dao.ServiceStateRequest{serviceID, serviceStateID}

	var logs string
	err = client.GetServiceStateLogs(request, &logs)
	if err != nil {
		glog.Errorf("Unexpected error getting service state logs: %v", err)
		restServerError(w, err)
		return
	}
	w.WriteJson(&simpleResponse{logs, servicesLinks()})
}

func restGetServicedVersion(w *rest.ResponseWriter, r *rest.Request, client *node.ControlClient) {
	w.WriteJson(servicedversion.GetVersion())
}

func RestBackupCreate(w *rest.ResponseWriter, r *rest.Request, client *node.ControlClient) {
	home := os.Getenv("SERVICED_HOME")
	if home == "" {
		glog.Infof("SERVICED_HOME not set.  Backups will save to /tmp.")
		home = "/tmp"
	}

	dir := home + "/backup"
	filepath := ""
	err := client.AsyncBackup(dir, &filepath)
	if err != nil {
		glog.Errorf("Unexpected error during backup: %v", err)
		restServerError(w, err)
		return
	}
	w.WriteJson(&simpleResponse{filepath, servicesLinks()})
}

func RestBackupRestore(w *rest.ResponseWriter, r *rest.Request, client *node.ControlClient) {
	home := os.Getenv("SERVICED_HOME")
	if home == "" {
		glog.Infof("SERVICED_HOME not set.  Backups will save to /tmp.")
		home = "/tmp"
	}

	err := r.ParseForm()
	filepath := r.FormValue("filename")

	if err != nil || filepath == "" {
		restBadRequest(w, err)
		return
	}

	unused := 0

	err = client.AsyncRestore(home+"/backup/"+filepath, &unused)
	if err != nil {
		glog.Errorf("Unexpected error during restore: %v", err)
		restServerError(w, err)
		return
	}
	w.WriteJson(&simpleResponse{string(unused), servicesLinks()})
}

func RestBackupFileList(w *rest.ResponseWriter, r *rest.Request, ctx *requestContext) {
	type JsonizableFileInfo struct {
		Name    string      `json:"name"`
		Size    int64       `json:"size"`
		Mode    os.FileMode `json:"mode"`
		ModTime time.Time   `json:"mod_time"`
	}

	fileData := []JsonizableFileInfo{}
	home := os.Getenv("SERVICED_HOME")
	if home == "" {
		glog.Infof("SERVICED_HOME not set.  Backups will save to /tmp.")
		home = "/tmp"
	}
	backupFiles, _ := ioutil.ReadDir(home + "/backup")

	for _, backupFileInfo := range backupFiles {
		if !backupFileInfo.IsDir() {
			fileInfo := JsonizableFileInfo{backupFileInfo.Name(), backupFileInfo.Size(), backupFileInfo.Mode(), backupFileInfo.ModTime()}
			fileData = append(fileData, fileInfo)
		}
	}

	w.WriteJson(&fileData)
}

func RestBackupStatus(w *rest.ResponseWriter, r *rest.Request, client *node.ControlClient) {
	backupStatus := ""
	err := client.BackupStatus("", &backupStatus)
	if err != nil {
		glog.Errorf("Unexpected error during backup status: %v", err)
		writeJSON(w, &simpleResponse{err.Error(), homeLink()}, http.StatusInternalServerError)
		return
	}
	w.WriteJson(&simpleResponse{backupStatus, servicesLinks()})
}

func RestRestoreStatus(w *rest.ResponseWriter, r *rest.Request, client *node.ControlClient) {
	restoreStatus := ""
	err := client.RestoreStatus("", &restoreStatus)
	if err != nil {
		glog.Errorf("Unexpected error during restore status: %v", err)
		writeJSON(w, &simpleResponse{err.Error(), homeLink()}, http.StatusInternalServerError)
		return
	}
	w.WriteJson(&simpleResponse{restoreStatus, servicesLinks()})
}
