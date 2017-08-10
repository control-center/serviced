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

	"github.com/Sirupsen/logrus"
	"github.com/zenoss/go-json-rest"

	"github.com/control-center/serviced/dao"
	daoclient "github.com/control-center/serviced/dao/client"
	"github.com/control-center/serviced/domain"
	"github.com/control-center/serviced/domain/addressassignment"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/facade"
	"github.com/control-center/serviced/isvcs"
	"github.com/control-center/serviced/servicedversion"
	"github.com/control-center/serviced/utils"
	"github.com/control-center/serviced/volume"
)

var empty interface{}

var snapshotSpacePercent int

type handlerFunc func(w *rest.ResponseWriter, r *rest.Request)
type handlerClientFunc func(w *rest.ResponseWriter, r *rest.Request, client *daoclient.ControlClient)

func restDockerIsLoggedIn(w *rest.ResponseWriter, r *rest.Request, client *daoclient.ControlClient) {
	w.WriteJson(&map[string]bool{"dockerLoggedIn": utils.DockerIsLoggedIn()})
}

func getTaggedServices(ctx *requestContext, tags, nmregex string, tenantID string) ([]service.Service, error) {
	tagsSlice := strings.Split(tags, ",")
	serviceRequest := dao.ServiceRequest{
		Tags:      tagsSlice,
		TenantID:  tenantID,
		NameRegex: nmregex,
	}
	if svcs, err := ctx.getFacade().GetTaggedServices(ctx.getDatastoreContext(), serviceRequest); err == nil {
		plog.WithField("numservices", len(svcs)).Debug("Returning tagged services")
		return svcs, nil
	} else {
		return nil, err
	}
}

func getNamedServices(ctx *requestContext, nmregex string, tenantID string) ([]service.Service, error) {
	var emptySlice []string
	serviceRequest := dao.ServiceRequest{
		Tags:      emptySlice,
		TenantID:  tenantID,
		NameRegex: nmregex,
	}
	if svcs, err := ctx.getFacade().GetServices(ctx.getDatastoreContext(), serviceRequest); err == nil {
		plog.WithField("numservices", len(svcs)).Debug("Returning named services")
		return svcs, nil
	} else {
		return nil, err
	}
}

func getServices(ctx *requestContext, tenantID string, since time.Duration) ([]service.Service, error) {
	var emptySlice []string
	serviceRequest := dao.ServiceRequest{
		Tags:         emptySlice,
		TenantID:     tenantID,
		UpdatedSince: since,
		NameRegex:    "",
	}
	if svcs, err := ctx.getFacade().GetServices(ctx.getDatastoreContext(), serviceRequest); err == nil {
		plog.WithField("numservices", len(svcs)).Debug("Returning services")
		return svcs, nil
	} else {
		return nil, err
	}
}

func getISVCS() []service.Service {
	services := []service.Service{}
	services = append(services, isvcs.InternalServicesISVC)
	services = append(services, isvcs.ElasticsearchServicedISVC)
	services = append(services, isvcs.ElasticsearchLogStashISVC)
	services = append(services, isvcs.ZookeeperISVC)
	services = append(services, isvcs.LogstashISVC)
	services = append(services, isvcs.OpentsdbISVC)
	services = append(services, isvcs.DockerRegistryISVC)
	services = append(services, isvcs.KibanaISVC)
	return services
}

func getIRS() []dao.RunningService {
	services := []dao.RunningService{}
	services = append(services, isvcs.InternalServicesIRS)
	services = append(services, isvcs.ElasticsearchServicedIRS)
	services = append(services, isvcs.ElasticsearchLogStashIRS)
	services = append(services, isvcs.GetZookeeperInstances()...)
	services = append(services, isvcs.LogstashIRS)
	services = append(services, isvcs.OpentsdbIRS)
	services = append(services, isvcs.DockerRegistryIRS)
	services = append(services, isvcs.KibanaIRS)
	return services
}

