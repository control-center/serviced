package web

import (
	"github.com/zenoss/glog"
	"github.com/zenoss/go-json-rest"
	"github.com/zenoss/serviced"
	"github.com/zenoss/serviced/dao"
	"github.com/zenoss/serviced/domain/service"
	"github.com/zenoss/serviced/domain/servicetemplate"
	"github.com/zenoss/serviced/servicedversion"
	"net/url"
	"regexp"
	"strings"
	"time"
	"os"
	"io/ioutil"
//	"encoding/json"
)

var empty interface{}

type HandlerFunc func(w *rest.ResponseWriter, r *rest.Request)
type HandlerClientFunc func(w *rest.ResponseWriter, r *rest.Request, client *serviced.ControlClient)

func RestGetAppTemplates(w *rest.ResponseWriter, r *rest.Request, client *serviced.ControlClient) {
	var unused int
	var templatesMap map[string]*servicetemplate.ServiceTemplate
	client.GetServiceTemplates(unused, &templatesMap)
	w.WriteJson(&templatesMap)
}

func RestDeployAppTemplate(w *rest.ResponseWriter, r *rest.Request, client *serviced.ControlClient) {
	var payload dao.ServiceTemplateDeploymentRequest
	err := r.DecodeJsonPayload(&payload)
	if err != nil {
		glog.V(1).Info("Could not decode deployment payload: ", err)
		RestBadRequest(w)
		return
	}
	var tenantId string
	err = client.DeployTemplate(payload, &tenantId)
	if err != nil {
		glog.Error("Could not deploy template: ", err)
		RestServerError(w)
		return
	}
	glog.V(0).Info("Deployed template ", payload)

	// TODO: the UI needs a way to disable that automatic IP assignment (see CmdDeployTemplate)
	assignmentRequest := dao.AssignmentRequest{tenantId, "", true}
	if err := client.AssignIPs(assignmentRequest, nil); err != nil {
		glog.Error("Could not automatically assign IPs: %v", err)
		return
	}

	glog.Infof("Automatically assigned IP addresses to service: %v", tenantId)
	// end of automatic IP assignment

	w.WriteJson(&SimpleResponse{tenantId, servicesLinks()})
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

func getTaggedServices(client *serviced.ControlClient, tags, nmregex string) ([]*service.Service, error) {
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

func getNamedServices(client *serviced.ControlClient, nmregex string) ([]*service.Service, error) {
	services := []*service.Service{}
	if err := client.GetServices(&empty, &services); err != nil {
		glog.Errorf("Could not get named services: %v", err)
		return nil, err
	}

	return filterByNameRegex(nmregex, services)
}

func getServices(client *serviced.ControlClient) ([]*service.Service, error) {
	services := []*service.Service{}
	if err := client.GetServices(&empty, &services); err != nil {
		glog.Errorf("Could not get services: %v", err)
		return nil, err
	}

	glog.V(2).Infof("Returning %d services", len(services))
	return services, nil
}

func RestGetAllServices(w *rest.ResponseWriter, r *rest.Request, client *serviced.ControlClient) {
	if tags := r.URL.Query().Get("tags"); tags != "" {
		nmregex := r.URL.Query().Get("name")
		result, err := getTaggedServices(client, tags, nmregex)
		if err != nil {
			RestServerError(w)
			return
		}

		w.WriteJson(&result)
		return
	}

	if nmregex := r.URL.Query().Get("name"); nmregex != "" {
		result, err := getNamedServices(client, nmregex)
		if err != nil {
			RestServerError(w)
			return
		}

		w.WriteJson(&result)
		return
	}

	result, err := getServices(client)
	if err != nil {
		RestServerError(w)
		return
	}

	w.WriteJson(&result)
}

func RestGetRunningForHost(w *rest.ResponseWriter, r *rest.Request, client *serviced.ControlClient) {
	hostId, err := url.QueryUnescape(r.PathParam("hostId"))
	if err != nil {
		RestBadRequest(w)
		return
	}
	var services []*dao.RunningService
	err = client.GetRunningServicesForHost(hostId, &services)
	if err != nil {
		glog.Errorf("Could not get services: %v", err)
		RestServerError(w)
		return
	}
	if services == nil {
		glog.V(3).Info("Running services was nil, returning empty list instead")
		services = []*dao.RunningService{}
	}
	glog.V(2).Infof("Returning %d running services for host %s", len(services), hostId)
	w.WriteJson(&services)
}

func RestGetRunningForService(w *rest.ResponseWriter, r *rest.Request, client *serviced.ControlClient) {
	serviceId, err := url.QueryUnescape(r.PathParam("serviceId"))
	if err != nil {
		RestBadRequest(w)
		return
	}
	var services []*dao.RunningService
	err = client.GetRunningServicesForService(serviceId, &services)
	if err != nil {
		glog.Errorf("Could not get services: %v", err)
		RestServerError(w)
		return
	}
	if services == nil {
		glog.V(3).Info("Running services was nil, returning empty list instead")
		services = []*dao.RunningService{}
	}
	glog.V(2).Infof("Returning %d running services for service %s", len(services), serviceId)
	w.WriteJson(&services)
}

func RestGetAllRunning(w *rest.ResponseWriter, r *rest.Request, client *serviced.ControlClient) {
	var services []*dao.RunningService
	err := client.GetRunningServices(&empty, &services)
	if err != nil {
		glog.Errorf("Could not get services: %v", err)
		RestServerError(w)
		return
	}
	if services == nil {
		glog.V(3).Info("Services was nil, returning empty list instead")
		services = []*dao.RunningService{}
	}
	glog.V(2).Infof("Return %d running services", len(services))
	w.WriteJson(&services)
}

func RestKillRunning(w *rest.ResponseWriter, r *rest.Request, client *serviced.ControlClient) {
	serviceStateId, err := url.QueryUnescape(r.PathParam("serviceStateId"))
	if err != nil {
		RestBadRequest(w)
		return
	}
	hostId, err := url.QueryUnescape(r.PathParam("hostId"))
	if err != nil {
		RestBadRequest(w)
		return
	}
	request := dao.HostServiceRequest{hostId, serviceStateId}
	glog.V(1).Info("Received request to kill ", request)

	var unused int
	err = client.StopRunningInstance(request, &unused)
	if err != nil {
		glog.Errorf("Unable to stop service: %v", err)
		RestServerError(w)
		return
	}

	w.WriteJson(&SimpleResponse{"Marked for death", servicesLinks()})
}

func RestGetTopServices(w *rest.ResponseWriter, r *rest.Request, client *serviced.ControlClient) {
	var allServices []*service.Service
	topServices := []*service.Service{}

	err := client.GetServices(&empty, &allServices)
	if err != nil {
		glog.Errorf("Could not get services: %v", err)
		RestServerError(w)
		return
	}
	for _, service := range allServices {
		if len(service.ParentServiceId) == 0 {
			topServices = append(topServices, service)
		}
	}
	glog.V(2).Infof("Returning %d services as top services", len(topServices))
	w.WriteJson(&topServices)
}

func RestGetService(w *rest.ResponseWriter, r *rest.Request, client *serviced.ControlClient) {
	var allServices []*service.Service

	if err := client.GetServices(&empty, &allServices); err != nil {
		glog.Errorf("Could not get services: %v", err)
		RestServerError(w)
		return
	}

	sid, err := url.QueryUnescape(r.PathParam("serviceId"))
	if err != nil {
		RestBadRequest(w)
		return
	}

	for _, service := range allServices {
		if service.Id == sid {
			w.WriteJson(&service)
			return
		}
	}

	glog.Errorf("No such service [%v]", sid)
	RestServerError(w)
}

func RestAddService(w *rest.ResponseWriter, r *rest.Request, client *serviced.ControlClient) {
	var payload service.Service
	var serviceId string
	err := r.DecodeJsonPayload(&payload)
	if err != nil {
		glog.V(1).Info("Could not decode service payload: ", err)
		RestBadRequest(w)
		return
	}
	svc, err := service.NewService()
	if err != nil {
		glog.Errorf("Could not create service: %v", err)
		RestServerError(w)
		return
	}
	now := time.Now()
	svc.Name = payload.Name
	svc.Description = payload.Description
	svc.Context = payload.Context
	svc.Tags = payload.Tags
	svc.PoolId = payload.PoolId
	svc.ImageId = payload.ImageId
	svc.Startup = payload.Startup
	svc.Instances = payload.Instances
	svc.ParentServiceId = payload.ParentServiceId
	svc.DesiredState = payload.DesiredState
	svc.Launch = payload.Launch
	svc.Endpoints = payload.Endpoints
	svc.ConfigFiles = payload.ConfigFiles
	svc.Volumes = payload.Volumes
	svc.CreatedAt = now
	svc.UpdatedAt = now

	//for each endpoint, evaluate it's Application
	if err = svc.EvaluateEndpointTemplates(client); err != nil {
		glog.Errorf("Unable to evaluate service endpoints: %v", err)
		RestServerError(w)
		return
	}

	//add the service to the data store
	err = client.AddService(*svc, &serviceId)
	if err != nil {
		glog.Errorf("Unable to add service: %v", err)
		RestServerError(w)
		return
	}

	glog.V(0).Info("Added service ", serviceId)
	w.WriteJson(&SimpleResponse{"Added service", serviceLinks(serviceId)})
}

func RestUpdateService(w *rest.ResponseWriter, r *rest.Request, client *serviced.ControlClient) {
	serviceId, err := url.QueryUnescape(r.PathParam("serviceId"))
	glog.V(3).Infof("Received update request for %s", serviceId)
	if err != nil {
		RestBadRequest(w)
		return
	}
	var payload service.Service
	var unused int
	err = r.DecodeJsonPayload(&payload)
	if err != nil {
		glog.V(1).Info("Could not decode service payload: ", err)
		RestBadRequest(w)
		return
	}
	err = client.UpdateService(payload, &unused)
	if err != nil {
		glog.Errorf("Unable to update service %s: %v", serviceId, err)
		RestServerError(w)
		return
	}
	glog.V(1).Info("Updated service ", serviceId)
	w.WriteJson(&SimpleResponse{"Updated service", serviceLinks(serviceId)})
}

func RestRemoveService(w *rest.ResponseWriter, r *rest.Request, client *serviced.ControlClient) {
	var unused int
	serviceId, err := url.QueryUnescape(r.PathParam("serviceId"))
	if err != nil {
		RestBadRequest(w)
		return
	}
	err = client.RemoveService(serviceId, &unused)
	if err != nil {
		glog.Errorf("Could not remove service: %v", err)
		RestServerError(w)
		return
	}
	glog.V(0).Info("Removed service ", serviceId)
	w.WriteJson(&SimpleResponse{"Removed service", servicesLinks()})
}

func RestGetServiceLogs(w *rest.ResponseWriter, r *rest.Request, client *serviced.ControlClient) {
	serviceId, err := url.QueryUnescape(r.PathParam("serviceId"))
	if err != nil {
		RestBadRequest(w)
		return
	}
	var logs string
	err = client.GetServiceLogs(serviceId, &logs)
	if err != nil {
		glog.Errorf("Unexpected error getting service logs: %v", err)
		RestServerError(w)
	}
	w.WriteJson(&SimpleResponse{logs, serviceLinks(serviceId)})
}

// RestStartService starts the service with the given id and all of its children
// Note: Your mother, trebek.
func RestStartService(w *rest.ResponseWriter, r *rest.Request, client *serviced.ControlClient) {
	serviceId, err := url.QueryUnescape(r.PathParam("serviceId"))
	if err != nil {
		RestBadRequest(w)
		return
	}
	var i string
	err = client.StartService(serviceId, &i)
	if err != nil {
		glog.Errorf("Unexpected error starting service: %v", err)
		RestServerError(w)
	}
	w.WriteJson(&SimpleResponse{"Started service", serviceLinks(serviceId)})
}

// RestStopService stop the service with the given id and all of its children
func RestStopService(w *rest.ResponseWriter, r *rest.Request, client *serviced.ControlClient) {
	serviceId, err := url.QueryUnescape(r.PathParam("serviceId"))
	if err != nil {
		RestBadRequest(w)
		return
	}
	var i int
	err = client.StopService(serviceId, &i)
	if err != nil {
		glog.Errorf("Unexpected error stopping service: %v", err)
		RestServerError(w)
	}
	w.WriteJson(&SimpleResponse{"Stopped service", serviceLinks(serviceId)})
}

func RestSnapshotService(w *rest.ResponseWriter, r *rest.Request, client *serviced.ControlClient) {
	serviceId, err := url.QueryUnescape(r.PathParam("serviceId"))
	if err != nil {
		RestBadRequest(w)
		return
	}
	var label string
	err = client.Snapshot(serviceId, &label)
	if err != nil {
		glog.Errorf("Unexpected error snapshotting service: %v", err)
		RestServerError(w)
	}
	w.WriteJson(&SimpleResponse{label, serviceLinks(serviceId)})
}

func RestGetRunningService(w *rest.ResponseWriter, r *rest.Request, client *serviced.ControlClient) {
	serviceStateId, err := url.QueryUnescape(r.PathParam("serviceStateId"))
	if err != nil {
		RestBadRequest(w)
		return
	}
	serviceId, err := url.QueryUnescape(r.PathParam("serviceId"))
	if err != nil {
		RestBadRequest(w)
		return
	}
	request := dao.ServiceStateRequest{serviceId, serviceStateId}

	var running dao.RunningService
	err = client.GetRunningService(request, &running)
	if err != nil {
		glog.Errorf("Unexpected error retrieving services: %v", err)
		RestServerError(w)
	}
	w.WriteJson(running)
}

func RestGetServiceStateLogs(w *rest.ResponseWriter, r *rest.Request, client *serviced.ControlClient) {
	serviceStateId, err := url.QueryUnescape(r.PathParam("serviceStateId"))
	if err != nil {
		RestBadRequest(w)
		return
	}
	serviceId, err := url.QueryUnescape(r.PathParam("serviceId"))
	if err != nil {
		RestBadRequest(w)
		return
	}
	request := dao.ServiceStateRequest{serviceId, serviceStateId}

	var logs string
	err = client.GetServiceStateLogs(request, &logs)
	if err != nil {
		glog.Errorf("Unexpected error getting service state logs: %v", err)
		RestServerError(w)
	}
	w.WriteJson(&SimpleResponse{logs, servicesLinks()})
}

func RestGetServicedVersion(w *rest.ResponseWriter, r *rest.Request, client *serviced.ControlClient){
	w.WriteJson(&SimpleResponse{servicedversion.Version, servicesLinks()})
}

func RestBackupCreate(w *rest.ResponseWriter, r *rest.Request, client *serviced.ControlClient) {
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
		RestServerError(w)
	}
	w.WriteJson(&SimpleResponse{filepath, servicesLinks()})
}

func RestBackupRestore(w *rest.ResponseWriter, r *rest.Request, client *serviced.ControlClient) {
	home := os.Getenv("SERVICED_HOME")
	if home == "" {
		glog.Infof("SERVICED_HOME not set.  Backups will save to /tmp.")
		home = "/tmp"
	}

	err := r.ParseForm()
	filepath := r.FormValue("filename")

	if err != nil || filepath == ""{
		RestBadRequest(w)
		return
	}

	unused := 0

	err = client.AsyncRestore(home + "/backup/" + filepath, &unused)
	if err != nil {
		glog.Errorf("Unexpected error during restore: %v", err)
		RestServerError(w)
	}
	w.WriteJson(&SimpleResponse{string(unused), servicesLinks()})
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
	backupFiles, _ := ioutil.ReadDir(home+"/backup")

	for _, backupFileInfo := range backupFiles {
		if !backupFileInfo.IsDir(){
			fileInfo := JsonizableFileInfo{backupFileInfo.Name(), backupFileInfo.Size(), backupFileInfo.Mode(), backupFileInfo.ModTime()}
			fileData = append(fileData, fileInfo)
		}
	}

	w.WriteJson(&fileData)
}

func RestBackupStatus(w *rest.ResponseWriter, r *rest.Request, client *serviced.ControlClient) {
	backupStatus := ""
	err := client.BackupStatus("", &backupStatus)
	if err != nil {
		glog.Errorf("Unexpected error during backup status: %v", err)
		RestServerError(w)
	}
	w.WriteJson(&SimpleResponse{backupStatus, servicesLinks()})
}

func RestRestoreStatus(w *rest.ResponseWriter, r *rest.Request, client *serviced.ControlClient) {
	restoreStatus := ""
	err := client.RestoreStatus("", &restoreStatus)
	if err != nil {
		glog.Errorf("Unexpected error during restore status: %v", err)
		RestServerError(w)
	}
	w.WriteJson(&SimpleResponse{restoreStatus, servicesLinks()})
}
