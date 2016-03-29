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
	"time"

	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/domain"
	"github.com/control-center/serviced/domain/applicationendpoint"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/health"
	"github.com/zenoss/glog"
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

func (a *HostAgent) Ping(waitFor time.Duration, timestamp *time.Time) error {
	*timestamp = <-time.After(waitFor)
	return nil
}

func (a *HostAgent) GetServiceEndpoints(serviceId string, response *map[string][]applicationendpoint.ApplicationEndpoint) (err error) {
	myList := make(map[string][]applicationendpoint.ApplicationEndpoint)

	a.addControlPlaneEndpoint(myList)
	a.addControlPlaneConsumerEndpoint(myList)
	a.addLogstashEndpoint(myList)

	*response = myList
	return nil
}

func (a *HostAgent) GetService(serviceID string, response *service.Service) (err error) {
	*response = service.Service{}

	controlClient, err := NewControlClient(a.master)
	if err != nil {
		glog.Errorf("Could not start ControlPlane client %v", err)
		return nil
	}
	defer controlClient.Close()

	err = controlClient.GetService(serviceID, response)
	if response == nil {
		*response = service.Service{}
	}
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
	*response = service.Service{}

	controlClient, err := NewControlClient(a.master)
	if err != nil {
		glog.Errorf("Could not start ControlPlane client %v", err)
		return nil
	}
	defer controlClient.Close()

	err = controlClient.GetService(req.ServiceID, response)
	if response == nil {
		*response = service.Service{}
	}
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

// GetHealthCheck returns the health check configuration for a service, if it exists
func (a *HostAgent) GetHealthCheck(req HealthCheckRequest, healthChecks *map[string]health.HealthCheck) error {
	glog.V(4).Infof("ControlPlaneAgent.GetHealthCheck()")
	*healthChecks = make(map[string]health.HealthCheck, 0)

	controlClient, err := NewControlClient(a.master)
	if err != nil {
		glog.Errorf("Could not start ControlPlane client %v", err)
		return err
	}
	defer controlClient.Close()

	var svc service.Service
	err = controlClient.GetService(req.ServiceID, &svc)
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
	svc.EvaluateHealthCheckTemplate(getSvc, findChild, req.InstanceID)
	if svc.HealthChecks != nil {
		*healthChecks = svc.HealthChecks
	}
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

// ReportHealthStatus proxies ReportHealthStatus to the master server.
func (a *HostAgent) ReportHealthStatus(req dao.HealthStatusRequest, unused *int) error {
	client, err := NewControlClient(a.master)
	if err != nil {
		glog.Errorf("Could not start Control Center client: %s", err)
		return err
	}
	defer client.Close()
	return client.ReportHealthStatus(req, unused)
}

// ReportInstanceDead proxies ReportInstanceDead to the master server.
func (a *HostAgent) ReportInstanceDead(req dao.ServiceInstanceRequest, unused *int) error {
	client, err := NewControlClient(a.master)
	if err != nil {
		glog.Errorf("Could not start Control Center client; %s", err)
		return err
	}
	defer client.Close()
	return client.ReportInstanceDead(req, unused)
}

// addControlPlaneEndpoint adds an application endpoint mapping for the master control center api
func (a *HostAgent) addControlPlaneEndpoint(endpoints map[string][]applicationendpoint.ApplicationEndpoint) {
	key := "tcp" + a.uiport
	endpoint := applicationendpoint.ApplicationEndpoint{}
	endpoint.ServiceID = "controlplane"
	endpoint.Application = "controlplane"
	endpoint.ContainerIP = "127.0.0.1"
	port, err := strconv.Atoi(a.uiport[1:])
	if err != nil {
		glog.Errorf("Unable to interpret ui port.")
		return
	}
	endpoint.ContainerPort = uint16(port)
	endpoint.ProxyPort = uint16(port)
	endpoint.HostPort = uint16(port)
	endpoint.HostIP = strings.Split(a.master, ":")[0]
	endpoint.Protocol = "tcp"
	a.addEndpoint(key, endpoint, endpoints)
}

// addControlPlaneConsumerEndpoint adds an application endpoint mapping for the master control center api
func (a *HostAgent) addControlPlaneConsumerEndpoint(endpoints map[string][]applicationendpoint.ApplicationEndpoint) {
	key := "tcp:8444"
	endpoint := applicationendpoint.ApplicationEndpoint{}
	endpoint.ServiceID = "controlplane_consumer"
	endpoint.Application = "controlplane_consumer"
	endpoint.ContainerIP = "127.0.0.1"
	endpoint.ContainerPort = 8443
	endpoint.ProxyPort = 8444
	endpoint.HostPort = 8443
	endpoint.HostIP = strings.Split(a.master, ":")[0]
	endpoint.Protocol = "tcp"
	a.addEndpoint(key, endpoint, endpoints)
}

// addLogstashEndpoint adds an application endpoint mapping for the master control center api
func (a *HostAgent) addLogstashEndpoint(endpoints map[string][]applicationendpoint.ApplicationEndpoint) {
	tcp_endpoint := applicationendpoint.ApplicationEndpoint{
		ServiceID:     "controlplane_logstash_tcp",
		Application:   "controlplane_logstash_tcp",
		ContainerIP:   "127.0.0.1",
		ContainerPort: 5042,
		HostPort:      5042,
		ProxyPort:     5042,
		HostIP:        strings.Split(a.master, ":")[0],
		Protocol:      "tcp",
	}
	a.addEndpoint("tcp:5042", tcp_endpoint, endpoints)

	ljack_endpoint := applicationendpoint.ApplicationEndpoint{
		ServiceID:     "controlplane_logstash_lumberjack",
		Application:   "controlplane_logstash_lumberjack",
		ContainerIP:   "127.0.0.1",
		ContainerPort: 5043,
		HostPort:      5043,
		ProxyPort:     5043,
		HostIP:        strings.Split(a.master, ":")[0],
		Protocol:      "tcp",
	}
	a.addEndpoint("tcp:5043", ljack_endpoint, endpoints)
}

// addEndpoint adds a mapping to defined application, if a mapping does not exist this method creates the list and adds the first element
func (a *HostAgent) addEndpoint(key string, endpoint applicationendpoint.ApplicationEndpoint, endpoints map[string][]applicationendpoint.ApplicationEndpoint) {
	if _, ok := endpoints[key]; !ok {
		endpoints[key] = make([]applicationendpoint.ApplicationEndpoint, 0)
	} else {
		if len(endpoints[key]) > 0 {
			glog.Warningf("Service %s has duplicate internal endpoint for key %s len(endpointList)=%d", endpoint.ServiceID, key, len(endpoints[key]))
			for _, ep := range endpoints[key] {
				glog.Warningf(" %+v", ep)
			}
		}
	}
	endpoints[key] = append(endpoints[key], endpoint)
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
	*bindmounts = make(map[string]string, 0)

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
