package api

import (
	"fmt"
	"os"
	"strings"

	"github.com/zenoss/serviced/shell"
)

// ShellConfig is the deserialized object from the command-line
type ShellConfig struct {
	ServiceID string
	Command   string
	Args      []string
	SaveAs    string
	IsTTY     bool
}

// StartShell runs a command for a given service
func (a *api) StartShell(config ShellConfig) error {
	command := []string{config.Command}
	command = append(command, config.Args...)

	cfg := shell.ProcessConfig{
		ServiceId: config.ServiceID,
		IsTTY:     config.IsTTY,
		SaveAs:    config.SaveAs,
		Command:   strings.Join(command, " "),
	}

	// TODO: change me to use sockets
	cmd, err := shell.StartDocker(&cfg, options.Port)
	if err != nil {
		return fmt.Errorf("failed to connect to service: %s", err)
	}

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()

	return nil
}

// RunShell runs a predefined service shell command via the service definition
func (a *api) RunShell(config ShellConfig) error {
	client, err := a.connectDAO()
	if err != nil {
		return err
	}

	service, err := a.GetService(config.ServiceID)
	if err != nil {
		return err
	}

	if err := service.EvaluateRunsTemplate(client); err != nil {
		fmt.Errorf("error evaluating service:%s Runs:%+v  error:%s", service.Id, service.Runs, err)
	}
	command, ok := service.Runs[config.Command]
	if !ok {
		return fmt.Errorf("command not found for service")
	}
	command = strings.Join(append([]string{command}, config.Args...), " ")

	cfg := shell.ProcessConfig{
		ServiceId: config.ServiceID,
		IsTTY:     config.IsTTY,
		SaveAs:    config.SaveAs,
		Command:   fmt.Sprintf("su - zenoss -c \"%s\"", command),
	}

	// TODO: change me to use sockets
	cmd, err := shell.StartDocker(&cfg, options.Port)
	if err != nil {
		return fmt.Errorf("failed to connect to service: %s", err)
	}

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()

	return nil
}
