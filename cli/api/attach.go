package api

import (
	"github.com/zenoss/glog"

	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

// AttachConfig is the deserialized object from the command-line
type AttachConfig struct {
	ServiceStateID string
	Command        []string
}

// exePaths returns the full path to the given executables in a map
func exePaths(exes []string) (map[string]string, error) {
	exeMap := map[string]string{}

	for _, exe := range exes {
		path, err := exec.LookPath(exe)
		if err != nil {
			glog.Errorf("exe:'%v' not found error:%v\n", exe, err)
			return nil, err
		}

		exeMap[exe] = path
	}

	return exeMap, nil
}

// attachExecUsingContainerID connects to a container and executes an arbitrary bash command
func attachExecUsingContainerID(containerID string, cmd []string) error {
	if containerID == "" {
		return fmt.Errorf("will not attach to container with empty containerID")
	}

	exeMap, err := exePaths([]string{"sudo", "nsinit"})
	if err != nil {
		return err
	}

	NSINIT_ROOT := "/var/lib/docker/execdriver/native" // has container.json

	attachCmd := fmt.Sprintf("cd %s/%s && %s exec %s", NSINIT_ROOT, containerID,
		exeMap["nsinit"], strings.Join(cmd, " "))
	fullCmd := []string{exeMap["sudo"], "--", "/bin/bash", "-c", attachCmd}
	glog.V(1).Infof("exec cmd: %v\n", fullCmd)
	return syscall.Exec(fullCmd[0], fullCmd[0:], os.Environ())
}

// attachExecUsingServiceStateID connects to a container and executes an arbitrary bash command
func (a *api) attachExecUsingServiceStateID(serviceStateID string, cmd []string) error {
	// validate that the given dockerID is a service
	running, err := a.getRunningServiceFromServiceID(serviceStateID)
	if err != nil {
		glog.Errorf("could not get service from serviceStateID:%s  error:%v\n", serviceStateID, err)
		return err
	}

	glog.V(1).Infof("retrieved service/state using serviceStateID:%s ==> serviceID:%s  serviceName:%s  dockerId:%s\n",
		serviceStateID, running.Service.Id, running.Service.Name, running.State.DockerId)
	return attachExecUsingContainerID(running.State.DockerId, cmd)
}

// getRunningServiceFromServiceID retrieves the service and state from the DAO
func (a *api) getRunningServiceFromServiceID(serviceStateID string) (*RunningService, error) {
	// retrieve the service state
	state, err := a.GetServiceState(serviceStateID)
	if err != nil {
		return nil, err
	}

	// retrieve the service
	service, err := a.GetService(state.ServiceId)
	if err != nil {
		return nil, err
	}

	running := RunningService{
		Service: service,
		State:   state,
	}

	return &running, err
}

// Attach runs an arbitrary shell command in a running service container
func (a *api) Attach(config AttachConfig) error {
	serviceStateID := config.ServiceStateID
	command := config.Command
	if err := a.attachExecUsingServiceStateID(serviceStateID, command); err != nil {
		glog.Errorf("error running bash command:'%v'  error:%v\n", command, err)
		return err
	}

	return nil
}
