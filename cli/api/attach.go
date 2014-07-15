// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package api

import (
	"fmt"

	"github.com/zenoss/serviced/dao"
	"github.com/zenoss/serviced/utils"
)

// AttachConfig is the deserialized object from the command-line
type AttachConfig struct {
	Running *dao.RunningService
	Command string
	Args    []string
}

func (a *api) GetRunningServices() ([]*dao.RunningService, error) {
	client, err := a.connectDAO()
	if err != nil {
		return nil, err
	}

	var rss []*dao.RunningService
	if err := client.GetRunningServices(&empty, &rss); err != nil {
		return nil, err
	}

	return rss, nil
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
