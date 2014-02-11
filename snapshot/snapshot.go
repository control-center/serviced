// snapshot package.
package snapshot

import (
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/dao"
	"github.com/zenoss/serviced/volume"

	"errors"
	"fmt"
	"os/exec"
	"time"
)

func callQuiescePause(cpDao dao.ControlPlane) error {
	// assuming lxc-attach is setuid for docker group
	//   sudo chgrp docker /usr/bin/lxc-attach
	//   sudo chmod u+s /usr/bin/lxc-attach

	var request dao.EntityRequest
	var servicesList []*dao.Service
	if err := cpDao.GetServices(request, &servicesList); err != nil {
		return err
	}
	for _, service := range servicesList {
		if service.Snapshot.Pause != "" && service.Snapshot.Resume != "" {
			glog.V(2).Infof("quiesce pause  service: %+v", service)
			cmd := exec.Command("echo", "TODO:", "lxc-attach", "-n", string(service.Id), "--", service.Snapshot.Pause)
			output, err := cmd.CombinedOutput()
			if err != nil {
				glog.Errorf("Unable to quiesce pause service %+v with cmd %+v because: %v", service, cmd, err)
				return err
			}
			glog.V(2).Infof("quiesce paused service - output:%s", string(output))
		}
	}

	// TODO: deficiency of this algorithm is that if one service fails to pause,
	//       all paused services will stay paused
	//       Perhaps one way to fix it is to call resume for all paused services
	//       if any of them fail to pause

	return nil
}

func callQuiesceResume(cpDao dao.ControlPlane) error {
	var request dao.EntityRequest
	var servicesList []*dao.Service
	if err := cpDao.GetServices(request, &servicesList); err != nil {
		return err
	}
	for _, service := range servicesList {
		if service.Snapshot.Pause != "" && service.Snapshot.Resume != "" {
			glog.V(2).Infof("quiesce resume service: %+v", service)
			cmd := exec.Command("echo", "TODO:", "lxc-attach", "-n", string(service.Id), "--", service.Snapshot.Resume)
			output, err := cmd.CombinedOutput()
			if err != nil {
				glog.Errorf("Unable to resume service %+v with cmd %+v because: %v", service, cmd, err)
				return err
			}
			glog.V(2).Infof("quiesce resume service - output:%+v", output)
		}
	}

	// TODO: deficiency of this algorithm is that if one service fails to resume,
	//       all remaining paused services will stay paused
	//       Perhaps one way to fix it is to call resume for all paused services
	//       if any of them fail to resume

	return nil
}

// ExecuteSnapshot is called by the Leader to perform the snapshot
func ExecuteSnapshot(cpDao dao.ControlPlane, serviceId string, label *string) error {
	glog.V(2).Infof("snapshot.ExecuteSnapshot service=%+v", serviceId)

	var tenantId string
	if err := cpDao.GetTenantId(serviceId, &tenantId); err != nil {
		glog.V(2).Infof("snapshot.ExecuteSnapshot cpDao.GetTenantId() service=%+v err=%s", serviceId, err)
		return err
	}
	var service dao.Service
	if err := cpDao.GetService(tenantId, &service); err != nil {
		glog.V(2).Infof("snapshot.ExecuteSnapshot cpDao.GetService() service=%+v err=%s", serviceId, err)
		return err
	}

	// simplest case - do everything here

	// call quiesce pause for services with 'Snapshot' definition
	if err := callQuiescePause(cpDao); err != nil {
		glog.V(2).Infof("snapshot.ExecuteSnapshot service=%+v err=%s", serviceId, err)
		return err
	}

	// create a snapshot
	var theVolume *volume.Volume
	if err := cpDao.GetVolume(tenantId, &theVolume); err != nil {
		glog.V(2).Infof("snapshot.ExecuteSnapshot cpDao.GetVolume() service=%+v err=%s", serviceId, err)
		return err
	} else if theVolume == nil {
		glog.V(2).Infof("snapshot.ExecuteSnapshot cpDao.GetVolume() volume is nil service=%+v", serviceId)
		return errors.New(fmt.Sprintf("GetVolume() is nil - tenantId:%s", tenantId))
	} else {
		glog.V(2).Infof("snapshot.ExecuteSnapshot service=%+v theVolume=%+v", service, theVolume)
		snapLabel := snapShotName(theVolume.Name())
		if err := theVolume.Snapshot(snapLabel); err != nil {
			return err
		} else {
			*label = snapLabel
		}
	}

	// call quiesce resume for services with 'Snapshot' definition
	if err := callQuiesceResume(cpDao); err != nil {
		glog.V(2).Infof("snapshot.ExecuteSnapshot service=%+v err=%s", serviceId, err)
		return err
	}

	return nil
}

func snapShotName(volumeName string) string {
	format := "20060102-150405"
	loc := time.Now()
	utc := loc.UTC()
	return volumeName + "_" + utc.Format(format)
}
