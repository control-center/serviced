/*******************************************************************************
* Copyright (C) Zenoss, Inc. 2013, 2014, all rights reserved.
*
* This content is made available according to terms specified in
* License.zenoss under the directory where your Zenoss product is installed.
*
*******************************************************************************/

// This file implements the LoadBalancer interface aspect of the host agent.
package serviced

import (
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/dao"

	"errors"
	"strings"
)

// assert that the HostAgent implements the LoadBalancer interface
var _ LoadBalancer = &HostAgent{}

func (a *HostAgent) GetServiceEndpoints(serviceId string, response *map[string][]*dao.ApplicationEndpoint) (err error) {
	controlClient, err := NewControlClient(a.master)
	if err != nil {
		glog.Errorf("Could not start ControlPlane client %v", err)
		return
	}
	defer controlClient.Close()

	err = controlClient.GetServiceEndpoints(serviceId, response)
	if err != nil {
		return err
	}
	endpoints := *response
	// add our internal services
	for key, endpoint := range a.getInternalServiceEndpoints() {
		if _, ok := (endpoints)[key]; ok {
			endpoints[key] = append(endpoints[key], &endpoint)
		}
		endpoints[key] = make([]*dao.ApplicationEndpoint, 0)
		endpoints[key] = append(endpoints[key], &endpoint)
	}
	return nil
}

// GetProxySnapshotQuiece blocks until there is a snapshot request to the service
func (a *HostAgent) GetProxySnapshotQuiece(serviceId string, snapshotId *string) error {
	glog.Errorf("GetProxySnapshotQuiece() Unimplemented")
	return errors.New("unimplemented")
}

// AckProxySnapshotQuiece is called by clients when the snapshot command has
// shown the service is quieced; the agent returns a response when the snapshot is complete
func (a *HostAgent) AckProxySnapshotQuiece(snapshotId string, unused *interface{}) error {
	glog.Errorf("AckProxySnapshotQuiece() Unimplemented")
	return errors.New("unimplemented")
}

// getInternalServiceEndpoints lists every internal service that we wish to expose to the containers running on this agent
func (a *HostAgent) getInternalServiceEndpoints() map[string]dao.ApplicationEndpoint {
	// master is of the form host:port, we just need the host
	master := strings.Split(a.master, ":")[0]
	return map[string]dao.ApplicationEndpoint{
		"tcp:8787": dao.ApplicationEndpoint{
			"controlplane",
			8787,
			8787,
			master,
			"127.0.0.1",
			"tcp",
		},
	}
}
