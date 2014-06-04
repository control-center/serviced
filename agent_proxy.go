// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

// Package agent implements a service that runs on a serviced node. It is
// responsible for ensuring that a particular node is running the correct services
// and reporting the state and health of those services back to the master
// serviced.

// This file implements the LoadBalancer interface aspect of the host agent.
package serviced

import (
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/dao"
	"github.com/zenoss/serviced/domain"

	"errors"
	"github.com/zenoss/serviced/domain/service"
	"strconv"
	"strings"
)

// assert that the HostAgent implements the LoadBalancer interface
var _ LoadBalancer = &HostAgent{}

type ServiceLogInfo struct {
	ServiceID string
	Message   string
}

func (a *HostAgent) SendLogMessage(serviceLogInfo ServiceLogInfo, _ *struct{}) (err error) {
	glog.Infof("Service: %v message: %v", serviceLogInfo.ServiceID, serviceLogInfo.Message)
	return nil
}

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

func (a *HostAgent) GetService(serviceId string, response *service.Service) (err error) {
	controlClient, err := NewControlClient(a.master)
	if err != nil {
		glog.Errorf("Could not start ControlPlane client %v", err)
		return
	}
	defer controlClient.Close()

	err = controlClient.GetService(serviceId, response)
	if err != nil {
		return err
	}

	return nil
}

// Call the master's to retrieve its tenant id
func (a *HostAgent) GetTenantId(serviceId string, tenantId *string) error {
	client, err := NewControlClient(a.master)
	if err != nil {
		glog.Errorf("Could not start ControlPlane client %v", err)
		return err
	}
	defer client.Close()
	return client.GetTenantId(serviceId, tenantId)
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

// GetHealthCheck returns the health check configuration for a service, if it exists
func (a *HostAgent) GetHealthCheck(serviceId string, healthChecks *map[string]domain.HealthCheck) error {
	glog.V(4).Infof("ControlPlaneAgent.GetHealthCheck()")
	controlClient, err := NewControlClient(a.master)
	if err != nil {
		glog.Errorf("Could not start ControlPlane client %v", err)
		return err
	}
	defer controlClient.Close()

	var svc service.Service
	err = controlClient.GetService(serviceId, &svc)
	if err != nil {
		return err
	}
	*healthChecks = svc.HealthChecks
	return nil
}

// LogHealthCheck proxies RegisterHealthCheck.
func (a *HostAgent) LogHealthCheck(result domain.HealthCheckResult, unused *int) error {
	controlClient, err := NewControlClient(a.master)
	if err != nil {
		glog.Errorf("Could not start ControlPlane client %v", err)
		return err
	}
	defer controlClient.Close()
	err = controlClient.LogHealthCheck(result, unused)
	return err
}

// addContolPlaneEndpoint adds an application endpoint mapping for the master control plane api
func (a *HostAgent) addContolPlaneEndpoint(endpoints map[string][]*dao.ApplicationEndpoint) {
	key := "tcp" + a.uiport
	endpoint := dao.ApplicationEndpoint{}
	endpoint.ServiceID = "controlplane"
	endpoint.ContainerIP = "127.0.0.1"
	port, err := strconv.Atoi(a.uiport[1:])
	if err != nil {
		glog.Errorf("Unable to interpret ui port.")
		return
	}
	endpoint.ContainerPort = uint16(port)
	endpoint.HostPort = uint16(port)
	endpoint.HostIP = strings.Split(a.master, ":")[0]
	endpoint.Protocol = "tcp"
	a.addEndpoint(key, endpoint, endpoints)
}

// addContolPlaneConsumerEndpoint adds an application endpoint mapping for the master control plane api
func (a *HostAgent) addContolPlaneConsumerEndpoint(endpoints map[string][]*dao.ApplicationEndpoint) {
	key := "tcp:8444"
	endpoint := dao.ApplicationEndpoint{}
	endpoint.ServiceID = "controlplane_consumer"
	endpoint.ContainerIP = "127.0.0.1"
	endpoint.ContainerPort = 8444
	endpoint.HostPort = 8443
	endpoint.HostIP = strings.Split(a.master, ":")[0]
	endpoint.Protocol = "tcp"
	a.addEndpoint(key, endpoint, endpoints)
}

// addEndpoint adds a mapping to defined application, if a mapping does not exist this method creates the list and adds the first element
func (a *HostAgent) addEndpoint(key string, endpoint dao.ApplicationEndpoint, endpoints map[string][]*dao.ApplicationEndpoint) {
	if _, ok := endpoints[key]; !ok {
		endpoints[key] = make([]*dao.ApplicationEndpoint, 0)
	} else {
		if len(endpoints[key]) > 0 {
			glog.Warningf("Service %s has duplicate internal endpoint for key %s len(endpointList)=%d", endpoint.ServiceID, key, len(endpoints[key]))
			for _, ep := range endpoints[key] {
				glog.Warningf(" %+v", *ep)
			}
		}
	}
	endpoints[key] = append(endpoints[key], &endpoint)
}

// GetHostID returns the agent's host id
func (a *HostAgent) GetHostID(string, hostID *string) error {
	glog.V(4).Infof("ControlPlaneAgent.GetHostID(): %s", a.hostID)
	*hostID = a.hostID
	return nil
}

// GetZkDSN returns the agent's zookeeper connection string
func (a *HostAgent) GetZkDSN(string, dsn *string) error {
	localDSN := a.zkClient.ConnectionString()
	*dsn = strings.Replace(localDSN, "127.0.0.1", strings.Split(a.master, ":")[0], -1)
	glog.V(4).Infof("ControlPlaneAgent.GetZkDSN(): %s", *dsn)
	return nil
}
