// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

// Package agent implements a service that runs on a serviced node. It is
// responsible for ensuring that a particular node is running the correct services
// and reporting the state and health of those services back to the master
// serviced.

package serviced

import (
	"net/rpc"

	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/dao"
	"github.com/zenoss/serviced/domain/addressassignment"
	"github.com/zenoss/serviced/domain/service"
	"github.com/zenoss/serviced/domain/servicestate"
	"github.com/zenoss/serviced/domain/servicetemplate"
	"github.com/zenoss/serviced/volume"
)

// A serviced client.
type ControlClient struct {
	addr      string
	rpcClient *rpc.Client
}

// Ensure that ControlClient implements the ControlPlane interface.
var _ dao.ControlPlane = &ControlClient{}

// Create a new ControlClient.
func NewControlClient(addr string) (s *ControlClient, err error) {
	s = new(ControlClient)
	s.addr = addr
	glog.V(4).Infof("Connecting to %s", addr)
	rpcClient, err := rpc.DialHTTP("tcp", s.addr)
	s.rpcClient = rpcClient
	return s, err
}

// Return the matching hosts.
func (s *ControlClient) Close() (err error) {
	return s.rpcClient.Close()
}

func (s *ControlClient) GetServiceEndpoints(serviceId string, response *map[string][]*dao.ApplicationEndpoint) (err error) {
	return s.rpcClient.Call("ControlPlane.GetServiceEndpoints", serviceId, response)
}

func (s *ControlClient) GetServices(request dao.EntityRequest, replyServices *[]*service.Service) (err error) {
	return s.rpcClient.Call("ControlPlane.GetServices", request, replyServices)
}

func (s *ControlClient) GetTaggedServices(request dao.EntityRequest, replyServices *[]*service.Service) (err error) {
	return s.rpcClient.Call("ControlPlane.GetTaggedServices", request, replyServices)
}

func (s *ControlClient) GetService(serviceId string, service *service.Service) (err error) {
	return s.rpcClient.Call("ControlPlane.GetService", serviceId, &service)
}

func (s *ControlClient) GetTenantId(serviceId string, tenantId *string) (err error) {
	return s.rpcClient.Call("ControlPlane.GetTenantId", serviceId, tenantId)
}

func (s *ControlClient) AddService(service service.Service, serviceId *string) (err error) {
	return s.rpcClient.Call("ControlPlane.AddService", service, serviceId)
}

func (s *ControlClient) UpdateService(service service.Service, unused *int) (err error) {
	return s.rpcClient.Call("ControlPlane.UpdateService", service, unused)
}

func (s *ControlClient) RemoveService(serviceId string, unused *int) (err error) {
	return s.rpcClient.Call("ControlPlane.RemoveService", serviceId, unused)
}

func (s *ControlClient) AssignIPs(assignmentRequest dao.AssignmentRequest, _ *struct{}) (err error) {
	return s.rpcClient.Call("ControlPlane.AssignIPs", assignmentRequest, nil)
}

func (s *ControlClient) GetServiceAddressAssignments(serviceID string, addresses *[]*addressassignment.AddressAssignment) (err error) {
	return s.rpcClient.Call("ControlPlane.GetServiceAddressAssignments", serviceID, addresses)
}

func (s *ControlClient) GetServiceLogs(serviceId string, logs *string) error {
	return s.rpcClient.Call("ControlPlane.GetServiceLogs", serviceId, logs)
}

func (s *ControlClient) GetServiceStateLogs(request dao.ServiceStateRequest, logs *string) error {
	return s.rpcClient.Call("ControlPlane.GetServiceStateLogs", request, logs)
}

func (s *ControlClient) GetRunningServicesForHost(hostId string, runningServices *[]*dao.RunningService) (err error) {
	return s.rpcClient.Call("ControlPlane.GetRunningServicesForHost", hostId, runningServices)
}

func (s *ControlClient) GetRunningServicesForService(serviceId string, runningServices *[]*dao.RunningService) (err error) {
	return s.rpcClient.Call("ControlPlane.GetRunningServicesForService", serviceId, runningServices)
}

func (s *ControlClient) StopRunningInstance(request dao.HostServiceRequest, unused *int) (err error) {
	return s.rpcClient.Call("ControlPlane.StopRunningInstance", request, unused)
}

func (s *ControlClient) GetRunningServices(request dao.EntityRequest, runningServices *[]*dao.RunningService) (err error) {
	return s.rpcClient.Call("ControlPlane.GetRunningServices", request, runningServices)
}

func (s *ControlClient) GetServiceState(request dao.ServiceStateRequest, state *servicestate.ServiceState) error {
	return s.rpcClient.Call("ControlPlane.GetServiceState", request, state)
}

func (s *ControlClient) GetRunningService(request dao.ServiceStateRequest, running *dao.RunningService) error {
	return s.rpcClient.Call("ControlPlane.GetRunningService", request, running)
}

