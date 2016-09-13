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
	"github.com/control-center/serviced/domain/addressassignment"
	"github.com/control-center/serviced/domain/applicationendpoint"
	"github.com/control-center/serviced/domain/service"
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
	return s.rpcClient.Call("ControlCenter.GetServiceEndpoints", serviceId, response, 0)
}

func (s *ControlClient) GetServices(request dao.ServiceRequest, replyServices *[]service.Service) (err error) {
	return s.rpcClient.Call("ControlCenter.GetServices", request, replyServices, 0)
}

func (s *ControlClient) GetTaggedServices(request dao.ServiceRequest, replyServices *[]service.Service) (err error) {
	return s.rpcClient.Call("ControlCenter.GetTaggedServices", request, replyServices, 0)
}

func (s *ControlClient) GetService(serviceId string, service *service.Service) (err error) {
	return s.rpcClient.Call("ControlCenter.GetService", serviceId, service, 0)
}

func (s *ControlClient) FindChildService(request dao.FindChildRequest, service *service.Service) (err error) {
	return s.rpcClient.Call("ControlCenter.FindChildService", request, service, 0)
}

func (s *ControlClient) AddService(service service.Service, serviceId *string) (err error) {
	return s.rpcClient.Call("ControlCenter.AddService", service, serviceId, 0)
}

func (s *ControlClient) CloneService(request dao.ServiceCloneRequest, copiedServiceId *string) (err error) {
	return s.rpcClient.Call("ControlCenter.CloneService", request, copiedServiceId, 0)
}

func (s *ControlClient) DeployService(service dao.ServiceDeploymentRequest, serviceId *string) (err error) {
	return s.rpcClient.Call("ControlCenter.DeployService", service, serviceId, 0)
}

func (s *ControlClient) UpdateService(service service.Service, unused *int) (err error) {
	return s.rpcClient.Call("ControlCenter.UpdateService", service, unused, 0)
}

func (s *ControlClient) MigrateServices(request dao.ServiceMigrationRequest, unused *int) (err error) {
	return s.rpcClient.Call("ControlCenter.MigrateServices", request, unused, 0)
}

func (s *ControlClient) GetServiceList(serviceID string, services *[]service.Service) (err error) {
	return s.rpcClient.Call("ControlCenter.GetServiceList", serviceID, services, 0)
}

func (s *ControlClient) RemoveService(serviceId string, unused *int) (err error) {
	return s.rpcClient.Call("ControlCenter.RemoveService", serviceId, unused, 0)
}

func (s *ControlClient) AssignIPs(assignmentRequest addressassignment.AssignmentRequest, _ *int) (err error) {
	return s.rpcClient.Call("ControlCenter.AssignIPs", assignmentRequest, nil, 0)
}

func (s *ControlClient) GetServiceLogs(serviceId string, logs *string) error {
	return s.rpcClient.Call("ControlCenter.GetServiceLogs", serviceId, logs, 0)
}

func (s *ControlClient) GetServiceStateLogs(request dao.ServiceStateRequest, logs *string) error {
	return s.rpcClient.Call("ControlCenter.GetServiceStateLogs", request, logs, 0)
}

func (s *ControlClient) GetRunningServicesForHost(hostId string, runningServices *[]dao.RunningService) (err error) {
	return s.rpcClient.Call("ControlCenter.GetRunningServicesForHost", hostId, runningServices, 0)
}

func (s *ControlClient) GetRunningServicesForService(serviceId string, runningServices *[]dao.RunningService) (err error) {
	return s.rpcClient.Call("ControlCenter.GetRunningServicesForService", serviceId, runningServices, 0)
}

func (s *ControlClient) StopRunningInstance(request dao.HostServiceRequest, unused *int) (err error) {
	return s.rpcClient.Call("ControlCenter.StopRunningInstance", request, unused, 0)
}

func (s *ControlClient) GetRunningServices(request dao.EntityRequest, runningServices *[]dao.RunningService) (err error) {
	return s.rpcClient.Call("ControlCenter.GetRunningServices", request, runningServices, 10*time.Second)
}

func (s *ControlClient) StartService(request dao.ScheduleServiceRequest, affected *int) (err error) {
	return s.rpcClient.Call("ControlCenter.StartService", request, affected, 0)
}

func (s *ControlClient) RestartService(request dao.ScheduleServiceRequest, affected *int) (err error) {
	return s.rpcClient.Call("ControlCenter.RestartService", request, affected, 0)
}

func (s *ControlClient) StopService(request dao.ScheduleServiceRequest, affected *int) (err error) {
	return s.rpcClient.Call("ControlCenter.StopService", request, affected, 0)
}

func (s *ControlClient) WaitService(request dao.WaitServiceRequest, _ *int) (err error) {
	return s.rpcClient.Call("ControlCenter.WaitService", request, nil, 0)
}

func (s *ControlClient) GetServiceStatus(serviceID string, statusmap *[]service.Instance) (err error) {
	return s.rpcClient.Call("ControlCenter.GetServiceStatus", serviceID, statusmap, 0)
}

func (s *ControlClient) Action(req dao.AttachRequest, unused *int) error {
	return s.rpcClient.Call("ControlCenter.Action", req, unused, 0)
}

