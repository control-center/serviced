// Copyright 2016 The Serviced Authors.
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

package api

import (
	"fmt"
	"os"
	"syscall"

	dockerclient "github.com/control-center/serviced/commons/docker"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/utils"
)

// TODO: what to do about logging?

// GetServiceInstances returns all instances running on a service
func (a *api) GetServiceInstances(serviceID string) ([]service.Instance, error) {
	client, err := a.connectMaster()
	if err != nil {
		return nil, err
	}
	return client.GetServiceInstances(serviceID)
}

// StopServiceInstance stops a running instance of a service.
func (a *api) StopServiceInstance(serviceID string, instanceID int) error {
	client, err := a.connectMaster()
	if err != nil {
		return err
	}
	return client.StopServiceInstance(serviceID, instanceID)
}

// AttachServiceInstance locates and attaches to a running instance of a service
func (a *api) AttachServiceInstance(serviceID string, instanceID int, command string, args []string) error {
	client, err := a.connectMaster()
	if err != nil {
		return err
	}

	// get the location of the running instance
	location, err := client.LocateServiceInstance(serviceID, instanceID)
	if err != nil {
		return err
	}

	// check to see if it is running on this host
	hostID, err := utils.HostID()
	if err != nil {
		return err
	}

	// attach to the container
	cmd := []string{}
	if location.HostID != hostID {
		cmd := []string{
			"/usr/bin/ssh",
			"-t", location.HostIP, "--",
			"serviced", "--endpoint", GetOptionsRPCEndpoint(),
			"service", "attach", fmt.Sprintf("%s/%d", serviceID, instanceID),
		}
		cmd = append(cmd, command)
		cmd = append(cmd, args...)
		return syscall.Exec(cmd[0], cmd[0:], os.Environ())
	} else {
		if command == "" {
			cmd = append(cmd, "/bin/bash")
		} else {
			cmd = append(cmd, command)
			cmd = append(cmd, args...)
		}
		return utils.AttachAndExec(location.ContainerID, cmd)
	}
}

// LogsForServiceInstance returns the logs for the service instance
func (a *api) LogsForServiceInstance(serviceID string, instanceID int, command string, args []string) error {
	client, err := a.connectMaster()
	if err != nil {
		return err
	}

	// get the location of the running instance
	location, err := client.LocateServiceInstance(serviceID, instanceID)
	if err != nil {
		return err
	}

	// check to see if it is running on this host
	hostID, err := utils.HostID()
	if err != nil {
		return err
	}

	// report container logs
	cmd := []string{}
	if location.HostID != hostID {
		cmd := []string{
			"/usr/bin/ssh",
			"-t", location.HostIP, "--",
			"serviced", "--endpoint", GetOptionsRPCEndpoint(),
			"service", "logs", fmt.Sprintf("%s/%d", serviceID, instanceID),
		}
		cmd = append(cmd, command)
		cmd = append(cmd, args...)
		return syscall.Exec(cmd[0], cmd[0:], os.Environ())
	} else {
		cmd = append(cmd, command)
		cmd = append(cmd, args...)
		return dockerclient.Logs(location.ContainerID, cmd)
	}
}

// SendDockerAction submits an action to a running service instance
func (a *api) SendDockerAction(serviceID string, instanceID int, action string, args []string) error {
	client, err := a.connectMaster()
	if err != nil {
		return err
	}

	return client.SendDockerAction(serviceID, instanceID, action, args)
}
