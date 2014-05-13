// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package elasticsearch

import (
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/dao"
	"github.com/zenoss/serviced/domain/service"
	"github.com/zenoss/serviced/volume"

	"errors"
	"fmt"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"time"
)

func (this *ControlPlaneDao) DeleteSnapshot(snapshotId string, unused *int) error {
	return this.dfs.DeleteSnapshot(snapshotId)
}

func (this *ControlPlaneDao) DeleteSnapshots(serviceId string, unused *int) error {
	var tenantId string
	if err := this.GetTenantId(serviceId, &tenantId); err != nil {
		glog.V(2).Infof("ControlPlaneDao.DeleteSnapshots err=%s", err)
		return err
	}

	if serviceId != tenantId {
		glog.Infof("ControlPlaneDao.DeleteSnapshots service is not the parent, service=%s, tenant=%s", serviceId, tenantId)
		return nil
	}

	return this.dfs.DeleteSnapshots(tenantId)
}

func (this *ControlPlaneDao) Rollback(snapshotId string, unused *int) error {
	return this.dfs.Rollback(snapshotId)
}

// Takes a snapshot of the DFS via the host
func (this *ControlPlaneDao) LocalSnapshot(serviceId string, label *string) error {
	var tenantId string
	if err := this.GetTenantId(serviceId, &tenantId); err != nil {
		glog.Errorf("ControlPlaneDao.LocalSnapshot err=%s", err)
		return err
	}

	if id, err := this.dfs.Snapshot(tenantId); err != nil {
		glog.Errorf("ControlPlaneDao.LocalSnapshot err=%s", err)
		return err
	} else {
		*label = id
	}

	return nil
}

// Snapshot is called via RPC by the CLI to take a snapshot for a serviceId
func (this *ControlPlaneDao) Snapshot(serviceId string, label *string) error {
	glog.V(3).Infof("ControlPlaneDao.Snapshot entering snapshot with service=%s", serviceId)
	defer glog.V(3).Infof("ControlPlaneDao.Snapshot finished snapshot for service=%s", serviceId)

	var tenantId string
	if err := this.GetTenantId(serviceId, &tenantId); err != nil {
		glog.V(2).Infof("ControlPlaneDao: dao.LocalSnapshot err=%s", err)
		return err
	}

	// request a snapshot by placing request znode in zookeeper - leader will notice
	snapshotRequest, err := dao.NewSnapshotRequest(serviceId, "")
	if err != nil {
		glog.V(2).Infof("ControlPlaneDao: dao.NewSnapshotRequest err=%s", err)
		return err
	}
	if err := this.zkDao.AddSnapshotRequest(snapshotRequest); err != nil {
		glog.V(2).Infof("ControlPlaneDao.zkDao.AddSnapshotRequest err=%s", err)
		return err
	}
	// TODO:
	//	requestId := snapshotRequest.Id
	//	defer this.zkDao.RemoveSnapshotRequest(requestId)

	glog.Infof("added snapshot request: %+v", snapshotRequest)

	// wait for completion of snapshot request - check only once a second
	// BEWARE: this.zkDao.LoadSnapshotRequestW does not block like it should
	//         thus cannot use idiomatic select on eventChan and time.After() channels
	timeOutValue := time.Second * 60
	endTime := time.Now().Add(timeOutValue)
	for time.Now().Before(endTime) {
		glog.V(2).Infof("watching for snapshot completion for request: %+v", snapshotRequest)
		_, err := this.zkDao.LoadSnapshotRequestW(snapshotRequest.Id, snapshotRequest)
		switch {
		case err != nil:
			glog.Infof("failed snapshot request: %+v  error: %s", snapshotRequest, err)
			return err
		case snapshotRequest.SnapshotError != "":
			glog.Infof("failed snapshot request: %+v  error: %s", snapshotRequest, snapshotRequest.SnapshotError)
			return errors.New(snapshotRequest.SnapshotError)
		case snapshotRequest.SnapshotLabel != "":
			*label = snapshotRequest.SnapshotLabel
			glog.Infof("completed snapshot request: %+v  label: %s", snapshotRequest, *label)
			return nil
		}

		time.Sleep(1 * time.Second)
	}

	err = fmt.Errorf("timed out waiting %v for snapshot: %+v", timeOutValue, snapshotRequest)
	glog.Error(err)
	return err
}

