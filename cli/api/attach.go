package api

import (
	"github.com/zenoss/serviced/dao"
	zkdocker "github.com/zenoss/serviced/zzk/docker"
)

// AttachConfig is the deserialized object from the command-line
type AttachConfig struct {
	ServiceID      string
	ServiceStateID string
	Command        string
	Args           []string
}

func (a *api) GetRunningServices() ([]*dao.RunningService, error) {
	client, err := a.connectDAO()
	if err != nil {
		return nil, err
	}

	var rss []*dao.RunningService
	if err := client.GetRunningServices(nil, &rss); err != nil {
		return nil, err
	}

	return rss, nil
}

// Attach runs an arbitrary shell command in a running service container
func (a *api) Attach(config AttachConfig) error {
	client, err := a.connectDAO()
	if err != nil {
		return err
	}

	req := dao.AttachRequest{
		ServiceID:      config.ServiceID,
		ServiceStateID: config.ServiceStateID,
		Command:        config.Command,
		Args:           config.Args,
	}

	var res zkdocker.Attach
	if err := client.Attach(req, &res); err != nil {
		return err
	}
	return res.Error
}

// Action runs a predefined action in a running service container
func (a *api) Action(config AttachConfig) ([]byte, error) {
	client, err := a.connectDAO()
	if err != nil {
		return nil, err
	}

	req := dao.AttachRequest{
		ServiceID:      config.ServiceID,
		ServiceStateID: config.ServiceStateID,
		Command:        config.Command,
		Args:           config.Args,
	}

	var res zkdocker.Attach
	if err := client.Action(req, &res); err != nil {
		return nil, err
	}

	return res.Output, res.Error
}
