package api

import (
	"github.com/zenoss/serviced/dao"
	"github.com/zenoss/serviced/utils"
	zkdocker "github.com/zenoss/serviced/zzk/docker"
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

func isLocal(request dao.AttachRequest) (bool, error) {
	if hostID, err := utils.HostID(); err != nil {
		return false, err
	} else if hostID == request.Running.HostID {
		var command []string
		if request.Command != "" {
			command = append([]string{request.Command}, request.Args...)
		}

		cmd := zkdocker.Attach{
			HostID:   request.Running.HostID,
			DockerID: request.Running.DockerID,
			Command:  command,
		}
		return true, zkdocker.LocalAttach(&cmd)
	}
	return false, nil
}

// Attach runs an arbitrary shell command in a running service container
func (a *api) Attach(config AttachConfig) error {
	client, err := a.connectDAO()
	if err != nil {
		return err
	}

	req := dao.AttachRequest{
		Running: config.Running,
		Command: config.Command,
		Args:    config.Args,
	}

	// Try to attach locally
	if ok, err := isLocal(req); ok || err != nil {
		return err
	}

	// Try to attach remotely
	return client.Attach(req, &unusedInt)
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
