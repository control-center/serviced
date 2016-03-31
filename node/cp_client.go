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
	"time"

	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/domain"
	"github.com/control-center/serviced/domain/addressassignment"
	"github.com/control-center/serviced/domain/applicationendpoint"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/servicestate"
	"github.com/control-center/serviced/domain/user"
	"github.com/control-center/serviced/health"
	"github.com/control-center/serviced/metrics"
	"github.com/control-center/serviced/rpc/rpcutils"
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

func (s *ControlClient) GetServiceEndpoints(serviceId string, response *map[string][]applicationendpoint.ApplicationEndpoint) (err error) {
	return s.rpcClient.Call("ControlPlane.GetServiceEndpoints", serviceId, response, 0)
}

func (s *ControlClient) GetServices(request dao.ServiceRequest, replyServices *[]service.Service) (err error) {
	return s.rpcClient.Call("ControlPlane.GetServices", request, replyServices, 0)
}

func (s *ControlClient) GetTaggedServices(request dao.ServiceRequest, replyServices *[]service.Service) (err error) {
	return s.rpcClient.Call("ControlPlane.GetTaggedServices", request, replyServices, 0)
}

func (s *ControlClient) GetService(serviceId string, service *service.Service) (err error) {
	return s.rpcClient.Call("ControlPlane.GetService", serviceId, service, 0)
}

func (s *ControlClient) FindChildService(request dao.FindChildRequest, service *service.Service) (err error) {
	return s.rpcClient.Call("ControlPlane.FindChildService", request, service, 0)
}

func (s *ControlClient) GetTenantId(serviceId string, tenantId *string) (err error) {
	return s.rpcClient.Call("ControlPlane.GetTenantId", serviceId, tenantId, 0)
}

func (s *ControlClient) AddService(service service.Service, serviceId *string) (err error) {
	return s.rpcClient.Call("ControlPlane.AddService", service, serviceId, 0)
}

func (s *ControlClient) CloneService(request dao.ServiceCloneRequest, copiedServiceId *string) (err error) {
	return s.rpcClient.Call("ControlPlane.CloneService", request, copiedServiceId, 0)
}

func (s *ControlClient) DeployService(service dao.ServiceDeploymentRequest, serviceId *string) (err error) {
	return s.rpcClient.Call("ControlPlane.DeployService", service, serviceId, 0)
}

func (s *ControlClient) UpdateService(service service.Service, unused *int) (err error) {
	return s.rpcClient.Call("ControlPlane.UpdateService", service, unused, 0)
}

func (s *ControlClient) MigrateServices(request dao.ServiceMigrationRequest, unused *int) (err error) {
	return s.rpcClient.Call("ControlPlane.MigrateServices", request, unused, 0)
}

func (s *ControlClient) GetServiceList(serviceID string, services *[]service.Service) (err error) {
	return s.rpcClient.Call("ControlPlane.GetServiceList", serviceID, services, 0)
}

func (s *ControlClient) RemoveService(serviceId string, unused *int) (err error) {
	return s.rpcClient.Call("ControlPlane.RemoveService", serviceId, unused, 0)
}

func (s *ControlClient) AssignIPs(assignmentRequest addressassignment.AssignmentRequest, _ *int) (err error) {
	return s.rpcClient.Call("ControlPlane.AssignIPs", assignmentRequest, nil, 0)
}

func (s *ControlClient) GetServiceLogs(serviceId string, logs *string) error {
	return s.rpcClient.Call("ControlPlane.GetServiceLogs", serviceId, logs, 0)
}

func (s *ControlClient) GetServiceStateLogs(request dao.ServiceStateRequest, logs *string) error {
	return s.rpcClient.Call("ControlPlane.GetServiceStateLogs", request, logs, 0)
}

func (s *ControlClient) GetRunningServicesForHost(hostId string, runningServices *[]dao.RunningService) (err error) {
	return s.rpcClient.Call("ControlPlane.GetRunningServicesForHost", hostId, runningServices, 0)
}

func (s *ControlClient) GetRunningServicesForService(serviceId string, runningServices *[]dao.RunningService) (err error) {
	return s.rpcClient.Call("ControlPlane.GetRunningServicesForService", serviceId, runningServices, 0)
}

