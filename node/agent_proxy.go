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
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/control-center/serviced/rpc/master"
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

func (a *HostAgent) GetEvaluatedService(request EvaluateServiceRequest, response *EvaluateServiceResponse) (err error) {
	logger := plog.WithFields(log.Fields{
		"serviceid":  request.ServiceID,
		"instanceID": request.InstanceID,
	})

	masterClient, err := master.NewClient(a.master)
	if err != nil {
		logger.WithField("master", a.master).WithError(err).Error("Could not connect to the master")
		return err
	}
	defer masterClient.Close()

	svc, tenantID, err := masterClient.GetEvaluatedService(request.ServiceID, request.InstanceID)
	if err != nil {
		logger.WithError(err).Error("Failed to get service")
		return err
	}
	response.Service = *svc
	response.TenantID = tenantID
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

// ReportHealthStatus proxies ReportHealthStatus to the master server.
func (a *HostAgent) ReportHealthStatus(req master.HealthStatusRequest, unused *int) error {
	masterClient, err := master.NewClient(a.master)
	if err != nil {
		glog.Errorf("Could not start Control Center client: %s", err)
		return err
	}
	defer masterClient.Close()
	return masterClient.ReportHealthStatus(req.Key, req.Value, req.Expires)
}

// ReportInstanceDead proxies ReportInstanceDead to the master server.
func (a *HostAgent) ReportInstanceDead(req master.ServiceInstanceRequest, unused *int) error {
	masterClient, err := master.NewClient(a.master)
	if err != nil {
		glog.Errorf("Could not start Control Center client; %s", err)
		return err
	}
	defer masterClient.Close()
	return masterClient.ReportInstanceDead(req.ServiceID, req.InstanceID)
}

// GetHostID returns the agent's host id
func (a *HostAgent) GetHostID(_ string, hostID *string) error {
	glog.V(4).Infof("ControlCenterAgent.GetHostID(): %s", a.hostID)
	*hostID = a.hostID
	return nil
}

// GetZkInfo returns the agent's zookeeper connection string and its poolID
func (a *HostAgent) GetZkInfo(_ string, zkInfo *ZkInfo) error {
	localDSN := a.zkClient.ConnectionString()
	zkInfo.ZkDSN = strings.Replace(localDSN, "127.0.0.1", strings.Split(a.master, ":")[0], -1)
	zkInfo.PoolID = a.poolID
	glog.V(4).Infof("ControlCenterAgent.GetZkInfo(): %+v", zkInfo)
	return nil
}

// GetServiceBindMounts returns the service bindmounts
func (a *HostAgent) GetServiceBindMounts(serviceID string, bindmounts *map[string]string) error {
	glog.V(4).Infof("ControlCenterAgent.GetServiceBindMounts(serviceID:%s)", serviceID)
	*bindmounts = make(map[string]string, 0)

	var evaluatedServiceResponse EvaluateServiceResponse
	if err := a.GetEvaluatedService(EvaluateServiceRequest{ServiceID: serviceID, InstanceID: 0}, &evaluatedServiceResponse); err != nil {
		return err
	}
	service := evaluatedServiceResponse.Service
	tenantID := evaluatedServiceResponse.TenantID

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
