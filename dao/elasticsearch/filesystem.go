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

package elasticsearch

import (
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/volume"
	"github.com/control-center/serviced/zzk"
	zkSnapshot "github.com/control-center/serviced/zzk/snapshot"
	"github.com/zenoss/glog"

	"errors"
	"os"
	"os/user"
	"path"
	"path/filepath"
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
func (this *ControlPlaneDao) TakeSnapshot(serviceID string, label *string) error {
	var tenantID string
	var err error
	if err = this.GetTenantId(serviceID, &tenantID); err != nil {
		glog.V(2).Infof("ControlPlaneDao.DeleteSnapshots err=%s", err)
		return err
	}
	*label, err = this.dfs.Snapshot(tenantID)
	return err
}

// Snapshot is called via RPC by the CLI to take a snapshot for a serviceId
func (this *ControlPlaneDao) Snapshot(serviceID string, label *string) error {
	myService, err := this.facade.GetService(datastore.Get(), serviceID)
	if err != nil {
		glog.Errorf("Unable to get service %v: %v", serviceID, err)
		return err
	}

	conn, err := zzk.GetLocalConnection(zzk.GeneratePoolPath(myService.PoolID))
	if err != nil {
		glog.Errorf("ControlPlaneDao.Snapshot err=%s", err)
		return err
	}

	if err := zkSnapshot.Send(conn, serviceID); err != nil {
		glog.Errorf("ControlPlaneDao.Snapshot err=%s", err)
		return err
	}

	var snapshot zkSnapshot.Snapshot
	if err := zkSnapshot.Recv(conn, serviceID, &snapshot); err != nil {
		glog.Errorf("ControlPlaneDao.Snapshot err=%s", err)
		return err
	}

	*label = snapshot.Label
	if snapshot.Err != "" {
		return errors.New(snapshot.Err)
	}
	return nil
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

	if volume, err := getSubvolume(this.vfs, service.PoolID, tenantId); err != nil {
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
	*theVolume = volume.Volume{}

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
	glog.V(3).Infof("ControlPlaneDao.GetVolume service=%+v poolId=%s", service, service.PoolID)

	aVolume, err := getSubvolume(this.vfs, service.PoolID, tenantId)
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
