// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

// Package agent implements a service that runs on a serviced node. It is
// responsible for ensuring that a particular node is running the correct services
// and reporting the state and health of those services back to the master
// serviced.

package serviced

import (
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/dao"
	"net/rpc"
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

// Return the matching hosts.
func (s *ControlClient) GetHosts(request dao.EntityRequest, replyHosts *map[string]*dao.Host) (err error) {
	return s.rpcClient.Call("ControlPlane.GetHosts", request, replyHosts)
}

func (s *ControlClient) AddHost(host dao.Host, hostId *string) (err error) {
	return s.rpcClient.Call("ControlPlane.AddHost", host, hostId)
}

func (s *ControlClient) UpdateHost(host dao.Host, unused *int) (err error) {
	return s.rpcClient.Call("ControlPlane.UpdateHost", host, unused)
}

func (s *ControlClient) GetHost(hostId string, host *dao.Host) (err error) {
	return s.rpcClient.Call("ControlPlane.GetHost", hostId, host)
}

func (s *ControlClient) RemoveHost(hostId string, unused *int) (err error) {
	return s.rpcClient.Call("ControlPlane.RemoveHost", hostId, unused)
}

func (s *ControlClient) GetServices(request dao.EntityRequest, replyServices *[]*dao.Service) (err error) {
	return s.rpcClient.Call("ControlPlane.GetServices", request, replyServices)
}

func (s *ControlClient) GetTaggedServices(request dao.EntityRequest, replyServices *[]*dao.Service) (err error) {
	return s.rpcClient.Call("ControlPlane.GetTaggedServices", request, replyServices)
}

func (s *ControlClient) GetService(serviceId string, service *dao.Service) (err error) {
	return s.rpcClient.Call("ControlPlane.GetService", serviceId, &service)
}

func (s *ControlClient) GetTenantId(serviceId string, tenantId *string) (err error) {
	return s.rpcClient.Call("ControlPlane.GetTenantId", serviceId, tenantId)
}

func (s *ControlClient) AddService(service dao.Service, serviceId *string) (err error) {
	return s.rpcClient.Call("ControlPlane.AddService", service, serviceId)
}

func (s *ControlClient) UpdateService(service dao.Service, unused *int) (err error) {
	return s.rpcClient.Call("ControlPlane.UpdateService", service, unused)
}

func (s *ControlClient) RemoveService(serviceId string, unused *int) (err error) {
	return s.rpcClient.Call("ControlPlane.RemoveService", serviceId, unused)
}

func (s *ControlClient) AddServiceDeployment(deployment dao.ServiceDeployment, unused *int) (err error) {
	return s.rpcClient.Call("ControlPlane.AddServiceDeployment", deployment, unused)
}

func (s *ControlClient) AssignIPs(assignmentRequest dao.AssignmentRequest, _ *struct{}) (err error) {
	return s.rpcClient.Call("ControlPlane.AssignIPs", assignmentRequest, nil)
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

func (s *ControlClient) GetServiceState(request dao.ServiceStateRequest, state *dao.ServiceState) error {
	return s.rpcClient.Call("ControlPlane.GetServiceState", request, state)
}

func (s *ControlClient) GetRunningService(request dao.ServiceStateRequest, running *dao.RunningService) error {
	return s.rpcClient.Call("ControlPlane.GetRunningService", request, running)
}

func (s *ControlClient) GetServiceStates(serviceId string, states *[]*dao.ServiceState) (err error) {
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

func (s *ControlClient) UpdateServiceState(state dao.ServiceState, unused *int) (err error) {
	return s.rpcClient.Call("ControlPlane.UpdateServiceState", state, unused)
}

func (s *ControlClient) GetResourcePools(request dao.EntityRequest, pools *map[string]*dao.ResourcePool) (err error) {
	return s.rpcClient.Call("ControlPlane.GetResourcePools", request, pools)
}

func (s *ControlClient) AddResourcePool(pool dao.ResourcePool, poolId *string) (err error) {
	return s.rpcClient.Call("ControlPlane.AddResourcePool", pool, poolId)
}

func (s *ControlClient) UpdateResourcePool(pool dao.ResourcePool, unused *int) (err error) {
	return s.rpcClient.Call("ControlPlane.UpdateResourcePool", pool, unused)
}

func (s *ControlClient) RemoveResourcePool(poolId string, unused *int) (err error) {
	return s.rpcClient.Call("ControlPlane.RemoveResourcePool", poolId, unused)
}

func (s *ControlClient) GetHostsForResourcePool(poolId string, poolHosts *[]*dao.PoolHost) (err error) {
	return s.rpcClient.Call("ControlPlane.GetHostsForResourcePool", poolId, poolHosts)
}

func (s *ControlClient) GetPoolHostIPInfo(poolId string, poolsHostsIpInfo *map[string][]dao.HostIPResource) (err error) {
	return s.rpcClient.Call("ControlPlane.GetPoolHostIPInfo", poolId, poolsHostsIpInfo)
}

func (s *ControlClient) AddHostToResourcePool(poolHost dao.PoolHost, unused *int) error {
	return s.rpcClient.Call("ControlPlane.AddHostToResourcePool", poolHost, unused)
}

func (s *ControlClient) RemoveHostFromResourcePool(poolHost dao.PoolHost, unused *int) error {
	return s.rpcClient.Call("ControlPlane.RemoveHostFromResourcePool", poolHost, unused)
}

func (s *ControlClient) DeployTemplate(request dao.ServiceTemplateDeploymentRequest, unused *int) error {
	return s.rpcClient.Call("ControlPlane.DeployTemplate", request, unused)
}

func (s *ControlClient) GetServiceTemplates(unused int, serviceTemplates *map[string]*dao.ServiceTemplate) error {
	return s.rpcClient.Call("ControlPlane.GetServiceTemplates", unused, serviceTemplates)
}

func (s *ControlClient) AddServiceTemplate(serviceTemplate dao.ServiceTemplate, templateId *string) error {
	return s.rpcClient.Call("ControlPlane.AddServiceTemplate", serviceTemplate, templateId)
}

func (s *ControlClient) UpdateServiceTemplate(serviceTemplate dao.ServiceTemplate, unused *int) error {
	return s.rpcClient.Call("ControlPlane.UpdateServiceTemplate", serviceTemplate, unused)
}

func (s *ControlClient) RemoveServiceTemplate(serviceTemplateId string, unused *int) error {
	return s.rpcClient.Call("ControlPlane.RemoveServiceTemplate", serviceTemplateId, unused)
}

func (s *ControlClient) StartShell(service dao.Service, unused *int) error {
	return s.rpcClient.Call("ControlPlane.StartShell", service, unused)
}

func (s *ControlClient) ExecuteShell(service dao.Service, command *string) error {
	return s.rpcClient.Call("ControlPlane.ExecuteShell", service, command)
}

func (s *ControlClient) ShowCommands(service dao.Service, unused *int) error {
	return s.rpcClient.Call("ControlPlane.ShowCommands", service, unused)
}

func (s *ControlClient) Rollback(serviceId string, unused *int) error {
	return s.rpcClient.Call("ControlPlane.Rollback", serviceId, unused)
}

func (s *ControlClient) Snapshot(serviceId string, label *string) error {
	return s.rpcClient.Call("ControlPlane.Snapshot", serviceId, label)
}

func (s *ControlClient) DeleteSnapshot(snapshotId string, unused *int) error {
	return s.rpcClient.Call("ControlPlane.DeleteSnapshot", snapshotId, unused)
}

func (s *ControlClient) Snapshots(serviceId string, labels *[]string) error {
	return s.rpcClient.Call("ControlPlane.Snapshots", serviceId, labels)
}

func (s *ControlClient) Get(service dao.Service, file *string) error {
	return s.rpcClient.Call("ControlPlane.Get", service, file)
}

func (s *ControlClient) Send(service dao.Service, files *[]string) error {
	return s.rpcClient.Call("ControlPlane.Send", service, files)
}
