package serviced

import (
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/dao"
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

// SendLogMessage
func (a *LBClient) SendLogMessage(serviceLogInfo ServiceLogInfo, _ *struct{}) error {
	glog.V(4).Infof("ControlPlaneAgent.SendLogMessage()")
	return a.rpcClient.Call("ControlPlaneAgent.SendLogMessage", serviceLogInfo, nil)
}

// GetServiceEndpoints returns a list of endpoints for the given service endpoint request.
func (a *LBClient) GetServiceEndpoints(serviceId string, endpoints *map[string][]*dao.ApplicationEndpoint) error {
	glog.V(4).Infof("ControlPlaneAgent.GetServiceEndpoints()")
	return a.rpcClient.Call("ControlPlaneAgent.GetServiceEndpoints", serviceId, endpoints)
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
