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
	Command     []string
}

// findServiceStates returns states in given serviceMap that match DockerId, ServiceName, or ServiceId
func (a *api) findServiceStates(serviceSpecifier string, serviceMap map[string]*dao.Service) ([]*dao.ServiceState, error) {
	var states []*dao.ServiceState
	for _, service := range serviceMap {
		glog.V(2).Infof("looking for specifier:%s in service:  ServiceId:%s  ServiceName:%s\n",
			serviceSpecifier, service.Id, service.Name)
		statesByServiceID, err := a.GetServiceStatesByServiceID(service.Id)
		if err != nil {
			return []*dao.ServiceState{}, err
		}

		for _, state := range statesByServiceID {
			glog.V(2).Infof("looking for specifier:%s in   state:  ServiceId:%s  ServiceName:%s  DockerId:%s\n",
				serviceSpecifier, state.ServiceId, service.Name, state.DockerId)
			if state.DockerId == "" {
				continue
			}
			if serviceSpecifier == state.ServiceId ||
				serviceSpecifier == service.Name ||
				strings.HasPrefix(state.DockerId, serviceSpecifier) {
				states = append(states, state)
			}
		}
	}

	return states, nil
}

// findContainerID finds the containerID from either DockerId, ServiceName, or ServiceId
func (a *api) findContainerID(serviceSpecifier string) (string, error) {
	// retrieve all services and populate serviceMap with ServiceId as key
	var serviceMap map[string]*dao.Service
	services, err := a.GetServices()
	if err != nil {
		return "", err
	}
	serviceMap = make(map[string]*dao.Service)
	for _, service := range services {
		serviceMap[service.Id] = service
	}

	// find running services that match specifier
	states, err := a.findServiceStates(serviceSpecifier, serviceMap)
	if err != nil {
		return "", err
	}

	// validate results
	if len(states) < 1 {
		return "", fmt.Errorf("did not find any running services matching specifier:'%s'", serviceSpecifier)
	}
	if len(states) > 1 {
		message := fmt.Sprintf("%d running services matched specifier:'%s'\n", len(states), serviceSpecifier)
		message += fmt.Sprintf("%-16s %-40s %s\n", "NAME", "SERVICEID", "DOCKERID")
		for _, state := range states {
			message += fmt.Sprintf("%-16s %-40s %s\n",
				serviceMap[state.ServiceId].Name, state.ServiceId, state.DockerId)
		}
		return "", fmt.Errorf("%s", message)
	}

	// return the docker container
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
	if config.ServiceSpec == "" {
		return fmt.Errorf("required ServiceAttachConfig.ServiceSpec is empty")
	}
	var command []string = []string{"bash"}
	if strings.TrimSpace(strings.Join(config.Command, "")) != "" {
		command = config.Command
	}

	containerID, err := a.findContainerID(config.ServiceSpec)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error looking for DOCKER_ID with specifier:'%v'  error:%v\n", config.ServiceSpec, err)
		return err
	}

	if err := attachContainerAndExec(containerID, command); err != nil {
		fmt.Fprintf(os.Stderr, "error running bash command:'%v'  error:%v\n", command, err)
		return err
	}

	return nil
}
