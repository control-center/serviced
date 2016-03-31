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
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/zenoss/glog"
	"github.com/zenoss/go-json-rest"

	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/domain"
	"github.com/control-center/serviced/domain/addressassignment"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/isvcs"
	"github.com/control-center/serviced/metrics"
	"github.com/control-center/serviced/node"
	"github.com/control-center/serviced/servicedversion"
	"github.com/control-center/serviced/utils"
	"github.com/control-center/serviced/volume"
)

var empty interface{}

type handlerFunc func(w *rest.ResponseWriter, r *rest.Request)
type handlerClientFunc func(w *rest.ResponseWriter, r *rest.Request, client *node.ControlClient)

func restDockerIsLoggedIn(w *rest.ResponseWriter, r *rest.Request, client *node.ControlClient) {
	w.WriteJson(&map[string]bool{"dockerLoggedIn": utils.DockerIsLoggedIn()})
}

func getTaggedServices(client *node.ControlClient, tags, nmregex string, tenantID string) ([]service.Service, error) {
	services := []service.Service{}
	tagsSlice := strings.Split(tags, ",")
	serviceRequest := dao.ServiceRequest{
		Tags:      tagsSlice,
		TenantID:  tenantID,
		NameRegex: nmregex,
	}
	if err := client.GetTaggedServices(serviceRequest, &services); err != nil {
		glog.Errorf("Could not get tagged services: %v", err)
		return nil, err
	}

	glog.V(2).Infof("Returning %d tagged services", len(services))
	return services, nil
}

func getNamedServices(client *node.ControlClient, nmregex string, tenantID string) ([]service.Service, error) {
	services := []service.Service{}
	var emptySlice []string
	serviceRequest := dao.ServiceRequest{
		Tags:      emptySlice,
		TenantID:  tenantID,
		NameRegex: nmregex,
	}
	if err := client.GetServices(serviceRequest, &services); err != nil {
		glog.Errorf("Could not get named services: %v", err)
		return nil, err
	}

	return services, nil
}

func getServices(client *node.ControlClient, tenantID string, since time.Duration) ([]service.Service, error) {
	services := []service.Service{}
	var emptySlice []string
	serviceRequest := dao.ServiceRequest{
		Tags:         emptySlice,
		TenantID:     tenantID,
		UpdatedSince: since,
		NameRegex:    "",
	}
	if err := client.GetServices(serviceRequest, &services); err != nil {
		glog.Errorf("Could not get services: %v", err)
		return nil, err
	}

	glog.V(2).Infof("Returning %d services", len(services))
	return services, nil
}

func getISVCS() []service.Service {
	services := []service.Service{}
	services = append(services, isvcs.InternalServicesISVC)
	services = append(services, isvcs.ElasticsearchServicedISVC)
	services = append(services, isvcs.ElasticsearchLogStashISVC)
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
	services = append(services, isvcs.ElasticsearchServicedIRS)
	services = append(services, isvcs.ElasticsearchLogStashIRS)
	services = append(services, isvcs.ZookeeperIRS)
	services = append(services, isvcs.LogstashIRS)
	services = append(services, isvcs.OpentsdbIRS)
	services = append(services, isvcs.CeleryIRS)
	services = append(services, isvcs.DockerRegistryIRS)
	return services
}

func restPostServicesForMigration(w *rest.ResponseWriter, r *rest.Request, client *node.ControlClient) {
	var migrationRequest dao.ServiceMigrationRequest
	err := r.DecodeJsonPayload(&migrationRequest)
	if err != nil {
		glog.Errorf("Could not decode services for migration: %v", err)
		restBadRequest(w, err)
		return
	}
	var unused int
	if err = client.MigrateServices(migrationRequest, &unused); err != nil {
		restServerError(w, err)
		return
	}
	w.WriteJson(&simpleResponse{"Migrated services.", []link{}})
}

