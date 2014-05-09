package serviced

import (
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/dao"
	"github.com/zenoss/serviced/domain"
	"github.com/zenoss/serviced/domain/service"

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

// LogHealthCheck TODO add comment.
func (a *LBClient) LogHealthCheck(result domain.HealthCheckResult, _ *struct{}) error {
	glog.V(4).Infof("ControlPlaneAgent.LogHealthCheck()")
	return a.rpcClient.Call("ControlPlaneAgent.LogHealthCheck", result, nil)
}

// GetHealthCheck returns the health check configuration for a service, if it exists
func (a *LBClient) GetHealthCheck(serviceId string, healthChecks *map[string]domain.HealthCheck) error {
	glog.V(4).Infof("ControlPlaneAgent.GetHealthCheck()")
	return a.rpcClient.Call("ControlPlaneAgent.GetHealthCheck", serviceId, healthChecks)
}
