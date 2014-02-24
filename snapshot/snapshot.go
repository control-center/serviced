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
	"strings"
	"time"
)

// getServiceDockerId returns the DockerId for the running container tied to the service
// assumption: Servicestate.DockerId is a one to one relationship to ServiceId
func getServiceDockerId(cpDao dao.ControlPlane, serviceId string) (string, error) {
	var states []*dao.ServiceState
	if err := cpDao.GetServiceStates(serviceId, &states); err != nil {
		return "", err
	}

	if len(states) > 1 {
		glog.Warningf("more than one ServiceState found for serviceId:%s numServiceStates:%d", serviceId, len(states))
	}

	// return the DockerId of the first ServiceState that matches serviceId
	for i, state := range states {
		glog.V(3).Infof("DEBUG states[%d]: serviceId:%s state:%+v", i, serviceId, state)
		if state.DockerId != "" && state.ServiceId == serviceId {
			return state.DockerId, nil
		}
	}

	return "", errors.New(fmt.Sprintf("unable to find DockerId for serviceId:%s", serviceId))
}

// runCommandInServiceContainer runs a command in a running container
func runCommandInServiceContainer(serviceId string, dockerId string, command string) (string, error) {
	dockerCommand := []string{"lxc-attach", "-n", dockerId, "-e", "--", "/bin/bash", "-c", command}
	cmd := exec.Command(dockerCommand[0], dockerCommand[1:len(dockerCommand)]...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		glog.Errorf("Error running cmd:'%s' for serviceId:%s - ERROR:%s  OUTPUT:%s", strings.Join(dockerCommand, " "), serviceId, err, output)
		return string(output), err
	}
	glog.V(0).Infof("Successfully ran cmd:'%s' for serviceId:%s - OUTPUT:%s", strings.Join(dockerCommand, " "), serviceId, string(output))
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

	// call quiesce for services with 'Snapshot.Pause' and 'Snapshot.Resume' definition
	// only root can run lxc-attach
	if whoami, err := user.Current(); err != nil {
		glog.Errorf("Unable to snapshot service - not able to retrieve user info error: %v", err)
		return err
	} else if "root" != whoami.Username {
		glog.Warningf("Unable to pause/resume service - Username is not root - whoami:%+v", whoami)
	} else {
		var request dao.EntityRequest
		var servicesList []*dao.Service
		if err := cpDao.GetServices(request, &servicesList); err != nil {
			return err
		}
		for _, service := range servicesList {
			if service.Snapshot.Pause == "" || service.Snapshot.Resume == "" {
				continue
			}

			dockerId, err := getServiceDockerId(cpDao, service.Id)
			if err != nil {
				glog.Warningf("Unable to pause service - not able to get DockerId for service.Id:%s service.Name:%s error:%s", service.Id, service.Name, err)
				continue
			}

			_, err = runCommandInServiceContainer(service.Id, dockerId, service.Snapshot.Pause)
			defer runCommandInServiceContainer(service.Id, dockerId, service.Snapshot.Resume)
			if err != nil {
				return err
			}
		}
	}

	// create a snapshot
	var theVolume volume.Volume
	if err := cpDao.GetVolume(tenantId, &theVolume); err != nil {
		glog.V(2).Infof("snapshot.ExecuteSnapshot cpDao.GetVolume() service=%+v err=%s", service, err)
		return err
	} else {
		glog.V(2).Infof("snapshot.ExecuteSnapshot service=%+v theVolume=%+v", service, theVolume)
		snapLabel := snapShotName(theVolume.Name())
		if err := theVolume.Snapshot(snapLabel); err != nil {
			return err
		} else {
			*label = snapLabel
		}
	}

	glog.V(2).Infof("Successfully created snapshot for service Id:%s Name:%s Label:%s", service.Id, service.Name, label)
	return nil
}

func snapShotName(volumeName string) string {
	format := "20060102-150405"
	loc := time.Now()
	utc := loc.UTC()
	return volumeName + "_" + utc.Format(format)
}