func restPostServicesForMigration(w *rest.ResponseWriter, r *rest.Request, client *daoclient.ControlClient) {
	var migrationRequest dao.ServiceMigrationRequest
	err := r.DecodeJsonPayload(&migrationRequest)
	if err != nil {
		plog.WithError(err).Error("Could not decode services for migration")
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

// DEPRECATED - This call is SUPER expensive at sites with 1000s of services.
//              As of 1.2.0, the UI no longer uses this endpoint, but Zenoss and/or
//              the CC ZenPack may.
// FIXME: Delete this method as soon as Zenoss and CC ZenPack no longer use this method.
func restGetAllServices(w *rest.ResponseWriter, r *rest.Request, ctx *requestContext) {

	tenantID := r.URL.Query().Get("tenantID")

	// load the internal monitoring data
	config, err := getInternalMetrics()
	if err != nil {
		plog.WithError(err).Error("Could not get internal monitoring metrics")
		restServerError(w, err)
		return
	}

	if tags := r.URL.Query().Get("tags"); tags != "" {
		nmregex := r.URL.Query().Get("name")
		result, err := getTaggedServices(ctx, tags, nmregex, tenantID)
		if err != nil {
			restServerError(w, err)
			return
		}

		for ii, svc := range result {
			if len(svc.Startup) > 2 {
				result[ii].MonitoringProfile.MetricConfigs = append(result[ii].MonitoringProfile.MetricConfigs, *config)
				result[ii].MonitoringProfile.GraphConfigs = append(result[ii].MonitoringProfile.GraphConfigs, getInternalGraphConfigs(result[ii].ID)...)
			}
		}
		w.WriteJson(&result)
		return
	}

	if nmregex := r.URL.Query().Get("name"); nmregex != "" {
		result, err := getNamedServices(ctx, nmregex, tenantID)
		if err != nil {
			restServerError(w, err)
			return
		}

		for ii, svc := range result {
			if len(svc.Startup) > 2 {
				result[ii].MonitoringProfile.MetricConfigs = append(result[ii].MonitoringProfile.MetricConfigs, *config)
				result[ii].MonitoringProfile.GraphConfigs = append(result[ii].MonitoringProfile.GraphConfigs, getInternalGraphConfigs(result[ii].ID)...)
			}
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
	result, err := getServices(ctx, tenantID, tsince)
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

	for ii, svc := range result {
		if strings.HasPrefix(result[ii].ID, "isvc-") {
			continue
		}
		if len(svc.Startup) > 2 {
			result[ii].MonitoringProfile.MetricConfigs = append(result[ii].MonitoringProfile.MetricConfigs, *config)
			result[ii].MonitoringProfile.GraphConfigs = append(result[ii].MonitoringProfile.GraphConfigs, getInternalGraphConfigs(result[ii].ID)...)
		}
	}
	w.WriteJson(&result)
}

func restGetRunningForHost(w *rest.ResponseWriter, r *rest.Request, client *daoclient.ControlClient) {
	hostID, err := url.QueryUnescape(r.PathParam("hostId"))
	if err != nil {
		restBadRequest(w, err)
		return
	}
	var services []dao.RunningService
	err = client.GetRunningServicesForHost(hostID, &services)
	if err != nil {
		plog.WithError(err).Error("Could not get services")
		restServerError(w, err)
		return
	}
	if services == nil {
		plog.Debug("Running services was nil")
		services = []dao.RunningService{}
	}
	plog.WithFields(logrus.Fields{
		"numservices": len(services),
		"hostid":      hostID,
	}).Debug("Got running services for host")
	w.WriteJson(&services)
}

func restGetRunningForService(w *rest.ResponseWriter, r *rest.Request, client *daoclient.ControlClient) {
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
		plog.WithError(err).WithField("serviceid", serviceID).Error("Could not get running services")
		restServerError(w, err)
		return
	}
	if services == nil {
		plog.Debug("Running services was nil")
		services = []dao.RunningService{}
	}
	plog.WithFields(logrus.Fields{
		"numservices": len(services),
		"serviceid":   serviceID,
	}).Debug("Got running services for service")
	w.WriteJson(&services)
}

type uiRunningService struct {
	dao.RunningService
	RAMMax     int64
	RAMLast    int64
	RAMAverage int64
}

func restKillRunning(w *rest.ResponseWriter, r *rest.Request, client *daoclient.ControlClient) {
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
	logger := plog.WithFields(logrus.Fields{
		"hostid":         hostID,
		"servicestateid": serviceStateID,
	})
	request := dao.HostServiceRequest{hostID, serviceStateID}
	logger.WithField("request", request).Debug("Received request to kill")

	var unused int
	err = client.StopRunningInstance(request, &unused)
	if err != nil {
		logger.WithError(err).Error("Unable to stop service")
		restServerError(w, err)
		return
	}

	w.WriteJson(&simpleResponse{"Marked for death", servicesLinks()})
}

// DEPRECATED - As of 1.2.0, the UI no longer uses this endpoint, but Zenoss and/or
//              the CC ZenPack may.
// FIXME: Delete this method as soon as Zenoss and CC ZenPack no longer use this method.
func restGetTopServices(w *rest.ResponseWriter, r *rest.Request, ctx *requestContext) {
	topServices := []service.Service{}

	tsince, err := getSinceParameter(r)
	if err != nil {
		restServerError(w, err)
		return
	}

	// Instead of getting all services, get ServiceDetails for just the tenant Apps
	allTenants, err := ctx.getFacade().GetServiceDetailsByParentID(ctx.getDatastoreContext(), "", tsince)
	if err != nil {
		plog.WithError(err).Error("Could not get services")
		restServerError(w, err)
		return
	}
	for _, tenant := range allTenants {
		service, err := ctx.getFacade().GetService(ctx.getDatastoreContext(), tenant.ID)
		if err != nil {
			plog.WithField("tenantid", tenant.ID).WithError(err).Error("Could not get service")
			restServerError(w, err)
			return
		}
		topServices = append(topServices, *service)
	}
	topServices = append(topServices, isvcs.InternalServicesISVC)
	plog.WithField("numservices", len(topServices)).Debug("Got top services")
	w.WriteJson(&topServices)
}

// DEPRECATED
func restGetService(w *rest.ResponseWriter, r *rest.Request, client *daoclient.ControlClient) {

	serviceID, err := url.QueryUnescape(r.PathParam("serviceId"))
	if err != nil {
		restBadRequest(w, err)
		return
	}

	// load the internal monitoring data
	config, err := getInternalMetrics()
	if err != nil {
		plog.WithError(err).Error("Could not get internal monitoring metrics")
		restServerError(w, err)
		return
	}

	includeChildren := r.URL.Query().Get("includeChildren")

	if includeChildren == "true" {
		services := []service.Service{}
		if err := client.GetServiceList(serviceID, &services); err != nil {
			plog.WithError(err).Error("Could not get services")
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
		plog.WithField("serviceid", serviceID).WithError(err).Error("Could not get service")
		restServerError(w, err)
		return
	}

	if svc.ID == serviceID {
		svc.MonitoringProfile.MetricConfigs = append(svc.MonitoringProfile.MetricConfigs, *config)
		svc.MonitoringProfile.GraphConfigs = append(svc.MonitoringProfile.GraphConfigs, getInternalGraphConfigs(svc.ID)...)
		w.WriteJson(&svc)
		return
	}

	plog.WithField("serviceid", serviceID).Error("No such service")
	restServerError(w, err)
}

func restAddService(w *rest.ResponseWriter, r *rest.Request, client *daoclient.ControlClient) {
	var svc service.Service
	var serviceID string
	err := r.DecodeJsonPayload(&svc)
	if err != nil {
		plog.WithError(err).Debug("Could not decode service payload")
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
	if err = svc.EvaluateEndpointTemplates(getSvc, findChild, 0); err != nil {
		plog.WithError(err).Error("Unable to evaluate service endpoints")
		restServerError(w, err)
		return
	}

	tags := map[string][]string{
		"controlplane_service_id": []string{svc.ID},
	}
	profile, err := svc.MonitoringProfile.ReBuild("1h-ago", tags)
	if err != nil {
		plog.WithError(err).Error("Unable to rebuild service monitoring profile")
		restServerError(w, err)
		return
	}
	svc.MonitoringProfile = *profile

	//add the service to the data store
	err = client.AddService(svc, &serviceID)
	if err != nil {
		plog.WithError(err).Error("Unable to add service")
		restServerError(w, err)
		return
	}

	//automatically assign virtual ips to new service
	request := addressassignment.AssignmentRequest{ServiceID: svc.ID, IPAddress: "", AutoAssignment: true}
	if err := client.AssignIPs(request, nil); err != nil {
		plog.WithField("request", request).WithError(err).Error("Failed to automatically assign IPs")
		restServerError(w, err)
		return
	}

	plog.WithField("serviceid", serviceID).Info("Added service")
	w.WriteJson(&simpleResponse{"Added service", serviceLinks(serviceID)})
}

func restDeployService(w *rest.ResponseWriter, r *rest.Request, client *daoclient.ControlClient) {
	var payload dao.ServiceDeploymentRequest
	err := r.DecodeJsonPayload(&payload)
	if err != nil {
		plog.WithError(err).Debug("Could not decode service payload")
		restBadRequest(w, err)
		return
	}

	var serviceID string
	err = client.DeployService(payload, &serviceID)
	if err != nil {
		plog.WithError(err).Error("Unable to deploy service")
		restServerError(w, err)
		return
	}

	plog.WithField("serviceid", serviceID).Info("Deployed service")
	w.WriteJson(&simpleResponse{"Deployed service", serviceLinks(serviceID)})
}

func restUpdateService(w *rest.ResponseWriter, r *rest.Request, client *daoclient.ControlClient) {
	serviceID, err := url.QueryUnescape(r.PathParam("serviceId"))
	logger := plog.WithField("serviceid", serviceID)
	logger.Debug("Received update request")
	if err != nil {
		restBadRequest(w, err)
		return
	}
	var payload service.Service
	var unused int
	err = r.DecodeJsonPayload(&payload)
	if err != nil {
		logger.WithError(err).Debug("Could not decode service payload")
		restBadRequest(w, err)
		return
	}
	err = client.UpdateService(payload, &unused)
	if err != nil {
		logger.WithField("serviceid", serviceID).WithError(err).Error("Unable to update service")
		restServerError(w, err)
		return
	}
	logger.Debug("Updated service")
	w.WriteJson(&simpleResponse{"Updated service", serviceLinks(serviceID)})
}

func restRemoveService(w *rest.ResponseWriter, r *rest.Request, ctx *requestContext) {
	serviceID, err := url.QueryUnescape(r.PathParam("serviceId"))
	if err != nil {
		restBadRequest(w, err)
		return
	}
	logger := plog.WithField("serviceid", serviceID)
	serviceFacade := ctx.getFacade()
	dataCtx := ctx.getDatastoreContext()
	err = serviceFacade.RemoveService(dataCtx, serviceID)
	if err != nil {
		logger.WithError(err).Error("Could not remove service")
		restServerError(w, err)
		return
	}
	logger.Info("Removed service")
	w.WriteJson(&simpleResponse{"Removed service", servicesLinks()})
}

func restGetServiceLogs(w *rest.ResponseWriter, r *rest.Request, client *daoclient.ControlClient) {
	serviceID, err := url.QueryUnescape(r.PathParam("serviceId"))
	if err != nil {
		restBadRequest(w, err)
		return
	}
	logger := plog.WithField("serviceid", serviceID)
	var logs string
	err = client.GetServiceLogs(serviceID, &logs)
	if err != nil {
		logger.WithError(err).Error("Unexpected error getting service logs")
		restServerError(w, err)
		return
	}
	w.WriteJson(&simpleResponse{logs, serviceLinks(serviceID)})
}

// restRestartService restarts the service with the given id and all of its children
func restRestartService(w *rest.ResponseWriter, r *rest.Request, ctx *requestContext) {
	serviceID, err := url.QueryUnescape(r.PathParam("serviceId"))
	if err != nil {
		restBadRequest(w, err)
		return
	}

	logger := plog.WithField("serviceid", serviceID)

	auto := r.FormValue("auto")
	autoLaunch := true

	switch auto {
	case "1", "True", "true":
		autoLaunch = true
	case "0", "False", "false":
		autoLaunch = false
	}

	serviceFacade := ctx.getFacade()
	dataCtx := ctx.getDatastoreContext()

	_, err = serviceFacade.RestartService( dataCtx, dao.ScheduleServiceRequest{
		ServiceIDs:  []string{serviceID},
		AutoLaunch:  autoLaunch,
		Synchronous: false,
	})
	// We handle this error differently because we don't want to return a 500
	if err == facade.ErrEmergencyShutdownNoOp {
		logger.WithError(err).Error("Error restarting service")
		writeJSON(w, &simpleResponse{err.Error(), homeLink()}, http.StatusServiceUnavailable)
		return
	} else if err != nil {
		logger.WithError(err).Error("Error restarting service")
		restServerError(w, err)
		return
	}
	w.WriteJson(&simpleResponse{"Restarted service", serviceLinks(serviceID)})
}

// restRetartServices starts the services with the given ids and all of their children
func restRestartServices(w *rest.ResponseWriter, r *rest.Request, ctx *requestContext) {

	var serviceRequest dao.ScheduleServiceRequest
	err := r.DecodeJsonPayload(&serviceRequest)
	if err != nil {
		plog.WithError(err).Debug("Could not decode service payload")
		restBadRequest(w, err)
		return

	}

	logger := plog.WithField("serviceids", serviceRequest.ServiceIDs)
	serviceFacade := ctx.getFacade()
	dataCtx := ctx.getDatastoreContext()
	_, err = serviceFacade.RestartService(dataCtx, serviceRequest)
	// We handle this error differently because we don't want to return a 500
	if err == facade.ErrEmergencyShutdownNoOp {
		logger.WithError(err).Error("Error restarting services")
		writeJSON(w, &simpleResponse{err.Error(), homeLink()}, http.StatusServiceUnavailable)
		return
	} else if err != nil {
		logger.WithError(err).Error("Error restarting services")
		restServerError(w, err)
		return
	}
	w.WriteJson(&simpleResponse{"restarted services", servicesLinks()})
}

// restStartService starts the service with the given id and all of its children
func restStartService(w *rest.ResponseWriter, r *rest.Request, ctx *requestContext) {
	serviceID, err := url.QueryUnescape(r.PathParam("serviceId"))
	if err != nil {
		restBadRequest(w, err)
		return
	}

	logger := plog.WithField("serviceid", serviceID)

	auto := r.FormValue("auto")
	autoLaunch := true

	switch auto {
	case "1", "True", "true":
		autoLaunch = true
	case "0", "False", "false":
		autoLaunch = false
	}

	serviceFacade := ctx.getFacade()
	dataCtx := ctx.getDatastoreContext()
	_,err = serviceFacade.StartService(dataCtx,dao.ScheduleServiceRequest{
		ServiceIDs:  []string{serviceID},
		AutoLaunch:  autoLaunch,
		Synchronous: false,
	})
	// We handle this error differently because we don't want to return a 500
	if err == facade.ErrEmergencyShutdownNoOp {
		logger.WithError(err).Error("Error starting service")
		writeJSON(w, &simpleResponse{err.Error(), homeLink()}, http.StatusServiceUnavailable)
		return
	} else if err != nil {
		logger.WithError(err).Error("Error starting service")
		restServerError(w, err)
		return
	}
	w.WriteJson(&simpleResponse{"Started service", serviceLinks(serviceID)})
}

// restStartServices starts the services with the given ids and all of their children
func restStartServices(w *rest.ResponseWriter, r *rest.Request, ctx  *requestContext) {

	var serviceRequest dao.ScheduleServiceRequest
	err := r.DecodeJsonPayload(&serviceRequest)
	if err != nil {
		plog.WithError(err).Debug("Could not decode service payload")
		restBadRequest(w, err)
		return

	}

	logger := plog.WithField("serviceids", serviceRequest.ServiceIDs)
	serviceFacade := ctx.getFacade()
	dataCtx := ctx.getDatastoreContext()
	_,err = serviceFacade.StartService(dataCtx, serviceRequest)
	// We handle this error differently because we don't want to return a 500
	if err == facade.ErrEmergencyShutdownNoOp {
		logger.WithError(err).Error("Error starting services")
		writeJSON(w, &simpleResponse{err.Error(), homeLink()}, http.StatusServiceUnavailable)
		return
	} else if err != nil {
		logger.WithError(err).Error("Error starting services")
		restServerError(w, err)
		return
	}
	w.WriteJson(&simpleResponse{"Started services", servicesLinks()})
}

// restStopService stop the service with the given id and all of its children
func restStopService(w *rest.ResponseWriter, r *rest.Request, ctx *requestContext) {
	serviceID, err := url.QueryUnescape(r.PathParam("serviceId"))
	if err != nil {
		restBadRequest(w, err)
		return
	}

	logger := plog.WithField("serviceid", serviceID)

	auto := r.FormValue("auto")
	autoLaunch := true

	switch auto {
	case "1", "True", "true":
		autoLaunch = true
	case "0", "False", "false":
		autoLaunch = false
	}

	serviceFacade := ctx.getFacade()
	dataCtx := ctx.getDatastoreContext()
	_,err = serviceFacade.StopService(dataCtx, dao.ScheduleServiceRequest{[]string{serviceID}, autoLaunch, false})
	if err != nil {
		logger.WithError(err).Error("Unexpected error stopping service")
		restServerError(w, err)
		return
	}
	w.WriteJson(&simpleResponse{"Stopped service", serviceLinks(serviceID)})
}

// restStopServices stops the services with the given ids and all of their children
func restStopServices(w *rest.ResponseWriter, r *rest.Request, ctx *requestContext) {

	var serviceRequest dao.ScheduleServiceRequest
	err := r.DecodeJsonPayload(&serviceRequest)
	if err != nil {
		plog.WithError(err).Debug("Could not decode service payload")
		restBadRequest(w, err)
		return

	}

	logger := plog.WithField("serviceids", serviceRequest.ServiceIDs)

	serviceFacade := ctx.getFacade()
	dataCtx := ctx.getDatastoreContext()
	_,err = serviceFacade.StopService(dataCtx,serviceRequest)
	// We handle this error differently because we don't want to return a 500
	if err != nil {
		logger.WithError(err).Error("Error stopping services")
		restServerError(w, err)
		return
	}
	w.WriteJson(&simpleResponse{"Stopped services", servicesLinks()})
}

func restSnapshotService(w *rest.ResponseWriter, r *rest.Request, client *daoclient.ControlClient) {
	serviceID, err := url.QueryUnescape(r.PathParam("serviceId"))
	if err != nil {
		restBadRequest(w, err)
		return
	}

	logger := plog.WithField("serviceid", serviceID)

	req := dao.SnapshotRequest{
		ServiceID: serviceID,
	}
	var label string
	err = client.Snapshot(req, &label)
	if err != nil {
		logger.WithError(err).Error("Unexpected error snapshotting service")
		restServerError(w, err)
		return
	}
	w.WriteJson(&simpleResponse{label, serviceLinks(serviceID)})
}

func restGetServiceStateLogs(w *rest.ResponseWriter, r *rest.Request, client *daoclient.ControlClient) {
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

	logger := plog.WithFields(logrus.Fields{
		"servicestateid": serviceStateID,
		"serviceid":      serviceID,
	})

	request := dao.ServiceStateRequest{serviceID, serviceStateID}

	var logs string
	err = client.GetServiceStateLogs(request, &logs)
	if err != nil {
		logger.WithError(err).Error("Unexpected error getting service state logs")
		restServerError(w, err)
		return
	}
	w.WriteJson(&simpleResponse{logs, servicesLinks()})
}

func downloadServiceStateLogs(w *rest.ResponseWriter, r *rest.Request, client *daoclient.ControlClient) {
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

	logger := plog.WithFields(logrus.Fields{
		"servicestateid": serviceStateID,
		"serviceid":      serviceID,
	})

	request := dao.ServiceStateRequest{serviceID, serviceStateID}

	var logs string
	err = client.GetServiceStateLogs(request, &logs)

	if err != nil {
		logger.WithError(err).Error("Unexpected error getting service state logs")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("Internal Server Error: %v", err)))
		return
	}

	var filename = serviceID + time.Now().Format("2006-01-02-15-04-05") + ".log"
	w.Header().Set("Content-Disposition", "attachment; filename="+filename)
	w.Header().Set("Content-Type", r.Header.Get("Content-Type"))
	w.Write([]byte(logs))
}

func restGetServicedVersion(w *rest.ResponseWriter, r *rest.Request, client *daoclient.ControlClient) {
	w.WriteJson(servicedversion.GetVersion())
}

func restGetStorage(w *rest.ResponseWriter, r *rest.Request, client *daoclient.ControlClient) {
	volumeStatuses := volume.GetStatus()
	if volumeStatuses == nil || len(volumeStatuses.GetAllStatuses()) == 0 {
		err := fmt.Errorf("Unexpected error getting volume status")
		plog.WithError(err).Error("Could not get volume status")
		restServerError(w, err)
		return
	}

	type VolumeInfo struct {
		Name              string
		Status            volume.Status
		MonitoringProfile domain.MonitorProfile
	}

	// REST collections should return arrays, not maps
	statuses := volumeStatuses.GetAllStatuses()
	storageInfo := make([]VolumeInfo, 0, len(statuses))
	for volumeName, volumeStatus := range statuses {
		vLogger := plog.WithFields(logrus.Fields{
			"volumename": volumeName,
		})
		volumeInfo := VolumeInfo{Name: volumeName, Status: volumeStatus}
		tags := map[string][]string{}
		profile, err := volumeProfile.ReBuild("1h-ago", tags)
		if err != nil {
			vLogger.WithError(err).Error("Unexpected error getting volume statuses")
			restServerError(w, err)
			return
		}
		//add graphs to profile
		profile.GraphConfigs = []domain.GraphConfig{
			newThinPoolDataUsageGraph(tags),
			newThinPoolMetadataUsageGraph(tags),
		}

		volumeInfo.MonitoringProfile = *profile
		storageInfo = append(storageInfo, volumeInfo)
	}

	w.WriteJson(storageInfo)
}

func restGetUIConfig(w *rest.ResponseWriter, r *rest.Request, client *daoclient.ControlClient) {
	w.WriteJson(uiConfig)
}

func RestBackupCheck(w *rest.ResponseWriter, r *rest.Request, client *daoclient.ControlClient) {
	dir := ""
	req := dao.BackupRequest{
		Dirpath:              dir,
		SnapshotSpacePercent: snapshotSpacePercent,
	}
	var estimate dao.BackupEstimate
	err := client.GetBackupEstimate(req, &estimate)
	if err != nil {
		plog.WithError(err).Error("Unexpected error during backup estimate")
		restServerError(w, err)
		return
	}
	w.WriteJson(&estimate)
}

func RestBackupCreate(w *rest.ResponseWriter, r *rest.Request, client *daoclient.ControlClient) {
	dir := ""
	filePath := ""
	username, getUserErr := getUser(r)
	if getUserErr != nil {
		plog.WithError(getUserErr).Error("Unable to get user name")
	}
	req := dao.BackupRequest{
		Dirpath:              dir,
		SnapshotSpacePercent: snapshotSpacePercent,
		Username:             username,
	}
	err := client.AsyncBackup(req, &filePath)
	if err != nil {
		plog.WithError(err).Error("Unexpected error during backup")
		restServerError(w, err)
		return
	}
	w.WriteJson(&simpleResponse{filePath, servicesLinks()})
}

func RestBackupRestore(w *rest.ResponseWriter, r *rest.Request, client *daoclient.ControlClient) {
	err := r.ParseForm()
	filePath := r.FormValue("filename")

	if err != nil || filePath == "" {
		restBadRequest(w, err)
		return
	}

	unused := 0
	username, getUserErr := getUser(r)
	if getUserErr != nil {
		plog.WithError(getUserErr).Error("Unable to get user name")
	}
	req := dao.RestoreRequest{
		Filename: filePath,
		Username: username,
	}
	err = client.AsyncRestore(req, &unused)
	if err != nil {
		plog.WithError(err).Error("Unexpected error during restore")
		restServerError(w, err)
		return
	}
	w.WriteJson(&simpleResponse{string(unused), servicesLinks()})
}

// RestBackupFileList implements a rest call that will return a list of the current backup files.
// The return value is a JSON struct of type JsonizableFileInfo.
func RestBackupFileList(w *rest.ResponseWriter, r *rest.Request, client *daoclient.ControlClient) {
	var fileData []dao.BackupFile
	if err := client.ListBackups("", &fileData); err != nil {
		restServerError(w, err)
		return
	}
	w.WriteJson(&fileData)
}

func RestBackupStatus(w *rest.ResponseWriter, r *rest.Request, client *daoclient.ControlClient) {
	backupStatus := ""
	err := client.BackupStatus(0, &backupStatus)
	if err != nil {
		plog.WithError(err).Error("Unexpected error getting backup status")
		writeJSON(w, &simpleResponse{err.Error(), homeLink()}, http.StatusInternalServerError)
		return
	}
	w.WriteJson(&simpleResponse{backupStatus, servicesLinks()})
}

func RestRestoreStatus(w *rest.ResponseWriter, r *rest.Request, client *daoclient.ControlClient) {
	restoreStatus := ""
	err := client.BackupStatus(0, &restoreStatus)
	if err != nil {
		plog.WithError(err).Error("Unexpected error getting restore status")
		writeJSON(w, &simpleResponse{err.Error(), homeLink()}, http.StatusInternalServerError)
		return
	}
	w.WriteJson(&simpleResponse{restoreStatus, servicesLinks()})
}
