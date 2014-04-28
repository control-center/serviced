package api

import (
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/dao"

	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

// ServiceAttachConfig is the deserialized object from the command-line
type ServiceAttachConfig struct {
	ServiceSpec string
	Command     string
	Args        []string
}

// findServiceStates finds states that match DockerId, ServiceName, or ServiceId
func findServiceStates(a *api, serviceSpecifier string) ([]*dao.ServiceState, map[string]*dao.Service, error) {
	var serviceMap map[string]*dao.Service
	var states []*dao.ServiceState

	// make serviceMap
	services, err := a.GetServices()
	if err != nil {
		return states, serviceMap, err
	}

	serviceMap = make(map[string]*dao.Service)
	for _, service := range services {
		serviceMap[service.Id] = service

		statesByServiceID, err := a.GetServiceStatesByServiceID(service.Id)
		if err != nil {
			return []*dao.ServiceState{}, map[string]*dao.Service{}, err
		}

		for _, state := range statesByServiceID {
			if serviceSpecifier == state.ServiceId ||
				serviceSpecifier == serviceMap[state.ServiceId].Name ||
				strings.HasPrefix(state.DockerId, serviceSpecifier) {
				states = append(states, state)
			}
		}
	}

	return states, serviceMap, nil
}

// findContainerID finds the containerID from either DockerId, ServiceName, or ServiceId
func findContainerID(a *api, serviceSpecifier string) (string, error) {
	states, serviceMap, err := findServiceStates(a, serviceSpecifier)
	if err != nil {
		return "", err
	}
	if len(states) < 1 {
		return "", fmt.Errorf("did not find any service instances matching specifier:'%s'", serviceSpecifier)
	}
	if len(states) > 1 {
		message := fmt.Sprintf("%d service instances matched specifier:'%s'\n", len(states), serviceSpecifier)
		for _, state := range states {
			message += fmt.Sprintf("  DockerId:%s ServiceId:%s ServiceName:%s\n",
				state.DockerId, state.ServiceId, serviceMap[state.ServiceId].Name)
		}
		return "", fmt.Errorf("%s", message)
	}

	return states[0].DockerId, nil
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

// ServiceAttach runs an arbitrary shell command in a running service container
func (a *api) ServiceAttach(config ServiceAttachConfig) error {
	containerID, err := findContainerID(a, config.ServiceSpec)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error looking for DOCKER_ID with specifier:'%v'  error:%v\n", config.ServiceSpec, err)
		return err
	}

	var command []string
	if config.Command == "" {
		return fmt.Errorf("required config.Command is empty")
	}
	command = append(command, config.Command)
	command = append(command, config.Args...)

	if err := attachContainerAndExec(containerID, command); err != nil {
		fmt.Fprintf(os.Stderr, "error running bash command:'%v'  error:%v\n", command, err)
		return err
	}

	return nil
}
