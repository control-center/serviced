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

package node

import (
	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/domain"
	"github.com/control-center/serviced/domain/service"
	"github.com/zenoss/glog"

	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
	"time"
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
	conn, err := net.Dial("tcp", s.addr)
	if err != nil {
		return nil, err
	}
	s.rpcClient = jsonrpc.NewClient(conn)
	return s, nil
}

func (a *LBClient) Close() error {
	return a.rpcClient.Close()
}

// Ping waits for the specified time then returns the server time
func (a *LBClient) Ping(waitFor time.Duration, timestamp *time.Time) error {
	glog.V(4).Infof("ControlPlaneAgent.Ping()")
	return a.rpcClient.Call("ControlPlaneAgent.Ping", waitFor, timestamp)
}

// SendLogMessage simply outputs the ServiceLogInfo on the serviced master
func (a *LBClient) SendLogMessage(serviceLogInfo ServiceLogInfo, _ *struct{}) error {
	glog.V(4).Infof("ControlPlaneAgent.SendLogMessage()")
	return a.rpcClient.Call("ControlPlaneAgent.SendLogMessage", serviceLogInfo, nil)
}

// GetServiceEndpoints returns a list of endpoints for the given service endpoint request.
func (a *LBClient) GetServiceEndpoints(serviceId string, endpoints *map[string][]dao.ApplicationEndpoint) error {
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

// LogHealthCheck stores a health check result.
func (a *LBClient) LogHealthCheck(result domain.HealthCheckResult, unused *int) error {
	glog.V(4).Infof("ControlPlaneAgent.LogHealthCheck()")
	return a.rpcClient.Call("ControlPlaneAgent.LogHealthCheck", result, unused)
}

// GetHealthCheck returns the health check configuration for a service, if it exists
func (a *LBClient) GetHealthCheck(req HealthCheckRequest, healthChecks *map[string]domain.HealthCheck) error {
	glog.V(4).Infof("ControlPlaneAgent.GetHealthCheck()")
	return a.rpcClient.Call("ControlPlaneAgent.GetHealthCheck", req, healthChecks)
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
