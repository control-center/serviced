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

package api

import (
	"fmt"

	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/utils"
)

// AttachConfig is the deserialized object from the command-line
type AttachConfig struct {
	Running *dao.RunningService
	Command string
	Args    []string
}

func (a *api) GetRunningServices() ([]dao.RunningService, error) {
	client, err := a.connectDAO()
	if err != nil {
		return nil, err
	}

	var rss []dao.RunningService
	if err := client.GetRunningServices(&empty, &rss); err != nil {
		return nil, err
	}

	return rss, nil
}

// StopRunningService halts the specifed running service container
func (a *api) StopRunningService(hostID string, serviceStateID string) error {
	client, err := a.connectDAO()
	if err != nil {
		return err
	}

	var unused int
	if err := client.StopRunningInstance(dao.HostServiceRequest{HostID: hostID, ServiceStateID: serviceStateID}, &unused); err != nil {
		return err
	}

	return nil
}

// Attach runs an arbitrary shell command in a running service container
func (a *api) Attach(config AttachConfig) error {
	if hostID, err := utils.HostID(); err != nil {
		return err
	} else if hostID == config.Running.HostID {
		var command []string
		if config.Command != "" {
			command = append([]string{config.Command}, config.Args...)
		} else {
			command = append([]string{}, "/bin/bash")
		}

		return utils.AttachAndExec(config.Running.DockerID, command)
	}

	return fmt.Errorf("container does not reside locally on host")
}

// Action runs a predefined action in a running service container
func (a *api) Action(config AttachConfig) error {
	client, err := a.connectDAO()
	if err != nil {
		return err
	}

	req := dao.AttachRequest{
		Running: config.Running,
		Command: config.Command,
		Args:    config.Args,
	}

	return client.Action(req, &unusedInt)
}