func (s *ControlClient) GetServiceStates(serviceId string, states *[]*servicestate.ServiceState) (err error) {
	return s.rpcClient.Call("ControlPlane.GetServiceStates", serviceId, states)
}

func (s *ControlClient) StartService(serviceId string, hostId *string) (err error) {
	return s.rpcClient.Call("ControlPlane.StartService", serviceId, hostId)
}

func (s *ControlClient) RestartService(serviceId string, unused *int) (err error) {
	return s.rpcClient.Call("ControlPlane.RestartService", serviceId, unused)
}

func (s *ControlClient) StopService(serviceId string, unused *int) (err error) {
	return s.rpcClient.Call("ControlPlane.StopService", serviceId, unused)
}

func (s *ControlClient) UpdateServiceState(state servicestate.ServiceState, unused *int) (err error) {
	return s.rpcClient.Call("ControlPlane.UpdateServiceState", state, unused)
}

func (s *ControlClient) DeployTemplate(request dao.ServiceTemplateDeploymentRequest, tenantId *string) error {
	return s.rpcClient.Call("ControlPlane.DeployTemplate", request, tenantId)
}

func (s *ControlClient) GetServiceTemplates(unused int, serviceTemplates *map[string]*servicetemplate.ServiceTemplate) error {
	return s.rpcClient.Call("ControlPlane.GetServiceTemplates", unused, serviceTemplates)
}

func (s *ControlClient) AddServiceTemplate(serviceTemplate servicetemplate.ServiceTemplate, templateId *string) error {
	return s.rpcClient.Call("ControlPlane.AddServiceTemplate", serviceTemplate, templateId)
}

func (s *ControlClient) UpdateServiceTemplate(serviceTemplate servicetemplate.ServiceTemplate, unused *int) error {
	return s.rpcClient.Call("ControlPlane.UpdateServiceTemplate", serviceTemplate, unused)
}

func (s *ControlClient) RemoveServiceTemplate(serviceTemplateId string, unused *int) error {
	return s.rpcClient.Call("ControlPlane.RemoveServiceTemplate", serviceTemplateId, unused)
}

// Commits a container to an image and updates the DFS
func (s *ControlClient) Commit(containerId string, label *string) error {
	return s.rpcClient.Call("ControlPlane.Commit", containerId, label)
}

// Rollbacks the DFS and updates the docker images
func (s *ControlClient) Rollback(serviceId string, unused *int) error {
	return s.rpcClient.Call("ControlPlane.Rollback", serviceId, unused)
}

// Performs a DFS snapshot locally (via the host)
func (s *ControlClient) LocalSnapshot(serviceId string, label *string) error {
	return s.rpcClient.Call("ControlPlane.LocalSnapshot", serviceId, label)
}

// Performs a DFS snapshot via the scheduler
func (s *ControlClient) Snapshot(serviceId string, label *string) error {
	return s.rpcClient.Call("ControlPlane.Snapshot", serviceId, label)
}

func (s *ControlClient) DeleteSnapshot(snapshotId string, unused *int) error {
	return s.rpcClient.Call("ControlPlane.DeleteSnapshot", snapshotId, unused)
}

func (s *ControlClient) Snapshots(serviceId string, labels *[]string) error {
	return s.rpcClient.Call("ControlPlane.Snapshots", serviceId, labels)
}

func (s *ControlClient) DeleteSnapshots(serviceId string, unused *int) error {
	return s.rpcClient.Call("ControlPlane.DeleteSnapshots", serviceId, unused)
}

func (s *ControlClient) GetVolume(serviceId string, volume *volume.Volume) error {
	// WARNING: it would not make sense to call this from the CLI
	// since volume is a pointer
	return s.rpcClient.Call("ControlPlane.GetVolume", serviceId, volume)
}

func (s *ControlClient) ValidateCredentials(user dao.User, result *bool) error {
	return s.rpcClient.Call("ControlPlane.ValidateCredentials", user, result)
}

func (s *ControlClient) GetSystemUser(unused int, user *dao.User) error {
	return s.rpcClient.Call("ControlPlane.GetSystemUser", unused, user)
}

func (s *ControlClient) ReadyDFS(unused bool, unusedint *int) error {
	return s.rpcClient.Call("ControlPlane.ReadyDFS", unused, unusedint)
}

func (s *ControlClient) Backup(backupDirectory string, backupFilePath *string) error {
	return s.rpcClient.Call("ControlPlane.Backup", backupDirectory, backupFilePath)
}

func (s *ControlClient) Restore(backupFilePath string, unused *int) error {
	return s.rpcClient.Call("ControlPlane.Restore", backupFilePath, unused)
}

func (s *ControlClient) Attach(req dao.AttachRequest, unused *int) error {
	return s.rpcClient.Call("ControlPlane.Attach", req, unused)
}

func (s *ControlClient) Action(req dao.AttachRequest, unused *int) error {
	return s.rpcClient.Call("ControlPlane.Action", req, unused)
}
