package proxy

import (
	"github.com/zenoss/serviced"
	"log"
	"net/rpc"
)

// A LBClient implementation.
type LBClient struct {
	addr      string
	rpcClient *rpc.Client
}

// assert that this implemenents the Agent interface
var _ serviced.LoadBalancer = &LBClient{}

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

// Retrieve a list of endpoints for the given service endpoint request.
func (a *LBClient) GetServiceEndpoints(serviceId string, endpoints *map[string][]*serviced.ApplicationEndpoint) error {
	log.Printf("Client.GetServiceEndpoints()\n")
	return a.rpcClient.Call("LoadBalancer.GetServiceEndpoints", serviceId, endpoints)
}