func (s *ControlClient) StopRunningInstance(request dao.HostServiceRequest, unused *int) (err error) {
	return s.rpcClient.Call("ControlPlane.StopRunningInstance", request, unused, 0)
}

func (s *ControlClient) GetRunningServices(request dao.EntityRequest, runningServices *[]dao.RunningService) (err error) {
	return s.rpcClient.Call("ControlPlane.GetRunningServices", request, runningServices, 10*time.Second)
}

func (s *ControlClient) GetServiceState(request dao.ServiceStateRequest, state *servicestate.ServiceState) error {
	return s.rpcClient.Call("ControlPlane.GetServiceState", request, state, 0)
}

func (s *ControlClient) GetRunningService(request dao.ServiceStateRequest, running *dao.RunningService) error {
	return s.rpcClient.Call("ControlPlane.GetRunningService", request, running, 0)
}

func (s *ControlClient) GetServiceStates(serviceId string, states *[]servicestate.ServiceState) (err error) {
	return s.rpcClient.Call("ControlPlane.GetServiceStates", serviceId, states, 0)
}

func (s *ControlClient) StartService(request dao.ScheduleServiceRequest, affected *int) (err error) {
	return s.rpcClient.Call("ControlPlane.StartService", request, affected, 0)
}

func (s *ControlClient) RestartService(request dao.ScheduleServiceRequest, affected *int) (err error) {
	return s.rpcClient.Call("ControlPlane.RestartService", request, affected, 0)
}

func (s *ControlClient) StopService(request dao.ScheduleServiceRequest, affected *int) (err error) {
	return s.rpcClient.Call("ControlPlane.StopService", request, affected, 0)
}

func (s *ControlClient) WaitService(request dao.WaitServiceRequest, _ *int) (err error) {
	return s.rpcClient.Call("ControlPlane.WaitService", request, nil, 0)
}

func (s *ControlClient) UpdateServiceState(state servicestate.ServiceState, unused *int) (err error) {
	return s.rpcClient.Call("ControlPlane.UpdateServiceState", state, unused, 0)
}

func (s *ControlClient) GetServiceStatus(serviceID string, statusmap *map[string]dao.ServiceStatus) (err error) {
	return s.rpcClient.Call("ControlPlane.GetServiceStatus", serviceID, statusmap, 0)
}

func (s *ControlClient) ValidateCredentials(user user.User, result *bool) error {
	return s.rpcClient.Call("ControlPlane.ValidateCredentials", user, result, 0)
}

func (s *ControlClient) GetSystemUser(unused int, user *user.User) error {
	return s.rpcClient.Call("ControlPlane.GetSystemUser", unused, user, 0)
}

func (s *ControlClient) Action(req dao.AttachRequest, unused *int) error {
	return s.rpcClient.Call("ControlPlane.Action", req, unused, 0)
}

func (s *ControlClient) GetHostMemoryStats(req dao.MetricRequest, stats *metrics.MemoryUsageStats) error {
	return s.rpcClient.Call("ControlPlane.GetHostMemoryStats", req, stats, 5*time.Second)
}
func (s *ControlClient) TagSnapshot(request dao.TagSnapshotRequest, unused *int) error {
	return s.rpcClient.Call("ControlPlane.TagSnapshot", request, unused, 0)
}

func (s *ControlClient) RemoveSnapshotTag(request dao.SnapshotByTagRequest, snapshotID *string) error {
	return s.rpcClient.Call("ControlPlane.RemoveSnapshotTag", request, snapshotID, 0)
}

func (s *ControlClient) GetSnapshotByServiceIDAndTag(request dao.SnapshotByTagRequest, snapshot *dao.SnapshotInfo) error {
	return s.rpcClient.Call("ControlPlane.GetSnapshotByServiceIDAndTag", request, snapshot, 0)
}

func (s *ControlClient) AsyncSnapshot(serviceId string, label *string) error {
	return s.rpcClient.Call("ControlPlane.AsyncSnapshot", serviceId, label, 0)
}

func (s *ControlClient) GetServiceMemoryStats(req dao.MetricRequest, stats *metrics.MemoryUsageStats) error {
	return s.rpcClient.Call("ControlPlane.GetServiceMemoryStats", req, stats, 5*time.Second)
}