func (this *ControlPlaneDao) Snapshots(serviceId string, labels *[]string) error {

	var tenantId string
	if err := this.GetTenantId(serviceId, &tenantId); err != nil {
		glog.V(2).Infof("ControlPlaneDao.Snapshots service=%+v err=%s", serviceId, err)
		return err
	}
	var service service.Service
	err := this.GetService(tenantId, &service)
	if err != nil {
		glog.V(2).Infof("ControlPlaneDao.Snapshots service=%+v err=%s", serviceId, err)
		return err
	}

	if volume, err := getSubvolume(this.vfs, service.PoolId, tenantId); err != nil {
		glog.V(2).Infof("ControlPlaneDao.Snapshots service=%+v err=%s", serviceId, err)
		return err
	} else {
		if snaplabels, err := volume.Snapshots(); err != nil {
			return err
		} else {
			glog.Infof("Got snap labels %v", snaplabels)
			*labels = snaplabels
		}
	}
	return nil
}
func (this *ControlPlaneDao) GetVolume(serviceId string, theVolume *volume.Volume) error {
	var tenantId string
	if err := this.GetTenantId(serviceId, &tenantId); err != nil {
		glog.V(2).Infof("ControlPlaneDao.GetVolume service=%+v err=%s", serviceId, err)
		return err
	}
	glog.V(3).Infof("ControlPlaneDao.GetVolume service=%+v tenantId=%s", serviceId, tenantId)
	var service service.Service
	if err := this.GetService(tenantId, &service); err != nil {
		glog.V(2).Infof("ControlPlaneDao.GetVolume service=%+v err=%s", serviceId, err)
		return err
	}
	glog.V(3).Infof("ControlPlaneDao.GetVolume service=%+v poolId=%s", service, service.PoolId)

	aVolume, err := getSubvolume(this.vfs, service.PoolId, tenantId)
	if err != nil {
		glog.V(2).Infof("ControlPlaneDao.GetVolume service=%+v err=%s", serviceId, err)
		return err
	}
	if aVolume == nil {
		glog.V(2).Infof("ControlPlaneDao.GetVolume service=%+v volume=nil", serviceId)
		return errors.New("volume is nil")
	}

	glog.V(2).Infof("ControlPlaneDao.GetVolume service=%+v volume2=%+v %v", serviceId, aVolume, aVolume)
	*theVolume = *aVolume
	return nil
}

// Commits a container to an image and saves it on the DFS
func (this *ControlPlaneDao) Commit(containerId string, label *string) error {
	if id, err := this.dfs.Commit(containerId); err != nil {
		glog.V(2).Infof("ControlPlaneDao.GetVolume containerId=%s err=%s", containerId, err)
		return err
	} else {
		*label = id
	}

	return nil
}

func getSubvolume(vfs, poolId, tenantId string) (*volume.Volume, error) {
	baseDir, err := filepath.Abs(path.Join(varPath(), "volumes", poolId))
	if err != nil {
		return nil, err
	}
	glog.Infof("Mounting vfs:%v tenantId:%v baseDir:%v\n", vfs, tenantId, baseDir)
	return volume.Mount(vfs, tenantId, baseDir)
}

func varPath() string {
	if len(os.Getenv("SERVICED_HOME")) > 0 {
		return path.Join(os.Getenv("SERVICED_HOME"), "var")
	} else if user, err := user.Current(); err == nil {
		return path.Join(os.TempDir(), "serviced-"+user.Username, "var")
	} else {
		defaultPath := "/tmp/serviced/var"
		glog.Warningf("Defaulting varPath to:%v\n", defaultPath)
		return defaultPath
	}
}

func (s *ControlPlaneDao) ReadyDFS(unused bool, unusedint *int) (err error) {
	s.dfs.Lock()
	s.dfs.Unlock()
	return
}
