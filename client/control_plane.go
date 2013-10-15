/*******************************************************************************
* Copyright (C) Zenoss, Inc. 2013, all rights reserved.
*
* This content is made available according to terms specified in
* License.zenoss under the directory where your Zenoss product is installed.
*
*******************************************************************************/

package client

import (
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
	rpcClient, err := rpc.DialHTTP("tcp", s.addr)
	s.rpcClient = rpcClient
	return s, err
}

// Return the matching hosts.
func (s *ControlClient) Close() (err error) {
	return s.rpcClient.Close()
}

// Return the matching hosts.
func (s *ControlClient) GetHosts(request dao.EntityRequest, replyHosts *map[string]*dao.Host) (err error) {
	return s.rpcClient.Call("ControlPlane.GetHosts", request, replyHosts)
}

func (s *ControlClient) AddHost(host dao.Host, unused *int) (err error) {
	return s.rpcClient.Call("ControlPlane.AddHost", host, unused)
}

func (s *ControlClient) UpdateHost(host dao.Host, unused *int) (err error) {
	return s.rpcClient.Call("ControlPlane.UpdateHost", host, unused)
}

func (s *ControlClient) RemoveHost(hostId string, unused *int) (err error) {
	return s.rpcClient.Call("ControlPlane.RemoveHost", hostId, unused)
}

func (s *ControlClient) GetServices(request dao.EntityRequest, replyServices *[]*dao.Service) (err error) {
	return s.rpcClient.Call("ControlPlane.GetServices", request, replyServices)
}

func (s *ControlClient) AddService(service dao.Service, unused *int) (err error) {
	return s.rpcClient.Call("ControlPlane.AddService", service, unused)
}

func (s *ControlClient) UpdateService(service dao.Service, unused *int) (err error) {
	return s.rpcClient.Call("ControlPlane.UpdateService", service, unused)
}

func (s *ControlClient) RemoveService(serviceId string, unused *int) (err error) {
	return s.rpcClient.Call("ControlPlane.RemoveService", serviceId, unused)
}

func (s *ControlClient) GetServicesForHost(hostId string, servicesForHost *[]*dao.Service) (err error) {
	return s.rpcClient.Call("ControlPlane.GetServicesForHost", hostId, servicesForHost)
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

func (s *ControlClient) AddResourcePool(pool dao.ResourcePool, unused *int) (err error) {
	return s.rpcClient.Call("ControlPlane.AddResourcePool", pool, unused)
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

func (s *ControlClient) AddHostToResourcePool(poolHost dao.PoolHost, unused *int) error {
	return s.rpcClient.Call("ControlPlane.AddHostToResourcePool", poolHost, unused)
}

func (s *ControlClient) RemoveHostFromResourcePool(poolHost dao.PoolHost, unused *int) error {
	return s.rpcClient.Call("ControlPlane.RemoveHostFromResourcePool", poolHost, unused)
}

func (s *ControlClient) DeployTemplate(request dao.ServiceTemplateDeploymentRequest, unused *int) error {
	return s.rpcClient.Call("ControlPlane.DeployTemplate", request, unused)
}

func (s *ControlClient) GetServiceTemplates(unused int, serviceTemplates *[]*dao.ServiceTemplate) error {
	return s.rpcClient.Call("ControlPlane.GetServiceTemplates", unused, serviceTemplates)
}

func (s *ControlClient) AddServiceTemplate(serviceTemplate dao.ServiceTemplate, unused *int) error {
	return s.rpcClient.Call("ControlPlane.AddServiceTemplate", serviceTemplate, unused)
}

func (s *ControlClient) UpdateServiceTemplate(serviceTemplate dao.ServiceTemplate, unused *int) error {
	return s.rpcClient.Call("ControlPlane.UpdateServiceTemplate", serviceTemplate, unused)
}

func (s *ControlClient) RemoveServiceTemplate(serviceTemplateId string, unused *int) error {
	return s.rpcClient.Call("ControlPlane.RemoveServiceTemplate", serviceTemplateId, unused)
}
