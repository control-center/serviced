// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package node

import (
	"github.com/zenoss/glog"
	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/domain/event"
	"github.com/control-center/serviced/domain/service"

	"net/rpc"
)

// A LBClient implementation.
type LBClient struct {
	addr      string
	rpcClient *rpc.Client
}

// assert that this implemenents the Agent interface
var _ LoadBalancer = &LBClient{}

// Create a new AgentClient.
func NewLBClient(addr string) (s *LBClient, err error) {
	s = new(LBClient)
	s.addr = addr
	rpcClient, err := rpc.DialHTTP("tcp", s.addr)
	s.rpcClient = rpcClient
	return s, err
}

func (a *LBClient) Close() error {
	return a.rpcClient.Close()
}

// SendLogMessage simply outputs the ServiceLogInfo on the serviced master
func (a *LBClient) SendLogMessage(serviceLogInfo ServiceLogInfo, _ *struct{}) error {
	glog.V(4).Infof("ControlPlaneAgent.SendLogMessage()")
	return a.rpcClient.Call("ControlPlaneAgent.SendLogMessage", serviceLogInfo, nil)
}

// GetServiceEndpoints returns a list of endpoints for the given service endpoint request.
func (a *LBClient) GetServiceEndpoints(serviceId string, endpoints *map[string][]*dao.ApplicationEndpoint) error {
	glog.V(4).Infof("ControlPlaneAgent.GetServiceEndpoints()")
	return a.rpcClient.Call("ControlPlaneAgent.GetServiceEndpoints", serviceId, endpoints)
}

// GetService returns a service for the given service id request.
func (a *LBClient) GetService(serviceId string, service *service.Service) error {
	glog.V(0).Infof("ControlPlaneAgent.GetService()")
	return a.rpcClient.Call("ControlPlaneAgent.GetService", serviceId, service)
}

// GetServiceInstance returns a service for the given service id request.
func (a *LBClient) GetServiceInstance(req ServiceInstanceRequest, service *service.Service) error {
	glog.V(0).Infof("ControlPlaneAgent.GetServiceInstance()")
	return a.rpcClient.Call("ControlPlaneAgent.GetServiceInstance", req, service)
}

// GetProxySnapshotQuiece blocks until there is a snapshot request to the service
func (a *LBClient) GetProxySnapshotQuiece(serviceId string, snapshotId *string) error {
	glog.V(4).Infof("ControlPlaneAgent.GetProxySnapshotQuiece()")
	return a.rpcClient.Call("ControlPlaneAgent.GetProxySnapshotQuiece", serviceId, snapshotId)
}

// AckProxySnapshotQuiece is called by clients when the snapshot command has
// shown the service is quieced; the agent returns a response when the snapshot is complete
func (a *LBClient) AckProxySnapshotQuiece(snapshotId string, unused *interface{}) error {
	glog.V(4).Infof("ControlPlaneAgent.AckProxySnapshotQuiece()")
	return a.rpcClient.Call("ControlPlaneAgent.AckProxySnapshotQuiece", snapshotId, unused)
}

// GetTenantId return's the service's tenant id
func (a *LBClient) GetTenantId(serviceId string, tenantId *string) error {
	glog.V(4).Infof("ControlPlaneAgent.GetTenantId()")
	return a.rpcClient.Call("ControlPlaneAgent.GetTenantId", serviceId, tenantId)
}

// SendEvent sends a system event.
func (a *LBClient) SendEvent(evt event.Event, unused *int) error {
	glog.V(4).Infof("ControlPlaneAgent.SendEvent()")
	return a.rpcClient.Call("ControlPlaneAgent.SendEvent", evt, unused)
}

// GetHostID returns the agent's host id
func (a *LBClient) GetHostID(hostID *string) error {
	glog.V(4).Infof("ControlPlaneAgent.GetHostID()")
	return a.rpcClient.Call("ControlPlaneAgent.GetHostID", "na", hostID)
}

// GetZkInfo returns the agent's zookeeper connection string
func (a *LBClient) GetZkInfo(zkInfo *ZkInfo) error {
	glog.V(4).Infof("ControlPlaneAgent.GetZkInfo()")
	return a.rpcClient.Call("ControlPlaneAgent.GetZkInfo", "na", zkInfo)
}

// GetServiceBindMounts returns the service
func (a *LBClient) GetServiceBindMounts(serviceID string, bindmounts *map[string]string) error {
	glog.V(4).Infof("ControlPlaneAgent.GetServiceBindMounts(serviceID:%s)", serviceID)
	return a.rpcClient.Call("ControlPlaneAgent.GetServiceBindMounts", serviceID, bindmounts)
}