func (s *ControlClient) GetHostMemoryStats(req dao.MetricRequest, stats *metrics.MemoryUsageStats) error {
	return s.rpcClient.Call("ControlCenter.GetHostMemoryStats", req, stats, 5*time.Second)
}
func (s *ControlClient) TagSnapshot(request dao.TagSnapshotRequest, unused *int) error {
	return s.rpcClient.Call("ControlCenter.TagSnapshot", request, unused, 0)
}

func (s *ControlClient) RemoveSnapshotTag(request dao.SnapshotByTagRequest, snapshotID *string) error {
	return s.rpcClient.Call("ControlCenter.RemoveSnapshotTag", request, snapshotID, 0)
}

func (s *ControlClient) GetSnapshotByServiceIDAndTag(request dao.SnapshotByTagRequest, snapshot *dao.SnapshotInfo) error {
	return s.rpcClient.Call("ControlCenter.GetSnapshotByServiceIDAndTag", request, snapshot, 0)
}

func (s *ControlClient) AsyncSnapshot(serviceId string, label *string) error {
	return s.rpcClient.Call("ControlCenter.AsyncSnapshot", serviceId, label, 0)
}

func (s *ControlClient) GetServiceMemoryStats(req dao.MetricRequest, stats *metrics.MemoryUsageStats) error {
	return s.rpcClient.Call("ControlCenter.GetServiceMemoryStats", req, stats, 5*time.Second)
}

func (s *ControlClient) GetInstanceMemoryStats(req dao.MetricRequest, stats *[]metrics.MemoryUsageStats) error {
	return s.rpcClient.Call("ControlCenter.GetInstanceMemoryStats", req, stats, 5*time.Second)
}

func (s *ControlClient) GetServicesHealth(unused int, results *map[string]map[int]map[string]health.HealthStatus) error {
	return s.rpcClient.Call("ControlCenter.GetServicesHealth", unused, results, 0)
}

func (s *ControlClient) ServicedHealthCheck(IServiceNames []string, results *[]dao.IServiceHealthResult) error {
	return s.rpcClient.Call("ControlCenter.ServicedHealthCheck", IServiceNames, results, 0)
}

func (s *ControlClient) ReportHealthStatus(req dao.HealthStatusRequest, unused *int) error {
	return s.rpcClient.Call("ControlCenter.ReportHealthStatus", req, unused, 0)
}

func (s *ControlClient) ReportInstanceDead(req dao.ServiceInstanceRequest, unused *int) error {
	return s.rpcClient.Call("ControlCenter.ReportInstanceDead", req, unused, 0)
}

func (s *ControlClient) Backup(backupRequest dao.BackupRequest, filename *string) (err error) {
	return s.rpcClient.Call("ControlCenter.Backup", backupRequest, filename, 0)
}

func (s *ControlClient) AsyncBackup(backupRequest dao.BackupRequest, filename *string) (err error) {
	return s.rpcClient.Call("ControlCenter.AsyncBackup", backupRequest, filename, 0)
}

func (s *ControlClient) Restore(filename string, unused *int) (err error) {
	return s.rpcClient.Call("ControlCenter.Restore", filename, unused, 0)
}

func (s *ControlClient) AsyncRestore(filename string, unused *int) (err error) {
	return s.rpcClient.Call("ControlCenter.AsyncRestore", filename, unused, 0)
}

func (s *ControlClient) ListBackups(dirpath string, files *[]dao.BackupFile) (err error) {
	return s.rpcClient.Call("ControlCenter.ListBackups", dirpath, files, 0)
}

func (s *ControlClient) BackupStatus(req dao.EntityRequest, status *string) (err error) {
	return s.rpcClient.Call("ControlCenter.BackupStatus", req, status, 0)
}

func (s *ControlClient) Snapshot(req dao.SnapshotRequest, snapshotID *string) (err error) {
	return s.rpcClient.Call("ControlCenter.Snapshot", req, snapshotID, 0)
}

func (s *ControlClient) Rollback(req dao.RollbackRequest, unused *int) (err error) {
	return s.rpcClient.Call("ControlCenter.Rollback", req, unused, 0)
}

func (s *ControlClient) DeleteSnapshot(snapshotID string, unused *int) (err error) {
	return s.rpcClient.Call("ControlCenter.DeleteSnapshot", snapshotID, unused, 0)
}

func (s *ControlClient) DeleteSnapshots(serviceID string, unused *int) (err error) {
	return s.rpcClient.Call("ControlCenter.DeleteSnapshots", serviceID, unused, 0)
}

func (s *ControlClient) ListSnapshots(serviceID string, snapshots *[]dao.SnapshotInfo) (err error) {
	return s.rpcClient.Call("ControlCenter.ListSnapshots", serviceID, snapshots, 0)
}

func (s *ControlClient) ResetRegistry(req dao.EntityRequest, unused *int) (err error) {
	return s.rpcClient.Call("ControlCenter.ResetRegistry", req, unused, 0)
}

func (s *ControlClient) RepairRegistry(req dao.EntityRequest, unused *int) (err error) {
	return s.rpcClient.Call("ControlCenter.RepairRegistry", req, unused, 0)
}

func (s *ControlClient) ReadyDFS(serviceID string, unused *int) (err error) {
	return s.rpcClient.Call("ControlCenter.ReadyDFS", serviceID, unused, 0)
}
