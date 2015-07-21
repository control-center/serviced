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

// Package agent implements a service that runs on a serviced node. It is
// responsible for ensuring that a particular node is running the correct services
// and reporting the state and health of those services back to the master
// serviced.

package node

import (
	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/domain"
	"github.com/control-center/serviced/domain/addressassignment"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/servicestate"
	"github.com/control-center/serviced/domain/servicetemplate"
	"github.com/control-center/serviced/domain/user"
	"github.com/control-center/serviced/metrics"
	"github.com/control-center/serviced/rpc/rpcutils"
	"github.com/control-center/serviced/volume"
)

// A serviced client.
type ControlClient struct {
	addr      string
	rpcClient rpcutils.Client
}

// Ensure that ControlClient implements the ControlPlane interface.
var _ dao.ControlPlane = &ControlClient{}

// Create a new ControlClient.
func NewControlClient(addr string) (s *ControlClient, err error) {
	client, err := rpcutils.GetCachedClient(addr)
	if err != nil {
		return nil, err
	}
	s = new(ControlClient)
	s.addr = addr
	s.rpcClient = client
	return s, nil
}

// Return the matching hosts.
func (s *ControlClient) Close() (err error) {
	return s.rpcClient.Close()
}

func (s *ControlClient) GetServiceEndpoints(serviceId string, response *map[string][]dao.ApplicationEndpoint) (err error) {
	return s.rpcClient.Call("ControlPlane.GetServiceEndpoints", serviceId, response, 0, true)
}

func (s *ControlClient) GetServices(request dao.ServiceRequest, replyServices *[]service.Service) (err error) {
	return s.rpcClient.Call("ControlPlane.GetServices", request, replyServices, 0, true)
}

func (s *ControlClient) GetTaggedServices(request dao.ServiceRequest, replyServices *[]service.Service) (err error) {
	return s.rpcClient.Call("ControlPlane.GetTaggedServices", request, replyServices, 0, true)
}

func (s *ControlClient) GetService(serviceId string, service *service.Service) (err error) {
	return s.rpcClient.Call("ControlPlane.GetService", serviceId, &service, 0, true)
}

func (s *ControlClient) FindChildService(request dao.FindChildRequest, service *service.Service) (err error) {
	return s.rpcClient.Call("ControlPlane.FindChildService", request, &service, 0, true)
}

func (s *ControlClient) GetTenantId(serviceId string, tenantId *string) (err error) {
	return s.rpcClient.Call("ControlPlane.GetTenantId", serviceId, tenantId, 0, true)
}

func (s *ControlClient) AddService(service service.Service, serviceId *string) (err error) {
	return s.rpcClient.Call("ControlPlane.AddService", service, serviceId, 0, true)
}

func (s *ControlClient) CloneService(request dao.ServiceCloneRequest, copiedServiceId *string) (err error) {
	return s.rpcClient.Call("ControlPlane.CloneService", request, copiedServiceId, 0, true)
}

func (s *ControlClient) DeployService(service dao.ServiceDeploymentRequest, serviceId *string) (err error) {
	return s.rpcClient.Call("ControlPlane.DeployService", service, serviceId, 0, true)
}

func (s *ControlClient) UpdateService(service service.Service, unused *int) (err error) {
	return s.rpcClient.Call("ControlPlane.UpdateService", service, unused, 0, true)
}

func (s *ControlClient) RunMigrationScript(request dao.RunMigrationScriptRequest, unused *int) (err error) {
	return s.rpcClient.Call("ControlPlane.RunMigrationScript", request, unused, 0, true)
}

func (s *ControlClient) MigrateServices(request dao.ServiceMigrationRequest, unused *int) (err error) {
	return s.rpcClient.Call("ControlPlane.MigrateServices", request, unused, 0, true)
}

func (s *ControlClient) GetServiceList(serviceID string, services *[]service.Service) (err error) {
	return s.rpcClient.Call("ControlPlane.GetServiceList", serviceID, services, 0, true)
}

func (s *ControlClient) RemoveService(serviceId string, unused *int) (err error) {
	return s.rpcClient.Call("ControlPlane.RemoveService", serviceId, unused, 0, true)
}

func (s *ControlClient) AssignIPs(assignmentRequest dao.AssignmentRequest, _ *struct{}) (err error) {
	return s.rpcClient.Call("ControlPlane.AssignIPs", assignmentRequest, nil, 0, true)
}

func (s *ControlClient) GetServiceAddressAssignments(serviceID string, addresses *[]addressassignment.AddressAssignment) (err error) {
	return s.rpcClient.Call("ControlPlane.GetServiceAddressAssignments", serviceID, addresses, 0, true)
}

func (s *ControlClient) GetServiceLogs(serviceId string, logs *string) error {
	return s.rpcClient.Call("ControlPlane.GetServiceLogs", serviceId, logs, 0, true)
}

