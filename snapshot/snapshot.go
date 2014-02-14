// snapshot package.
package snapshot

import (
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/dao"
	"github.com/zenoss/serviced/volume"

	"errors"
	"fmt"
	"os/exec"
	"os/user"
	"time"
)

// getServiceDockerId returns the DockerId for the running container tied to the service
// Servicestate.DockerId is a one to one relationship to Service.Id
func getServiceDockerId(cpDao dao.ControlPlane, service *dao.Service) (string, error) {
	var states []*dao.ServiceState
	if err := cpDao.GetServiceStates(service.Id, &states); err != nil {
		return "", err
	}

	if len(states) > 1 {
		glog.Warningf("more than one ServiceState found for serviceId:%s ===> states:%+v", service.Id, states)
	}

	for _, state := range states {
		// return the DockerId of the first ServiceState
		if state.DockerId == "" {
			return "", errors.New(fmt.Sprintf("unable to find DockerId for service:%+v", service))
		}
		return state.DockerId, nil
	}

	return "", errors.New(fmt.Sprintf("unable to find DockerId for service:%+v", service))
}

// runCommandInServiceContainer runs a command in a running container
func runCommandInServiceContainer(serviceId string, dockerId string, command string) (string, error) {
	cmd := exec.Command("lxc-attach", "-n", dockerId, "--", command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		glog.Errorf("Error running cmd:'%s' for serviceId:%s - error:%s", command, serviceId, err)
		return string(output), err
	}
	glog.V(0).Infof("Successfully ran cmd:'%s' for serviceId:%s - output: %s", command, serviceId, string(output))
	return string(output), nil
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

	// call quiesce pause/resume for services with 'Snapshot' definition
	// only root can run lxc-attach
	if whoami, err := user.Current(); err != nil {
		glog.Errorf("Unable to pause service - not able to retrieve user info error: %v", err)
		return err
	} else if "root" != whoami.Username {
		glog.Warningf("Unable to pause service - Username is not root - whoami:%+v", whoami)
	} else {
		var request dao.EntityRequest
		var servicesList []*dao.Service
		if err := cpDao.GetServices(request, &servicesList); err != nil {
			return err
		}
		for _, service := range servicesList {
			dockerId, err := getServiceDockerId(cpDao, service)
			if err != nil {
				glog.Warningf("Unable to pause service - not able to get DockerId for service:%+v", service)
				continue
			}

			if service.Snapshot.Pause != "" && service.Snapshot.Resume != "" {
				_, err := runCommandInServiceContainer(service.Id, dockerId, service.Snapshot.Pause)
				defer runCommandInServiceContainer(service.Id, dockerId, service.Snapshot.Resume)
				if err != nil {
					return err
				}
			}
		}
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

	glog.V(2).Infof("Successfully created snapshot for service:%s - label:%s", serviceId, label)
	return nil
}

func snapShotName(volumeName string) string {
	format := "20060102-150405"
	loc := time.Now()
	utc := loc.UTC()
	return volumeName + "_" + utc.Format(format)
}
