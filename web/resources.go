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
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
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
	"github.com/control-center/serviced/utils"
)

var empty interface{}

type handlerFunc func(w *rest.ResponseWriter, r *rest.Request)
type handlerClientFunc func(w *rest.ResponseWriter, r *rest.Request, client *node.ControlClient)

func restDockerIsLoggedIn(w *rest.ResponseWriter, r *rest.Request, client *node.ControlClient) {
	w.WriteJson(&map[string]bool{"dockerLoggedIn": utils.DockerIsLoggedIn()})
}

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

func filterByNameRegex(nmregex string, services []service.Service) ([]service.Service, error) {
	r, err := regexp.Compile(nmregex)
	if err != nil {
		glog.Errorf("Bad name regexp :%s", nmregex)
		return nil, err
	}

	matches := []service.Service{}
	for _, service := range services {
		if r.MatchString(service.Name) {
			matches = append(matches, service)
		}
	}
	glog.V(2).Infof("Returning %d named services", len(matches))
	return matches, nil
}

func getTaggedServices(client *node.ControlClient, tags, nmregex string) ([]service.Service, error) {
	services := []service.Service{}
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

func getNamedServices(client *node.ControlClient, nmregex string) ([]service.Service, error) {
	services := []service.Service{}
	if err := client.GetServices(&empty, &services); err != nil {
		glog.Errorf("Could not get named services: %v", err)
		return nil, err
	}

	return filterByNameRegex(nmregex, services)
}

func getServices(client *node.ControlClient) ([]service.Service, error) {
	services := []service.Service{}
	if err := client.GetServices(&empty, &services); err != nil {
		glog.Errorf("Could not get services: %v", err)
		return nil, err
	}

	glog.V(2).Infof("Returning %d services", len(services))
	return services, nil
}

func getISVCS() []service.Service {
	services := []service.Service{}
	services = append(services, isvcs.InternalServicesISVC)
	services = append(services, isvcs.ElasticsearchISVC)
	services = append(services, isvcs.ZookeeperISVC)
	services = append(services, isvcs.LogstashISVC)
	services = append(services, isvcs.OpentsdbISVC)
	services = append(services, isvcs.CeleryISVC)
	services = append(services, isvcs.DockerRegistryISVC)
	return services
}

func getIRS() []dao.RunningService {
	services := []dao.RunningService{}
	services = append(services, isvcs.InternalServicesIRS)
	services = append(services, isvcs.ElasticsearchIRS)
	services = append(services, isvcs.ZookeeperIRS)
	services = append(services, isvcs.LogstashIRS)
	services = append(services, isvcs.OpentsdbIRS)
	services = append(services, isvcs.CeleryIRS)
	services = append(services, isvcs.DockerRegistryIRS)
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

		for _, svc := range result {
			fillBuiltinMetrics(&svc)
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

		for _, svc := range result {
			fillBuiltinMetrics(&svc)
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

	for _, svc := range result {
		fillBuiltinMetrics(&svc)
	}
	w.WriteJson(&result)
}

func restGetRunningForHost(w *rest.ResponseWriter, r *rest.Request, client *node.ControlClient) {
	hostID, err := url.QueryUnescape(r.PathParam("hostId"))
	if err != nil {
		restBadRequest(w, err)
		return
	}
	var services []dao.RunningService
	err = client.GetRunningServicesForHost(hostID, &services)
	if err != nil {
		glog.Errorf("Could not get services: %v", err)
		restServerError(w, err)
		return
	}
	if services == nil {
		glog.V(3).Info("Running services was nil, returning empty list instead")
		services = []dao.RunningService{}
	}
	glog.V(2).Infof("Returning %d running services for host %s", len(services), hostID)
	w.WriteJson(&services)
}

func restGetRunningForService(w *rest.ResponseWriter, r *rest.Request, client *node.ControlClient) {
	serviceID, err := url.QueryUnescape(r.PathParam("serviceId"))
	if strings.Contains(serviceID, "isvc-") {
		w.WriteJson([]dao.RunningService{})
		return
	}
	if err != nil {
		restBadRequest(w, err)
		return
	}
	var services []dao.RunningService
	err = client.GetRunningServicesForService(serviceID, &services)
	if err != nil {
		glog.Errorf("Could not get running services for %s: %v", serviceID, err)
		restServerError(w, err)
		return
	}
	if services == nil {
		glog.V(3).Info("Running services was nil, returning empty list instead")
		services = []dao.RunningService{}
	}
	glog.V(2).Infof("Returning %d running services for service %s", len(services), serviceID)
	w.WriteJson(&services)
}

func restGetAllRunning(w *rest.ResponseWriter, r *rest.Request, client *node.ControlClient) {
	var services []dao.RunningService
	err := client.GetRunningServices(&empty, &services)
	if err != nil {
		glog.Errorf("Could not get services: %v", err)
		restServerError(w, err)
		return
	}
	if services == nil {
		glog.V(3).Info("Services was nil, returning empty list instead")
		services = []dao.RunningService{}
	}

	for _, rsvc := range services {
		var svc service.Service
		if err := client.GetService(rsvc.ServiceID, &svc); err != nil {
			glog.Errorf("Could not get services: %v", err)
			restServerError(w, err)
		}
		fillBuiltinMetrics(&svc)
		rsvc.MonitoringProfile = svc.MonitoringProfile
	}

	services = append(services, getIRS()...)
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
	var allServices []service.Service
	topServices := []service.Service{}

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
	topServices = append(topServices, isvcs.InternalServicesISVC)
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

	var allServices []service.Service

	if err := client.GetServices(&empty, &allServices); err != nil {
		glog.Errorf("Could not get services: %v", err)
		restServerError(w, err)
		return
	}

	for _, svc := range allServices {
		if svc.ID == sid {
			fillBuiltinMetrics(&svc)
			w.WriteJson(&svc)
			return
		}
	}

	glog.Errorf("No such service [%v]", sid)
	restServerError(w, err)
}

func restAddService(w *rest.ResponseWriter, r *rest.Request, client *node.ControlClient) {
	var svc service.Service
	var serviceID string
	err := r.DecodeJsonPayload(&svc)
	if err != nil {
		glog.V(1).Info("Could not decode service payload: ", err)
		restBadRequest(w, err)
		return

	}
	if id, err := utils.NewUUID36(); err != nil {
		restBadRequest(w, err)
		return
	} else {
		svc.ID = id
	}
	now := time.Now()
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

	tags := map[string][]string{
		"controlplane_service_id": []string{svc.ID},
	}
	profile, err := svc.MonitoringProfile.ReBuild("1h-ago", tags)
	if err != nil {
		glog.Errorf("Unable to rebuild service monitoring profile: %v", err)
		restServerError(w, err)
		return
	}
	svc.MonitoringProfile = *profile

	//add the service to the data store
	err = client.AddService(svc, &serviceID)
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
	filePath := ""
	err := client.AsyncBackup(dir, &filePath)
	if err != nil {
		glog.Errorf("Unexpected error during backup: %v", err)
		restServerError(w, err)
		return
	}
	w.WriteJson(&simpleResponse{filePath, servicesLinks()})
}

func RestBackupRestore(w *rest.ResponseWriter, r *rest.Request, client *node.ControlClient) {
	home := os.Getenv("SERVICED_HOME")
	if home == "" {
		glog.Infof("SERVICED_HOME not set.  Backups will save to /tmp.")
		home = "/tmp"
	}

	err := r.ParseForm()
	filePath := r.FormValue("filename")

	if err != nil || filePath == "" {
		restBadRequest(w, err)
		return
	}

	unused := 0

	err = client.AsyncRestore(home+"/backup/"+filePath, &unused)
	if err != nil {
		glog.Errorf("Unexpected error during restore: %v", err)
		restServerError(w, err)
		return
	}
	w.WriteJson(&simpleResponse{string(unused), servicesLinks()})
}

// RestBackupFileList implements a rest call that will return a list of the current backup files.
// The return value is a JSON struct of type JsonizableFileInfo.
func RestBackupFileList(w *rest.ResponseWriter, r *rest.Request, ctx *requestContext) {
	type JsonizableFileInfo struct {
		FullPath string      `json:"full_path"`
		Name     string      `json:"name"`
		Size     int64       `json:"size"`
		Mode     os.FileMode `json:"mode"`
		ModTime  time.Time   `json:"mod_time"`
	}

	fileData := []JsonizableFileInfo{}
	home := os.Getenv("SERVICED_HOME")
	if home == "" {
		glog.Infof("SERVICED_HOME not set.  Backups will save to /tmp.")
		home = "/tmp"
	}
	backupDir := home + "/backup"
	backupFiles, _ := ioutil.ReadDir(backupDir)

	hostIP, err := utils.GetIPAddress()
	if err != nil {
		glog.Errorf("Unable to get host IP: %v", err)
		restServerError(w, err)
		return
	}

	for _, backupFileInfo := range backupFiles {
		if !backupFileInfo.IsDir() {
			fullPath := hostIP + ":" + filepath.Join(backupDir, backupFileInfo.Name())
			fileInfo := JsonizableFileInfo{fullPath, backupFileInfo.Name(), backupFileInfo.Size(), backupFileInfo.Mode(), backupFileInfo.ModTime()}
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