func restGetAllServices(w *rest.ResponseWriter, r *rest.Request, client *node.ControlClient) {
	tenantID := r.URL.Query().Get("tenantID")
	if tags := r.URL.Query().Get("tags"); tags != "" {
		nmregex := r.URL.Query().Get("name")
		result, err := getTaggedServices(client, tags, nmregex, tenantID)
		if err != nil {
			restServerError(w, err)
			return
		}

		for ii, _ := range result {
			fillBuiltinMetrics(&result[ii])
		}
		w.WriteJson(&result)
		return
	}

	if nmregex := r.URL.Query().Get("name"); nmregex != "" {
		result, err := getNamedServices(client, nmregex, tenantID)
		if err != nil {
			restServerError(w, err)
			return
		}

		for ii, _ := range result {
			fillBuiltinMetrics(&result[ii])
		}
		w.WriteJson(&result)
		return
	}

	since := r.URL.Query().Get("since")
	var tsince time.Duration
	if since == "" {
		tsince = time.Duration(0)
	} else {
		tint, err := strconv.ParseInt(since, 10, 64)
		if err != nil {
			restServerError(w, err)
			return
		}
		tsince = time.Duration(tint) * time.Millisecond
	}
	result, err := getServices(client, tenantID, tsince)
	if err != nil {
		restServerError(w, err)
		return
	}

	if tenantID == "" { //Don't add isvcs if a tenant is specified
		if since == "" {
			result = append(result, getISVCS()...)
		} else {
			t0 := time.Now().Add(-tsince)
			for _, isvc := range getISVCS() {
				if isvc.UpdatedAt.After(t0) {
					result = append(result, isvc)
				}
			}
		}
	}

	for ii, _ := range result {
		fillBuiltinMetrics(&result[ii])
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

func restGetStatusForService(w *rest.ResponseWriter, r *rest.Request, client *node.ControlClient) {
	serviceID, err := url.QueryUnescape(r.PathParam("serviceId"))
	if strings.Contains(serviceID, "isvc-") {
		w.WriteJson([]dao.RunningService{})
		return
	}
	if err != nil {
		restBadRequest(w, err)
		return
	}
	var statusmap map[string]dao.ServiceStatus
	err = client.GetServiceStatus(serviceID, &statusmap)
	if err != nil {
		glog.Errorf("Could not get status for service %s: %v", serviceID, err)
		restServerError(w, err)
		return
	}
	if statusmap == nil {
		glog.V(3).Info("Running services was nil, returning empty map instead")
		statusmap = map[string]dao.ServiceStatus{}
	}
	glog.V(2).Infof("Returning %d states for service %s", len(statusmap), serviceID)
	w.WriteJson(&statusmap)
}

type uiRunningService struct {
	dao.RunningService
	RAMMax     int64
	RAMLast    int64
	RAMAverage int64
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

	instances := []metrics.ServiceInstance{}

	for _, rsvc := range services {
		var svc service.Service
		if err := client.GetService(rsvc.ServiceID, &svc); err != nil {
			glog.Errorf("Could not get services: %v", err)
			restServerError(w, err)
		}
		fillBuiltinMetrics(&svc)
		rsvc.MonitoringProfile = svc.MonitoringProfile
		instances = append(instances, metrics.ServiceInstance{
			rsvc.ServiceID,
			rsvc.InstanceID,
		})
	}

	query := dao.MetricRequest{
		StartTime: time.Now().Add(-time.Hour * 24),
		Instances: instances,
	}

	response := []metrics.MemoryUsageStats{}

	if len(instances) > 0 {
		client.GetInstanceMemoryStats(query, &response)
	}

	musMap := make(map[string]metrics.MemoryUsageStats)

	for _, mus := range response {
		key := fmt.Sprintf("%s-%s", mus.ServiceID, mus.InstanceID)
		musMap[key] = mus
	}

	uiServices := []uiRunningService{}

	services = append(services, getIRS()...)

	for _, rsvc := range services {
		key := fmt.Sprintf("%s-%d", rsvc.ServiceID, rsvc.InstanceID)
		if mus, ok := musMap[key]; ok {
			uiServices = append(uiServices, uiRunningService{
				rsvc,
				mus.Max,
				mus.Last,
				mus.Average,
			})
		} else {
			uiServices = append(uiServices, uiRunningService{
				rsvc,
				-1,
				-1,
				-1,
			})
		}
	}

	glog.V(2).Infof("Return %d running services", len(uiServices))
	w.WriteJson(&uiServices)
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
	var serviceRequest dao.ServiceRequest
	err := client.GetServices(serviceRequest, &allServices)
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
	serviceID, err := url.QueryUnescape(r.PathParam("serviceId"))
	if err != nil {
		restBadRequest(w, err)
		return
	}

	includeChildren := r.URL.Query().Get("includeChildren")

	if includeChildren == "true" {
		services := []service.Service{}
		if err := client.GetServiceList(serviceID, &services); err != nil {
			glog.Errorf("Could not get services: %v", err)
			restServerError(w, err)
			return
		}
		w.WriteJson(&services)
		return
	}

	if strings.Contains(serviceID, "isvc-") {
		w.WriteJson(isvcs.ISVCSMap[serviceID])
		return
	}
	svc := service.Service{}
	if err := client.GetService(serviceID, &svc); err != nil {
		glog.Errorf("Could not get service %v: %v", serviceID, err)
		restServerError(w, err)
		return
	}

	if svc.ID == serviceID {
		fillBuiltinMetrics(&svc)
		w.WriteJson(&svc)
		return
	}

	glog.Errorf("No such service [%v]", serviceID)
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

	//for each endpoint, evaluate its EndpointTemplates
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
	request := addressassignment.AssignmentRequest{ServiceID: svc.ID, IPAddress: "", AutoAssignment: true}
	if err := client.AssignIPs(request, nil); err != nil {
		glog.Errorf("Failed to automatically assign IPs: %+v -> %v", request, err)
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

// restRestartService restarts the service with the given id and all of its children
func restRestartService(w *rest.ResponseWriter, r *rest.Request, client *node.ControlClient) {
	serviceID, err := url.QueryUnescape(r.PathParam("serviceId"))
	if err != nil {
		restBadRequest(w, err)
		return
	}

	auto := r.FormValue("auto")
	autoLaunch := true

	switch auto {
	case "1", "True", "true":
		autoLaunch = true
	case "0", "False", "false":
		autoLaunch = false
	}

	var affected int
	if err := client.RestartService(dao.ScheduleServiceRequest{serviceID, autoLaunch}, &affected); err != nil {
		glog.Errorf("Unexpected error restarting service: %s", err)
		restServerError(w, err)
		return
	}
	w.WriteJson(&simpleResponse{"Restarted service", serviceLinks(serviceID)})
}

// restStartService starts the service with the given id and all of its children
func restStartService(w *rest.ResponseWriter, r *rest.Request, client *node.ControlClient) {
	serviceID, err := url.QueryUnescape(r.PathParam("serviceId"))
	if err != nil {
		restBadRequest(w, err)
		return
	}

	auto := r.FormValue("auto")
	autoLaunch := true

	switch auto {
	case "1", "True", "true":
		autoLaunch = true
	case "0", "False", "false":
		autoLaunch = false
	}

	var affected int
	if err := client.StartService(dao.ScheduleServiceRequest{serviceID, autoLaunch}, &affected); err != nil {
		glog.Errorf("Unexpected error starting service: %s", err)
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

	auto := r.FormValue("auto")
	autoLaunch := true

	switch auto {
	case "1", "True", "true":
		autoLaunch = true
	case "0", "False", "false":
		autoLaunch = false
	}

	var affected int
	if err := client.StopService(dao.ScheduleServiceRequest{serviceID, autoLaunch}, &affected); err != nil {
		glog.Errorf("Unexpected error stopping service: %s", err)
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
	req := dao.SnapshotRequest{
		ServiceID: serviceID,
	}
	var label string
	err = client.Snapshot(req, &label)
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

func downloadServiceStateLogs(w *rest.ResponseWriter, r *rest.Request, client *node.ControlClient) {
	serviceStateID, err := url.QueryUnescape(r.PathParam("serviceStateId"))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("Bad Request: %v", err)))
		return
	}
	serviceID, err := url.QueryUnescape(r.PathParam("serviceId"))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("Bad Request: %v", err)))
		return
	}

	request := dao.ServiceStateRequest{serviceID, serviceStateID}

	var logs string
	err = client.GetServiceStateLogs(request, &logs)

	if err != nil {
		glog.Errorf("Unexpected error getting service state logs: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("Internal Server Error: %v", err)))
		return
	}

	var filename = serviceID + time.Now().Format("2006-01-02-15-04-05") + ".log"
	w.Header().Set("Content-Disposition", "attachment; filename="+filename)
	w.Header().Set("Content-Type", r.Header.Get("Content-Type"))
	w.Write([]byte(logs))
}

func restGetServicedVersion(w *rest.ResponseWriter, r *rest.Request, client *node.ControlClient) {
	w.WriteJson(servicedversion.GetVersion())
}

func restGetStorage(w *rest.ResponseWriter, r *rest.Request, client *node.ControlClient) {
	volumeStatuses := volume.GetStatus()
	if volumeStatuses == nil || volumeStatuses.StatusMap == nil {
		err := fmt.Errorf("Unexpected error getting volume status")
		glog.Errorf("%s", err)
		restServerError(w, err)
		return
	}

	type VolumeInfo struct {
		Name              string
		Status            volume.Status
		MonitoringProfile domain.MonitorProfile
	}

	// REST collections should return arrays, not maps
	storageInfo := make([]VolumeInfo, 0, len(volumeStatuses.StatusMap))
	for volumeName, volumeStatus := range volumeStatuses.StatusMap {
		volumeInfo := VolumeInfo{Name: volumeName, Status: volumeStatus}
		tags := map[string][]string{}
		profile, err := volumeProfile.ReBuild("1h-ago", tags)
		if err != nil {
			glog.Errorf("Unexpected error getting volume statuses: %v", err)
			restServerError(w, err)
			return
		}
		//add graphs to profile
		profile.GraphConfigs = make([]domain.GraphConfig, 1)
		profile.GraphConfigs[0] = newVolumeUsageGraph(tags)
		volumeInfo.MonitoringProfile = *profile
		storageInfo = append(storageInfo, volumeInfo)
	}

	w.WriteJson(storageInfo)
}

func restGetUIConfig(w *rest.ResponseWriter, r *rest.Request, client *node.ControlClient) {
	w.WriteJson(uiConfig)
}

func RestBackupCreate(w *rest.ResponseWriter, r *rest.Request, client *node.ControlClient) {
	dir := ""
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
	err := r.ParseForm()
	filePath := r.FormValue("filename")

	if err != nil || filePath == "" {
		restBadRequest(w, err)
		return
	}

	unused := 0

	err = client.AsyncRestore(filePath, &unused)
	if err != nil {
		glog.Errorf("Unexpected error during restore: %v", err)
		restServerError(w, err)
		return
	}
	w.WriteJson(&simpleResponse{string(unused), servicesLinks()})
}

// RestBackupFileList implements a rest call that will return a list of the current backup files.
// The return value is a JSON struct of type JsonizableFileInfo.
func RestBackupFileList(w *rest.ResponseWriter, r *rest.Request, client *node.ControlClient) {
	var fileData []dao.BackupFile
	if err := client.ListBackups("", &fileData); err != nil {
		restServerError(w, err)
		return
	}
	w.WriteJson(&fileData)
}

func RestBackupStatus(w *rest.ResponseWriter, r *rest.Request, client *node.ControlClient) {
	backupStatus := ""
	err := client.BackupStatus(0, &backupStatus)
	if err != nil {
		glog.Errorf("Unexpected error during backup status: %v", err)
		writeJSON(w, &simpleResponse{err.Error(), homeLink()}, http.StatusInternalServerError)
		return
	}
	w.WriteJson(&simpleResponse{backupStatus, servicesLinks()})
}

func RestRestoreStatus(w *rest.ResponseWriter, r *rest.Request, client *node.ControlClient) {
	restoreStatus := ""
	err := client.BackupStatus(0, &restoreStatus)
	if err != nil {
		glog.Errorf("Unexpected error during restore status: %v", err)
		writeJSON(w, &simpleResponse{err.Error(), homeLink()}, http.StatusInternalServerError)
		return
	}
	w.WriteJson(&simpleResponse{restoreStatus, servicesLinks()})
}
