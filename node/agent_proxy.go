// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

// Package agent implements a service that runs on a serviced node. It is
// responsible for ensuring that a particular node is running the correct services
// and reporting the state and health of those services back to the master
// serviced.

// This file implements the LoadBalancer interface aspect of the host agent.
package node

import (
	"errors"
	"strconv"
	"strings"

	"github.com/zenoss/glog"

	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/domain/event"
	"github.com/control-center/serviced/domain/service"
)

// assert that the HostAgent implements the LoadBalancer interface
var _ LoadBalancer = &HostAgent{}

type ServiceLogInfo struct {
	ServiceID string
	Message   string
}

type ZkInfo struct {
	ZkDSN  string
	PoolID string
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

	a.addControlPlaneEndpoint(*response)
	a.addControlPlaneConsumerEndpoint(*response)
	a.addLogstashEndpoint(*response)
	return nil
}

func (a *HostAgent) GetService(serviceID string, response *service.Service) (err error) {
	controlClient, err := NewControlClient(a.master)
	if err != nil {
		glog.Errorf("Could not start ControlPlane client %v", err)
		return nil
	}
	defer controlClient.Close()

	err = controlClient.GetService(serviceID, response)
	if err != nil {
		return err
	}

	getSvc := func(svcID string) (service.Service, error) {
		svc := service.Service{}
		err := controlClient.GetService(svcID, &svc)
		return svc, err
	}

	findChild := func(svcID, childName string) (service.Service, error) {
		svc := service.Service{}
		err := controlClient.FindChildService(dao.FindChildRequest{svcID, childName}, &svc)
		return svc, err
	}

	return response.Evaluate(getSvc, findChild, 0)
}

func (a *HostAgent) GetServiceInstance(req ServiceInstanceRequest, response *service.Service) (err error) {
	controlClient, err := NewControlClient(a.master)
	if err != nil {
		glog.Errorf("Could not start ControlPlane client %v", err)
		return nil
	}
	defer controlClient.Close()

	err = controlClient.GetService(req.ServiceID, response)
	if err != nil {
		return err
	}

	getSvc := func(svcID string) (service.Service, error) {
		svc := service.Service{}
		err := controlClient.GetService(svcID, &svc)
		return svc, err
	}

	findChild := func(svcID, childName string) (service.Service, error) {
		svc := service.Service{}
		err := controlClient.FindChildService(dao.FindChildRequest{svcID, childName}, &svc)
		return svc, err
	}

	return response.Evaluate(getSvc, findChild, req.InstanceID)
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

// SendEvent sends a system event.
func (a *HostAgent) SendEvent(evt event.Event, unused *int) error {
	controlClient, err := NewControlClient(a.master)
	if err != nil {
		glog.Errorf("Could not start ControlPlane client %v", err)
		return err
	}
	defer controlClient.Close()
	return controlClient.SendEvent(evt, unused)
}

// addControlPlaneEndpoint adds an application endpoint mapping for the master control plane api
func (a *HostAgent) addControlPlaneEndpoint(endpoints map[string][]*dao.ApplicationEndpoint) {
	key := "tcp" + a.uiport
	endpoint := dao.ApplicationEndpoint{}
	endpoint.ServiceID = "controlplane"
	endpoint.Application = "controlplane"
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

// addControlPlaneConsumerEndpoint adds an application endpoint mapping for the master control plane api
func (a *HostAgent) addControlPlaneConsumerEndpoint(endpoints map[string][]*dao.ApplicationEndpoint) {
	key := "tcp:8444"
	endpoint := dao.ApplicationEndpoint{}
	endpoint.ServiceID = "controlplane_consumer"
	endpoint.Application = "controlplane_consumer"
	endpoint.ContainerIP = "127.0.0.1"
	endpoint.ContainerPort = 8444
	endpoint.HostPort = 8443
	endpoint.HostIP = strings.Split(a.master, ":")[0]
	endpoint.Protocol = "tcp"
	a.addEndpoint(key, endpoint, endpoints)
}

// addLogstashEndpoint adds an application endpoint mapping for the master control plane api
func (a *HostAgent) addLogstashEndpoint(endpoints map[string][]*dao.ApplicationEndpoint) {
	key := "tcp:5043"
	endpoint := dao.ApplicationEndpoint{}
	endpoint.ServiceID = "controlplane_logstash"
	endpoint.Application = "controlplane_logstash"
	endpoint.ContainerIP = "127.0.0.1"
	endpoint.ContainerPort = 5043
	endpoint.HostPort = 5043
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
func (a *HostAgent) GetHostID(_ string, hostID *string) error {
	glog.V(4).Infof("ControlPlaneAgent.GetHostID(): %s", a.hostID)
	*hostID = a.hostID
	return nil
}

// GetZkInfo returns the agent's zookeeper connection string and its poolID
func (a *HostAgent) GetZkInfo(_ string, zkInfo *ZkInfo) error {
	localDSN := a.zkClient.ConnectionString()
	zkInfo.ZkDSN = strings.Replace(localDSN, "127.0.0.1", strings.Split(a.master, ":")[0], -1)
	zkInfo.PoolID = a.poolID
	glog.V(4).Infof("ControlPlaneAgent.GetZkInfo(): %+v", zkInfo)
	return nil
}

// GetServiceBindMounts returns the service bindmounts
func (a *HostAgent) GetServiceBindMounts(serviceID string, bindmounts *map[string]string) error {
	glog.V(4).Infof("ControlPlaneAgent.GetServiceBindMounts(serviceID:%s)", serviceID)

	var tenantID string
	if err := a.GetTenantId(serviceID, &tenantID); err != nil {
		return err
	}

	var service service.Service
	if err := a.GetService(serviceID, &service); err != nil {
		return err
	}

	response := map[string]string{}
	for _, volume := range service.Volumes {
		if volume.Type != "" && volume.Type != "dfs" {
			continue
		}

		resourcePath, err := a.setupVolume(tenantID, &service, volume)
		if err != nil {
			return err
		}

		glog.V(4).Infof("retrieved bindmount resourcePath:%s containerPath:%s", resourcePath, volume.ContainerPath)
		response[resourcePath] = volume.ContainerPath
	}
	*bindmounts = response

	return nil
}
