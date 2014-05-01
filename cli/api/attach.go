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

// FindRunningServices returns running services that match DockerId, ServiceName, or ServiceId
func (a *api) FindRunningServices(keyword string) ([]*RunningService, error) {
	services, err := a.GetServices()
	if err != nil {
		return nil, err
	}

	var runningServices []*RunningService
	for serviceKey, service := range services {
		glog.V(2).Infof("looking for keyword:%s in service:  ServiceId:%s  ServiceName:%s\n",
			keyword, service.Id, service.Name)
		statesByServiceID, err := a.getServiceStatesByServiceID(service.Id)
		if err != nil {
			return []*RunningService{}, err
		}

		for stateKey, state := range statesByServiceID {
			glog.V(2).Infof("looking for keyword:%s in   state:  ServiceId:%s  ServiceName:%s  DockerId:%s\n",
				keyword, state.ServiceId, service.Name, state.DockerId)
			if state.DockerId == "" {
				continue
			}
			if keyword == state.ServiceId || keyword == service.Name || keyword == state.DockerId {

				// validate that the running service found is a running docker container
				serviceStateID, err := a.getServiceStateIDFromDocker(state.DockerId)
				if err != nil {
					continue
				}

				if serviceStateID != state.Id {
					glog.Warningf("docker.Name (serviceStateID:%s) does not match state.Id:%s",
						serviceStateID, state.Id)
					continue
				}

				running := RunningService{
					Service: services[serviceKey],
					State:   statesByServiceID[stateKey],
				}
				runningServices = append(runningServices, &running)
			}
		}
	}

	return runningServices, nil
}

// GetRunningService retrieves the service and state from the DAO
func (a *api) GetRunningService(serviceStateID string) (*RunningService, error) {
	// retrieve the service state
	state, err := a.GetServiceState(serviceStateID)
	if err != nil {
		glog.Errorf("could not get service state from serviceStateID:%s  error:%v\n", serviceStateID, err)
		return nil, err
	}

	// retrieve the service
	service, err := a.GetService(state.ServiceId)
	if err != nil {
		glog.Errorf("could not get service from state.ServiceID:%s  error:%v\n", state.ServiceId, err)
		return nil, err
	}

	running := RunningService{
		Service: service,
		State:   state,
	}

	return &running, err
}

// GetRunningServiceActionCommand retrieves the action command from the Service State
func (a *api) GetRunningServiceActionCommand(serviceStateID string, action string) (string, error) {
	running, err := a.GetRunningService(serviceStateID)
	if err != nil {
		return "", err
	} else if running == nil {
		return "", fmt.Errorf("no running service found for serviceStateID: %s", serviceStateID)
	}

	client, err := a.connectDAO()
	if err != nil {
		return "", err
	}

	// Evaluate service Actions for templates
	service := running.Service
	if err := running.Service.EvaluateActionsTemplate(client); err != nil {
		return "", fmt.Errorf("could not evaluate service:%s  Actions:%+v  error:%s", service.Id, service.Actions, err)
	}

	// Parse the command
	command, ok := service.Actions[action]
	if !ok {
		glog.Infof("service: %+v", service)
		glog.Fatalf("cannot access action: %s", action)
	}

	return command, nil
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

// generateAttachCommand returns a slice containing nsinit command to exec
func generateAttachCommand(containerID string, bashcmd []string) ([]string, error) {
	if containerID == "" {
		return []string{}, fmt.Errorf("will not attach to container with empty containerID")
	}

	exeMap, err := exePaths([]string{"sudo", "nsinit"})
	if err != nil {
		return []string{}, err
	}

	nsInitRoot := "/var/lib/docker/execdriver/native" // has container.json

	attachCmd := fmt.Sprintf("cd %s/%s && %s exec %s", nsInitRoot, containerID,
		exeMap["nsinit"], strings.Join(bashcmd, " "))
	fullCmd := []string{exeMap["sudo"], "--", "/bin/bash", "-c", attachCmd}
	glog.V(1).Infof("attach command for container:%v command: %v\n", containerID, fullCmd)
	return fullCmd, nil
}

// attachExecUsingContainerID connects to a container and executes an arbitrary bash command
func attachExecUsingContainerID(containerID string, cmd []string) error {
	fullCmd, err := generateAttachCommand(containerID, cmd)
	if err != nil {
		return err
	}
	return syscall.Exec(fullCmd[0], fullCmd[0:], os.Environ())
}

// attachExecUsingServiceStateID connects to a container and executes an arbitrary bash command
func (a *api) attachExecUsingServiceStateID(serviceStateID string, cmd []string) error {
	// validate that the given dockerID is a service
	running, err := a.GetRunningService(serviceStateID)
	if err != nil {
		return err
	}

	glog.V(1).Infof("retrieved service/state using serviceStateID:%s ==> serviceID:%s  serviceName:%s  dockerId:%s\n",
		serviceStateID, running.Service.Id, running.Service.Name, running.State.DockerId)
	return attachExecUsingContainerID(running.State.DockerId, cmd)
}

// Attach runs an arbitrary shell command in a running service container
func (a *api) Attach(config AttachConfig) error {
	glog.V(1).Infof("Attach(%+v)\n", config)

	serviceStateID := config.ServiceStateID
	command := config.Command
	if err := a.attachExecUsingServiceStateID(serviceStateID, command); err != nil {
		glog.Errorf("error running bash command:'%v'  error:%v\n", command, err)
		return err
	}

	return nil
}

// attachRunUsingContainerID attaches to a service state container and runs an arbitrary bash command
func attachRunUsingContainerID(containerID string, cmd []string) ([]byte, error) {
	fullCmd, err := generateAttachCommand(containerID, cmd)
	if err != nil {
		return nil, err
	}
	command := exec.Command(fullCmd[0], fullCmd[1:]...)

	output, err := command.CombinedOutput()
	if err != nil {
		glog.Errorf("Error running command:'%s' output: %s  error: %s\n", cmd, output, err)
		return output, err
	}
	glog.V(1).Infof("Successfully ran command:'%s' output: %s\n", cmd, output)
	return output, nil
}

// attachRunUsingServiceStateID attaches to a service state container and runs an arbitrary bash command
func (a *api) attachRunUsingServiceStateID(serviceStateID string, cmd []string) ([]byte, error) {
	// validate that the given dockerID is a service
	running, err := a.GetRunningService(serviceStateID)
	if err != nil {
		return nil, err
	}

	glog.V(1).Infof("retrieved service/state using serviceStateID:%s ==> serviceID:%s  serviceName:%s  dockerId:%s\n",
		serviceStateID, running.Service.Id, running.Service.Name, running.State.DockerId)
	return attachRunUsingContainerID(running.State.DockerId, cmd)
}

// Action runs a predefined action in a running service container
func (a *api) Action(config AttachConfig) ([]byte, error) {
	glog.V(1).Infof("Action(%+v)\n", config)

	serviceStateID := config.ServiceStateID
	command := config.Command
	output, err := a.attachRunUsingServiceStateID(serviceStateID, command)
	if err != nil {
		glog.Errorf("error running bash command:'%v'  error:%v\n", command, err)
		return nil, err
	}

	return output, nil
}
