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
	"os"
	"strings"
	"time"

	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/node"
	"github.com/control-center/serviced/shell"
	"github.com/control-center/serviced/utils"
	"github.com/zenoss/glog"
	dockerclient "github.com/zenoss/go-dockerclient"
)

// ShellConfig is the deserialized object from the command-line
type ShellConfig struct {
	ServiceID        string
	Command          string
	Args             []string
	SaveAs           string
	IsTTY            bool
	Mounts           []string
	ServicedEndpoint string
	LogToStderr      bool
	LogStash         struct {
		Enable        bool
		SettleTime    string
		IdleFlushTime string
	}
}

// getServiceBindMounts retrieves a service's bindmounts
func getServiceBindMounts(lbClientPort string, serviceID string) (map[string]string, error) {
	client, err := node.NewLBClient(lbClientPort)
	if err != nil {
		glog.Errorf("Could not create a client to endpoint: %s, %s", lbClientPort, err)
		return nil, err
	}
	defer client.Close()

	var bindmounts map[string]string
	err = client.GetServiceBindMounts(serviceID, &bindmounts)
	if err != nil {
		glog.Errorf("Error getting service %s's bindmounts, error: %s", serviceID, err)
		return nil, err
	}

	glog.V(1).Infof("getServiceBindMounts: service id=%s: %s", serviceID, bindmounts)
	return bindmounts, nil
}

func buildMounts(lbClientPort string, serviceID string, defaultMounts []string) ([]string, error) {
	bindmounts, err := getServiceBindMounts(lbClientPort, serviceID)
	if err != nil {
		return nil, err
	}

	mounts := defaultMounts
	for hostPath, containerPath := range bindmounts {
		bind := hostPath + "," + containerPath
		mounts = append(mounts, bind)
	}

	return mounts, nil
}

// StartShell runs a command for a given service
func (a *api) StartShell(config ShellConfig) error {
	mounts, err := buildMounts(config.ServicedEndpoint, config.ServiceID, config.Mounts)
	if err != nil {
		return err
	}

	command := []string{config.Command}
	command = append(command, config.Args...)

	cfg := shell.ProcessConfig{
		ServiceID: config.ServiceID,
		IsTTY:     config.IsTTY,
		SaveAs:    config.SaveAs,
		Mount:     mounts,
		Command:   strings.Join(command, " "),
	}

	// TODO: change me to use sockets
	cmd, err := shell.StartDocker(&cfg, options.Endpoint)
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

	svc, err := a.GetService(config.ServiceID)
	if err != nil {
		return err
	}

	getSvc := func(svcID string) (service.Service, error) {
		s := service.Service{}
		err := client.GetService(svcID, &s)
		return s, err
	}

	findChild := func(svcID, childName string) (service.Service, error) {
		s := service.Service{}
		err := client.FindChildService(dao.FindChildRequest{svcID, childName}, &s)
		return s, err
	}

	if err := svc.EvaluateRunsTemplate(getSvc, findChild); err != nil {
		fmt.Errorf("error evaluating service:%s Runs:%+v  error:%s", svc.ID, svc.Runs, err)
	}
	command, ok := svc.Runs[config.Command]
	if !ok {
		return fmt.Errorf("command not found for service")
	}
	mounts, err := buildMounts(config.ServicedEndpoint, config.ServiceID, config.Mounts)
	if err != nil {
		return err
	}

	quotedArgs := []string{}
	for _, arg := range config.Args {
		quotedArgs = append(quotedArgs, fmt.Sprintf("\\\"%s\\\"", arg))
	}
	command = strings.Join(append([]string{command}, quotedArgs...), " ")

	cfg := shell.ProcessConfig{
		ServiceID:   config.ServiceID,
		IsTTY:       config.IsTTY,
		SaveAs:      config.SaveAs,
		Mount:       mounts,
		Command:     fmt.Sprintf("su - zenoss -c \"%s\"", command),
		LogToStderr: config.LogToStderr,
	}

	cfg.LogStash.Enable = config.LogStash.Enable
	cfg.LogStash.SettleTime, err = time.ParseDuration(config.LogStash.SettleTime)
	if err != nil {
		return err
	}

	cfg.LogStash.IdleFlushTime, err = time.ParseDuration(config.LogStash.IdleFlushTime)
	if err != nil {
		return err
	}

	// TODO: change me to use sockets
	cmd, err := shell.StartDocker(&cfg, options.Endpoint)
	if err != nil {
		return fmt.Errorf("failed to connect to service: %s", err)
	}

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if _, ok := utils.GetExitStatus(err); !ok {
		glog.Fatalf("abnormal termination from shell command: %s", err)
	}

	dockercli, err := a.connectDocker()
	if err != nil {
		glog.Fatalf("unable to connect to the docker service: %s", err)
	}
	exitcode, err := dockercli.WaitContainer(config.SaveAs)
	if err != nil {
		glog.Fatalf("failure waiting for container: %s", err)
	}
	container, err := dockercli.InspectContainer(config.SaveAs)
	if err != nil {
		glog.Fatalf("cannot acquire information about container: %s (%s)", config.SaveAs, err)
	}
	glog.V(2).Infof("Container ID: %s", container.ID)

	switch exitcode {
	case 0:
		// Commit the container
		label := ""
		glog.V(0).Infof("Committing container")
		if err := client.Commit(container.ID, &label); err != nil {
			glog.Fatalf("Error committing container: %s (%s)", container.ID, err)
		}
	default:
		// Delete the container
		glog.V(0).Infof("Command returned non-zero exit code %d.  Container not commited.", exitcode)
		if err := dockercli.StopContainer(container.ID, 10); err != nil {
			glog.Fatalf("failed to stop container: %s (%s)", container.ID, err)
		} else if err := dockercli.RemoveContainer(dockerclient.RemoveContainerOptions{ID: container.ID}); err != nil {
			glog.Fatalf("failed to remove container: %s (%s)", container.ID, err)
		}
	}

	return nil
}
