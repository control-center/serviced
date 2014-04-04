package api

import ()

const ()

var ()

// ShellConfig is the deserialized object from the command-line
type ShellConfig struct {
}

// ListCommands lists all of the commands for a given service
func (a *api) ListCommands(id string) ([]string, error) {
	return nil, nil
}

// StartShell runs a command for a given service
func (a *api) StartShell(config ShellConfig) error {
	return nil
}
