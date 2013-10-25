/*******************************************************************************
* Copyright (C) Zenoss, Inc. 2013, all rights reserved.
*
* This content is made available according to terms specified in
* License.zenoss under the directory where your Zenoss product is installed.
*
*******************************************************************************/

package client

import (
	"github.com/zenoss/serviced"
	"github.com/zenoss/glog"
	"net/rpc"
)

// A serviced client.
type ControlClient struct {
	addr      string
	rpcClient *rpc.Client
}

// Ensure that ControlClient implements the ControlPlane interface.
var _ serviced.ControlPlane = &ControlClient{}

// Create a new ControlClient.
func NewControlClient(addr string) (s *ControlClient, err error) {
	s = new(ControlClient)
	s.addr = addr
	glog.Infof("Connecting to %s", addr)
	rpcClient, err := rpc.DialHTTP("tcp", s.addr)
	s.rpcClient = rpcClient
	return s, err
}

// Return the matching hosts.
func (s *ControlClient) Close() (err error) {
	return s.rpcClient.Close()
}

// Return the matching hosts.
func (s *ControlClient) GetHosts(request serviced.EntityRequest, replyHosts *map[string]*serviced.Host) (err error) {
	return s.rpcClient.Call("ControlPlane.GetHosts", request, replyHosts)
}

func (s *ControlClient) AddHost(host serviced.Host, unused *int) (err error) {
	return s.rpcClient.Call("ControlPlane.AddHost", host, unused)
}

func (s *ControlClient) UpdateHost(host serviced.Host, unused *int) (err error) {
	return s.rpcClient.Call("ControlPlane.UpdateHost", host, unused)
}

func (s *ControlClient) RemoveHost(hostId string, unused *int) (err error) {
	return s.rpcClient.Call("ControlPlane.RemoveHost", hostId, unused)
}

func (s *ControlClient) GetServices(request serviced.EntityRequest, replyServices *[]*serviced.Service) (err error) {
	return s.rpcClient.Call("ControlPlane.GetServices", request, replyServices)
}

func (s *ControlClient) AddService(service serviced.Service, unused *int) (err error) {
	return s.rpcClient.Call("ControlPlane.AddService", service, unused)
}

func (s *ControlClient) UpdateService(service serviced.Service, unused *int) (err error) {
	return s.rpcClient.Call("ControlPlane.UpdateService", service, unused)
}

func (s *ControlClient) RemoveService(serviceId string, unused *int) (err error) {
	return s.rpcClient.Call("ControlPlane.RemoveService", serviceId, unused)
}

func (s *ControlClient) GetServicesForHost(hostId string, servicesForHost *[]*serviced.Service) (err error) {
	return s.rpcClient.Call("ControlPlane.GetServicesForHost", hostId, servicesForHost)
}

func (s *ControlClient) GetServiceLogs(serviceId string, logs *string) error {
	return s.rpcClient.Call("ControlPlane.GetServiceLogs", serviceId, logs)
}

func (s *ControlClient) GetServiceStateLogs(serviceStateId string, logs *string) error {
	return s.rpcClient.Call("ControlPlane.GetServiceStateLogs", serviceStateId, logs)
}

func (s *ControlClient) GetRunningServicesForHost(hostId string, runningServices *[]*serviced.RunningService) (err error) {
	return s.rpcClient.Call("ControlPlane.GetRunningServicesForHost", hostId, runningServices)
}

func (s *ControlClient) GetRunningServices(request serviced.EntityRequest, runningServices *[]*serviced.RunningService) (err error) {
	return s.rpcClient.Call("ControlPlane.GetRunningServices", request, runningServices)
}

func (s *ControlClient) GetServiceStates(serviceId string, states *[]*serviced.ServiceState) (err error) {
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

func (s *ControlClient) UpdateServiceState(state serviced.ServiceState, unused *int) (err error) {
	return s.rpcClient.Call("ControlPlane.UpdateServiceState", state, unused)
}

func (s *ControlClient) GetResourcePools(request serviced.EntityRequest, pools *map[string]*serviced.ResourcePool) (err error) {
	return s.rpcClient.Call("ControlPlane.GetResourcePools", request, pools)
}

func (s *ControlClient) AddResourcePool(pool serviced.ResourcePool, unused *int) (err error) {
	return s.rpcClient.Call("ControlPlane.AddResourcePool", pool, unused)
}

func (s *ControlClient) UpdateResourcePool(pool serviced.ResourcePool, unused *int) (err error) {
	return s.rpcClient.Call("ControlPlane.UpdateResourcePool", pool, unused)
}

func (s *ControlClient) RemoveResourcePool(poolId string, unused *int) (err error) {
	return s.rpcClient.Call("ControlPlane.RemoveResourcePool", poolId, unused)
}

func (s *ControlClient) GetHostsForResourcePool(poolId string, poolHosts *[]*serviced.PoolHost) (err error) {
	return s.rpcClient.Call("ControlPlane.GetHostsForResourcePool", poolId, poolHosts)
}

func (s *ControlClient) AddHostToResourcePool(poolHost serviced.PoolHost, unused *int) error {
	return s.rpcClient.Call("ControlPlane.AddHostToResourcePool", poolHost, unused)
}

func (s *ControlClient) RemoveHostFromResourcePool(poolHost serviced.PoolHost, unused *int) error {
	return s.rpcClient.Call("ControlPlane.RemoveHostFromResourcePool", poolHost, unused)
}

func (s *ControlClient) DeployTemplate(request serviced.ServiceTemplateDeploymentRequest, unused *int) error {
	return s.rpcClient.Call("ControlPlane.DeployTemplate", request, unused)
}

func (s *ControlClient) GetServiceTemplates(unused int, serviceTemplates *map[string]*serviced.ServiceTemplate) error {
	return s.rpcClient.Call("ControlPlane.GetServiceTemplates", unused, serviceTemplates)
}

func (s *ControlClient) AddServiceTemplate(serviceTemplate serviced.ServiceTemplate, unused *int) error {
	return s.rpcClient.Call("ControlPlane.AddServiceTemplate", serviceTemplate, unused)
}

func (s *ControlClient) UpdateServiceTemplate(serviceTemplate serviced.ServiceTemplate, unused *int) error {
	return s.rpcClient.Call("ControlPlane.UpdateServiceTemplate", serviceTemplate, unused)
}

func (s *ControlClient) RemoveServiceTemplate(serviceTemplateId string, unused *int) error {
	return s.rpcClient.Call("ControlPlane.RemoveServiceTemplate", serviceTemplateId, unused)
}