func (s *ControlClient) GetInstanceMemoryStats(req dao.MetricRequest, stats *[]metrics.MemoryUsageStats) error {
	return s.rpcClient.Call("ControlPlane.GetInstanceMemoryStats", req, stats, 5*time.Second)
}

func (s *ControlClient) LogHealthCheck(result domain.HealthCheckResult, unused *int) error {
	return s.rpcClient.Call("ControlPlane.LogHealthCheck", result, unused, 0)
}

func (s *ControlClient) GetServicesHealth(unused int, results *map[string]map[int]map[string]health.HealthStatus) error {
	return s.rpcClient.Call("ControlPlane.GetServicesHealth", unused, results, 0)
}

func (s *ControlClient) ServicedHealthCheck(IServiceNames []string, results *[]dao.IServiceHealthResult) error {
	return s.rpcClient.Call("ControlPlane.ServicedHealthCheck", IServiceNames, results, 0)
}

func (s *ControlClient) ReportHealthStatus(req dao.HealthStatusRequest, unused *int) error {
	return s.rpcClient.Call("ControlPlane.ReportHealthStatus", req, unused, 0)
}

func (s *ControlClient) ReportInstanceDead(req dao.ServiceInstanceRequest, unused *int) error {
	return s.rpcClient.Call("ControlPlane.ReportInstanceDead", req, unused, 0)
}

func (s *ControlClient) Backup(dirpath string, filename *string) (err error) {
	return s.rpcClient.Call("ControlPlane.Backup", dirpath, filename, 0)
}

func (s *ControlClient) AsyncBackup(dirpath string, filename *string) (err error) {
	return s.rpcClient.Call("ControlPlane.AsyncBackup", dirpath, filename, 0)
}

func (s *ControlClient) Restore(filename string, unused *int) (err error) {
	return s.rpcClient.Call("ControlPlane.Restore", filename, unused, 0)
}

func (s *ControlClient) AsyncRestore(filename string, unused *int) (err error) {
	return s.rpcClient.Call("ControlPlane.AsyncRestore", filename, unused, 0)
}

func (s *ControlClient) ListBackups(dirpath string, files *[]dao.BackupFile) (err error) {
	return s.rpcClient.Call("ControlPlane.ListBackups", dirpath, files, 0)
}

func (s *ControlClient) BackupStatus(req dao.EntityRequest, status *string) (err error) {
	return s.rpcClient.Call("ControlPlane.BackupStatus", req, status, 0)
}

func (s *ControlClient) Snapshot(req dao.SnapshotRequest, snapshotID *string) (err error) {
	return s.rpcClient.Call("ControlPlane.Snapshot", req, snapshotID, 0)
}

func (s *ControlClient) Rollback(req dao.RollbackRequest, unused *int) (err error) {
	return s.rpcClient.Call("ControlPlane.Rollback", req, unused, 0)
}

func (s *ControlClient) DeleteSnapshot(snapshotID string, unused *int) (err error) {
	return s.rpcClient.Call("ControlPlane.DeleteSnapshot", snapshotID, unused, 0)
}

func (s *ControlClient) DeleteSnapshots(serviceID string, unused *int) (err error) {
	return s.rpcClient.Call("ControlPlane.DeleteSnapshots", serviceID, unused, 0)
}

func (s *ControlClient) ListSnapshots(serviceID string, snapshots *[]dao.SnapshotInfo) (err error) {
	return s.rpcClient.Call("ControlPlane.ListSnapshots", serviceID, snapshots, 0)
}

func (s *ControlClient) ResetRegistry(req dao.EntityRequest, unused *int) (err error) {
	return s.rpcClient.Call("ControlPlane.ResetRegistry", req, unused, 0)
}

func (s *ControlClient) RepairRegistry(req dao.EntityRequest, unused *int) (err error) {
	return s.rpcClient.Call("ControlPlane.RepairRegistry", req, unused, 0)
}

func (s *ControlClient) ReadyDFS(serviceID string, unused *int) (err error) {
	return s.rpcClient.Call("ControlPlane.ReadyDFS", serviceID, unused, 0)
}