func (s *ControlClient) GetServiceStateLogs(request dao.ServiceStateRequest, logs *string) error {
	return s.rpcClient.Call("ControlPlane.GetServiceStateLogs", request, logs, 0, true)
}

func (s *ControlClient) GetRunningServicesForHost(hostId string, runningServices *[]dao.RunningService) (err error) {
	return s.rpcClient.Call("ControlPlane.GetRunningServicesForHost", hostId, runningServices, 0, true)
}

func (s *ControlClient) GetRunningServicesForService(serviceId string, runningServices *[]dao.RunningService) (err error) {
	return s.rpcClient.Call("ControlPlane.GetRunningServicesForService", serviceId, runningServices, 0, true)
}

func (s *ControlClient) StopRunningInstance(request dao.HostServiceRequest, unused *int) (err error) {
	return s.rpcClient.Call("ControlPlane.StopRunningInstance", request, unused, 0, true)
}

func (s *ControlClient) GetRunningServices(request dao.EntityRequest, runningServices *[]dao.RunningService) (err error) {
	return s.rpcClient.Call("ControlPlane.GetRunningServices", request, runningServices, 10, false)
}

func (s *ControlClient) GetServiceState(request dao.ServiceStateRequest, state *servicestate.ServiceState) error {
	return s.rpcClient.Call("ControlPlane.GetServiceState", request, state, 0, true)
}

func (s *ControlClient) GetRunningService(request dao.ServiceStateRequest, running *dao.RunningService) error {
	return s.rpcClient.Call("ControlPlane.GetRunningService", request, running, 0, true)
}

func (s *ControlClient) GetServiceStates(serviceId string, states *[]servicestate.ServiceState) (err error) {
	return s.rpcClient.Call("ControlPlane.GetServiceStates", serviceId, states, 0, true)
}

func (s *ControlClient) StartService(request dao.ScheduleServiceRequest, affected *int) (err error) {
	return s.rpcClient.Call("ControlPlane.StartService", request, affected, 0, true)
}

func (s *ControlClient) RestartService(request dao.ScheduleServiceRequest, affected *int) (err error) {
	return s.rpcClient.Call("ControlPlane.RestartService", request, affected, 0, true)
}

func (s *ControlClient) StopService(request dao.ScheduleServiceRequest, affected *int) (err error) {
	return s.rpcClient.Call("ControlPlane.StopService", request, affected, 0, true)
}

func (s *ControlClient) WaitService(request dao.WaitServiceRequest, _ *struct{}) (err error) {
	return s.rpcClient.Call("ControlPlane.WaitService", request, nil, 0, true)
}

func (s *ControlClient) UpdateServiceState(state servicestate.ServiceState, unused *int) (err error) {
	return s.rpcClient.Call("ControlPlane.UpdateServiceState", state, unused, 0, true)
}

func (s *ControlClient) GetServiceStatus(serviceID string, statusmap *map[string]dao.ServiceStatus) (err error) {
	return s.rpcClient.Call("ControlPlane.GetServiceStatus", serviceID, statusmap, 0, true)
}

func (s *ControlClient) DeployTemplate(request dao.ServiceTemplateDeploymentRequest, tenantIDs *[]string) error {
	return s.rpcClient.Call("ControlPlane.DeployTemplate", request, tenantIDs, 0, false)
}

func (s *ControlClient) DeployTemplateStatus(request dao.ServiceTemplateDeploymentRequest, status *string) error {
	return s.rpcClient.Call("ControlPlane.DeployTemplateStatus", request, status, 0, true)
}

func (s *ControlClient) DeployTemplateActive(notUsed string, active *[]map[string]string) error {
	return s.rpcClient.Call("ControlPlane.DeployTemplateActive", notUsed, active, 0, true)
}

func (s *ControlClient) GetServiceTemplates(unused int, serviceTemplates *map[string]servicetemplate.ServiceTemplate) error {
	return s.rpcClient.Call("ControlPlane.GetServiceTemplates", unused, serviceTemplates, 0, true)
}

func (s *ControlClient) AddServiceTemplate(serviceTemplate servicetemplate.ServiceTemplate, templateId *string) error {
	return s.rpcClient.Call("ControlPlane.AddServiceTemplate", serviceTemplate, templateId, 0, true)
}

func (s *ControlClient) UpdateServiceTemplate(serviceTemplate servicetemplate.ServiceTemplate, unused *int) error {
	return s.rpcClient.Call("ControlPlane.UpdateServiceTemplate", serviceTemplate, unused, 0, true)
}

func (s *ControlClient) RemoveServiceTemplate(serviceTemplateID string, unused *int) error {
	return s.rpcClient.Call("ControlPlane.RemoveServiceTemplate", serviceTemplateID, unused, 0, true)
}

func (s *ControlClient) GetVolume(serviceID string, volume volume.Volume) error {
	return s.rpcClient.Call("ControlPlane.GetVolume", serviceID, volume, 0, true)
}

