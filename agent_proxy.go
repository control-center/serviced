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

	a.addContolPlaneEndpoint(*response)
	a.addContolPlaneConsumerEndpoint(*response)
	return nil
}

func (a *HostAgent) ExecAsService(execReq ExecRequest, unused *interface{}) error {
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

// addContolPlaneEndpoint adds an application endpoint mapping for the master control plane api
func (a *HostAgent) addContolPlaneEndpoint(endpoints map[string][]*dao.ApplicationEndpoint) {
	key := "tcp:8787"
	endpoint := dao.ApplicationEndpoint{}
	endpoint.ServiceId = "controlplane"
	endpoint.ContainerIp = "127.0.0.1"
	endpoint.ContainerPort = 8787
	endpoint.HostPort = 8787
	endpoint.HostIp = strings.Split(a.master, ":")[0]
	endpoint.Protocol = "tcp"
	a.addEndpoint(key, endpoint, endpoints)
}

// addContolPlaneConsumerEndpoint adds an application endpoint mapping for the master control plane api
func (a *HostAgent) addContolPlaneConsumerEndpoint(endpoints map[string][]*dao.ApplicationEndpoint) {
	key := "tcp:8444"
	endpoint := dao.ApplicationEndpoint{}
	endpoint.ServiceId = "controlplane_consumer"
	endpoint.ContainerIp = "127.0.0.1"
	endpoint.ContainerPort = 8444
	endpoint.HostPort = 8443
	endpoint.HostIp = strings.Split(a.master, ":")[0]
	endpoint.Protocol = "tcp"
	a.addEndpoint(key, endpoint, endpoints)
}

// addEndpoint adds a mapping to defined application, if a mapping does not exist this method creates the list and adds the first element
func (a *HostAgent) addEndpoint(key string, endpoint dao.ApplicationEndpoint, endpoints map[string][]*dao.ApplicationEndpoint) {
	if _, ok := endpoints[key]; !ok {
		endpoints[key] = make([]*dao.ApplicationEndpoint, 0)
	} else {
		if len(endpoints[key]) > 0 {
			glog.Warningf("Service %s has duplicate internal endpoint for key %s len(endpointList)=%d", endpoint.ServiceId, key, len(endpoints[key]))
			for _, ep := range endpoints[key] {
				glog.Warningf(" %+v", *ep)
			}
		}
	}
	endpoints[key] = append(endpoints[key], &endpoint)
}
