package api

import (
	"github.com/zenoss/glog"
	docker "github.com/zenoss/go-dockerclient"

	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

// AttachConfig is the deserialized object from the command-line
type AttachConfig struct {
	DockerId string
	Command  []string
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

// attachContainerAndExec connects to a container and executes an arbitrary bash command
func attachContainerAndExec(containerId string, cmd []string) error {
	exeMap, err := exePaths([]string{"sudo", "nsinit"})
	if err != nil {
		return err
	}

	NSINIT_ROOT := "/var/lib/docker/execdriver/native" // has container.json

	attachCmd := fmt.Sprintf("cd %s/%s && %s exec %s", NSINIT_ROOT, containerId,
		exeMap["nsinit"], strings.Join(cmd, " "))
	fullCmd := []string{exeMap["sudo"], "--", "/bin/bash", "-c", attachCmd}
	glog.V(1).Infof("exec cmd: %v\n", fullCmd)
	return syscall.Exec(fullCmd[0], fullCmd[0:], os.Environ())
}

// getServiceFromContainerID inspects a docker container and retrieves the service
func (a *api) getServiceFromContainerID(containerID string) (*RunningService, error) {
	// retrieve docker container name from containerID
	const DOCKER_ENDPOINT string = "unix:///var/run/docker.sock"
	dockerClient, err := docker.NewClient(DOCKER_ENDPOINT)
	if err != nil {
		glog.Errorf("could not attach to docker client error:%v\n\n", err)
		return nil, err
	}
	container, err := dockerClient.InspectContainer(containerID)
	if err != nil {
		glog.Errorf("could not inspect container error:%v\n\n", err)
		return nil, err
	}

	// retrieve the service state
	serviceStateID := container.Name
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
	containerID := config.DockerId

	// validate that the given dockerID is a service
	if running, err := a.getServiceFromContainerID(containerID); err != nil {
		glog.Errorf("could not get serviceID from containerID:%s  error:%v\n", containerID, err)
		return err
	} else {
		glog.V(2).Infof("retrieved from containerID:%s  serviceID:%s  serviceName:%s s\n",
			containerID, running.Service.Id, running.Service.Name)
	}

	// attach to the container and run the command
	command := config.Command
	if err := attachContainerAndExec(containerID, command); err != nil {
		glog.Errorf("error running bash command:'%v'  error:%v\n", command, err)
		return err
	}

	return nil
}
