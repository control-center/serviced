// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

// Package agent implements a service that runs on a serviced node. It is
// responsible for ensuring that a particular node is running the correct services
// and reporting the state and health of those services back to the master
// serviced.

package elasticsearch

import (
	"fmt"
	"strconv"
	"sync"

	"github.com/zenoss/elastigo/api"
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/dao"
	"github.com/zenoss/serviced/datastore"
	"github.com/zenoss/serviced/dfs"
	"github.com/zenoss/serviced/domain/service"
	"github.com/zenoss/serviced/facade"
	"github.com/zenoss/serviced/zzk"
	zkdocker "github.com/zenoss/serviced/zzk/docker"
)

const (
	DOCKER_ENDPOINT string = "unix:///var/run/docker.sock"
)

//assert interface
var _ dao.ControlPlane = &ControlPlaneDao{}

type ControlPlaneDao struct {
	hostName       string
	port           int
	varpath        string
	vfs            string
	dfs            *dfs.DistributedFileSystem
	facade         *facade.Facade
	dockerRegistry string
	backupLock     sync.RWMutex
	restoreLock    sync.RWMutex
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

	conn, err := zzk.GetBasePathConnection(zzk.GeneratePoolPath(svc.PoolID))
	if err != nil {
		return err
	}

	_, err = zkdocker.SendAction(conn, &req)
	return err
}

func (this *ControlPlaneDao) RestartService(serviceID string, unused *int) error {
	return dao.ControlPlaneError{Msg: "unimplemented"}
}

// Create a elastic search control plane data access object
func NewControlPlaneDao(hostName string, port int, facade *facade.Facade) (*ControlPlaneDao, error) {
	glog.V(0).Infof("Opening ElasticSearch ControlPlane Dao: hostName=%s, port=%d", hostName, port)
	api.Domain = hostName
	api.Port = strconv.Itoa(port)

	dao := &ControlPlaneDao{
		hostName: hostName,
		port:     port,
	}
	if dfs, err := dfs.NewDistributedFileSystem(dao, facade); err != nil {
		return nil, err
	} else {
		dao.dfs = dfs
	}

	return dao, nil
}

func NewControlSvc(hostName string, port int, facade *facade.Facade, varpath, vfs string) (*ControlPlaneDao, error) {
	glog.V(2).Info("calling NewControlSvc()")
	defer glog.V(2).Info("leaving NewControlSvc()")

	s, err := NewControlPlaneDao(hostName, port, facade)
	if err != nil {
		return nil, err
	}

	//Used to bridge old to new
	s.facade = facade

	s.varpath = varpath
	s.vfs = vfs

	// create the account credentials
	if err = createSystemUser(s); err != nil {
		return nil, err
	}

	return s, nil
}
