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

package elasticsearch

import (
	"fmt"
	"strconv"

	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/facade"
	"github.com/control-center/serviced/metrics"
	"github.com/control-center/serviced/zzk"
	zkdocker "github.com/control-center/serviced/zzk/docker"
	"github.com/zenoss/elastigo/api"
	"github.com/zenoss/glog"
)

const (
	DOCKER_ENDPOINT string = "unix:///var/run/docker.sock"
)

//assert interface
var _ dao.ControlPlane = &ControlPlaneDao{}

type ControlPlaneDao struct {
	hostName     string
	port         int
	rpcPort      int
	facade       *facade.Facade
	metricClient *metrics.Client
	backupsPath  string
}

func serviceGetter(ctx datastore.Context, f *facade.Facade) service.GetService {
	return func(svcID string) (service.Service, error) {
		svc, err := f.GetService(ctx, svcID)
		if err != nil {
			return service.Service{}, err
		}
		return *svc, nil
	}
}

func childFinder(ctx datastore.Context, f *facade.Facade) service.FindChildService {
	return func(svcID, childName string) (service.Service, error) {
		svc, err := f.FindChildService(ctx, svcID, childName)
		if err != nil {
			return service.Service{}, err
		} else if svc == nil {
			return service.Service{}, fmt.Errorf("no service found")
		}
		return *svc, nil
	}
}

func (this *ControlPlaneDao) Action(request dao.AttachRequest, unused *int) error {
	ctx := datastore.Get()
	svc, err := this.facade.GetService(ctx, request.Running.ServiceID)
	if err != nil {
		return err
	}

	var command []string
	if request.Command == "" {
		return fmt.Errorf("missing command")
	}

	if err := svc.EvaluateActionsTemplate(serviceGetter(ctx, this.facade), childFinder(ctx, this.facade), request.Running.InstanceID); err != nil {
		return err
	}

	action, ok := svc.Actions[request.Command]
	if !ok {
		return fmt.Errorf("action not found for service %s: %s", svc.ID, request.Command)
	}

	command = append([]string{action}, request.Args...)
	req := zkdocker.Action{
		HostID:   request.Running.HostID,
		DockerID: request.Running.DockerID,
		Command:  command,
	}

	conn, err := zzk.GetLocalConnection(zzk.GeneratePoolPath(svc.PoolID))
	if err != nil {
		return err
	}

	_, err = zkdocker.SendAction(conn, &req)
	return err
}

// Create a elastic search control center data access object
func NewControlPlaneDao(hostName string, port int, rpcPort int) (*ControlPlaneDao, error) {
	glog.V(0).Infof("Opening ElasticSearch ControlPlane Dao: hostName=%s, port=%d", hostName, port)
	api.Domain = hostName
	api.Port = strconv.Itoa(port)

	dao := &ControlPlaneDao{
		hostName: hostName,
		port:     port,
		rpcPort:  rpcPort,
	}

	return dao, nil
}

func NewControlSvc(hostName string, port int, facade *facade.Facade, backupsPath string, rpcPort int) (*ControlPlaneDao, error) {
	glog.V(2).Info("calling NewControlSvc()")
	defer glog.V(2).Info("leaving NewControlSvc()")

	s, err := NewControlPlaneDao(hostName, port, rpcPort)
	if err != nil {
		return nil, err
	}
	s.backupsPath = backupsPath

	//Used to bridge old to new
	s.facade = facade

	// create the account credentials
	if err = createSystemUser(s); err != nil {
		return nil, err
	}

	// initialize the metrics client
	metricClient, err := metrics.NewClient(fmt.Sprintf("http://%s:8888", hostName))
	if err != nil {
		return nil, err
	}
	s.metricClient = metricClient

	return s, nil
}