func (s *ControlClient) ResetRegistry(request dao.EntityRequest, unused *int) error {
	return s.rpcClient.Call("ControlPlane.ResetRegistry", request, unused, 0, true)
}

func (s *ControlClient) DeleteSnapshot(snapshotId string, unused *int) error {
	return s.rpcClient.Call("ControlPlane.DeleteSnapshot", snapshotId, unused, 0, true)
}

func (s *ControlClient) DeleteSnapshots(serviceId string, unused *int) error {
	return s.rpcClient.Call("ControlPlane.DeleteSnapshots", serviceId, unused, 0, true)
}

func (s *ControlClient) Rollback(request dao.RollbackRequest, unused *int) error {
	return s.rpcClient.Call("ControlPlane.Rollback", request, unused, 0, true)
}

func (s *ControlClient) Snapshot(request dao.SnapshotRequest, label *string) error {
	return s.rpcClient.Call("ControlPlane.Snapshot", request, label, 0, true)
}

func (s *ControlClient) AsyncSnapshot(serviceId string, label *string) error {
	return s.rpcClient.Call("ControlPlane.AsyncSnapshot", serviceId, label, 0, true)
}

func (s *ControlClient) ListSnapshots(serviceId string, snapshots *[]dao.SnapshotInfo) error {
	return s.rpcClient.Call("ControlPlane.ListSnapshots", serviceId, snapshots, 0, true)
}

func (s *ControlClient) Commit(containerId string, label *string) error {
	return s.rpcClient.Call("ControlPlane.Commit", containerId, label, 0, true)
}

func (s *ControlClient) ReadyDFS(unused bool, unusedint *int) error {
	return s.rpcClient.Call("ControlPlane.ReadyDFS", unused, unusedint, 0, true)
}

func (s *ControlClient) ListBackups(backupDirectory string, backupFiles *[]dao.BackupFile) error {
	return s.rpcClient.Call("ControlPlane.ListBackups", backupDirectory, backupFiles, 0, true)
}

func (s *ControlClient) Backup(backupDirectory string, backupFilePath *string) error {
	return s.rpcClient.Call("ControlPlane.Backup", backupDirectory, backupFilePath, 0, false)
}

func (s *ControlClient) AsyncBackup(backupDirectory string, backupFilePath *string) error {
	return s.rpcClient.Call("ControlPlane.AsyncBackup", backupDirectory, backupFilePath, 0, true)
}

func (s *ControlClient) Restore(backupFilePath string, unused *int) error {
	return s.rpcClient.Call("ControlPlane.Restore", backupFilePath, unused, 0, false)
}

func (s *ControlClient) AsyncRestore(backupFilePath string, unused *int) error {
	return s.rpcClient.Call("ControlPlane.AsyncRestore", backupFilePath, unused, 0, true)
}

func (s *ControlClient) BackupStatus(notUsed int, backupStatus *string) error {
	return s.rpcClient.Call("ControlPlane.BackupStatus", notUsed, backupStatus, 0, true)
}

func (s *ControlClient) ImageLayerCount(imageUUID string, layers *int) error {
	return s.rpcClient.Call("ControlPlane.ImageLayerCount", imageUUID, layers, 0, true)
}

func (s *ControlClient) ValidateCredentials(user user.User, result *bool) error {
	return s.rpcClient.Call("ControlPlane.ValidateCredentials", user, result, 0, true)
}

func (s *ControlClient) GetSystemUser(unused int, user *user.User) error {
	return s.rpcClient.Call("ControlPlane.GetSystemUser", unused, user, 0, true)
}

func (s *ControlClient) Action(req dao.AttachRequest, unused *int) error {
	return s.rpcClient.Call("ControlPlane.Action", req, unused, 0, true)
}

func (s *ControlClient) GetHostMemoryStats(req dao.MetricRequest, stats *metrics.MemoryUsageStats) error {
	return s.rpcClient.Call("ControlPlane.GetHostMemoryStats", req, stats, 5, false)
}

func (s *ControlClient) GetServiceMemoryStats(req dao.MetricRequest, stats *metrics.MemoryUsageStats) error {
	return s.rpcClient.Call("ControlPlane.GetServiceMemoryStats", req, stats, 5, false)
}

func (s *ControlClient) GetInstanceMemoryStats(req dao.MetricRequest, stats *[]metrics.MemoryUsageStats) error {
	return s.rpcClient.Call("ControlPlane.GetInstanceMemoryStats", req, stats, 5, false)
}

func (s *ControlClient) LogHealthCheck(result domain.HealthCheckResult, unused *int) error {
	return s.rpcClient.Call("ControlPlane.LogHealthCheck", result, unused, 0, true)
}

func (s *ControlClient) ServicedHealthCheck(IServiceNames []string, results *[]dao.IServiceHealthResult) error {
	return s.rpcClient.Call("ControlPlane.ServicedHealthCheck", IServiceNames, results, 0, true)
}
