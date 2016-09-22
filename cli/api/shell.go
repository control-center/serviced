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

	"github.com/Sirupsen/logrus"
	ccconfig "github.com/control-center/serviced/config"
	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/node"
	"github.com/control-center/serviced/shell"
	"github.com/control-center/serviced/utils"
	dockerclient "github.com/fsouza/go-dockerclient"
)

// ShellConfig is the deserialized object from the command-line
type ShellConfig struct {
	ServiceID        string
	Command          string
	Args             []string
	Username         string
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
	log := log.WithFields(logrus.Fields{
		"address":   lbClientPort,
		"serviceid": serviceID,
	})
	client, err := node.NewLBClient(lbClientPort)
	if err != nil {
		log.WithError(err).Error("Unable to connect to RPC server")
		return nil, err
	}
	defer client.Close()

	var bindmounts map[string]string
	err = client.GetServiceBindMounts(serviceID, &bindmounts)
	if err != nil {
		if strings.HasPrefix(err.Error(), "rpc: can't find service") {
			log.Error("`serviced service shell` is only available when running serviced in delegate mode")
			return nil, err
		}
		log.Error("Unable to retrieve service bind mounts")
		return nil, err
	}
	log.WithFields(logrus.Fields{
		"mounts": bindmounts,
	}).Debug("Got service bind mounts")
	return bindmounts, nil
}

func buildMounts(lbClientPort string, serviceID string, defaultMounts []string) ([]string, error) {
	bindmounts, err := getServiceBindMounts(lbClientPort, serviceID)
	if err != nil {
		return nil, err
	}

	dsts := map[string]string{}
	for _, mnt := range defaultMounts {
		parts := strings.Split(mnt, ",")
		if len(parts) > 1 {
			dsts[parts[1]] = parts[0] // dsts[dst] = src
		}
	}

	mounts := defaultMounts
	for hostPath, containerPath := range bindmounts {
		if _, ok := dsts[containerPath]; !ok {
			bind := hostPath + "," + containerPath
			mounts = append(mounts, bind)
		}
	}

	// add the etc path
	mounts = append(mounts, fmt.Sprintf("%s,%s", options.EtcPath, "/etc/serviced"))

	return mounts, nil
}

// StartShell runs a command for a given service
func (a *api) StartShell(config ShellConfig) error {
	mounts, err := buildMounts(config.ServicedEndpoint, config.ServiceID, config.Mounts)
	if err != nil {
		return err
	}

	command := append([]string{config.Command}, config.Args...)

	cfg := shell.ProcessConfig{
		ServiceID: config.ServiceID,
		IsTTY:     config.IsTTY,
		SaveAs:    config.SaveAs,
		Mount:     mounts,
		Command:   utils.ShellQuoteArgs(command),
	}

	options := ccconfig.GetOptions()
	cmd, err := shell.StartDocker(&cfg, options.Endpoint, config.ServicedEndpoint, options.DockerRegistry, options.ControllerBinary)
	if err != nil {
		return fmt.Errorf("failed to connect to service: %s", err)
	}

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// RunShell runs a predefined service shell command via the service definition
func (a *api) RunShell(config ShellConfig, stopChan chan struct{}) (int, error) {
	client, err := a.connectDAO()
	if err != nil {
		return 1, err
	}

	svc, err := a.GetService(config.ServiceID)
	if err != nil {
		return 1, err
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
		return 1, fmt.Errorf("error evaluating service:%s Runs:%+v  error:%s", svc.ID, svc.Runs, err)
	}
	run, ok := svc.Commands[config.Command]
	if !ok {
		return 1, fmt.Errorf("command not found for service")
	}
	mounts, err := buildMounts(config.ServicedEndpoint, config.ServiceID, config.Mounts)
	if err != nil {
		return 1, err
	}

	quotedArgs := utils.ShellQuoteArgs(config.Args)
	command := strings.Join([]string{run.Command, quotedArgs}, " ")

	asUser := "su - root -c "
	if config.Username != "" && config.Username != "root" {
		asUser = fmt.Sprintf("su - %s -c ", config.Username)
	}

	cfg := shell.ProcessConfig{
		ServiceID:   config.ServiceID,
		IsTTY:       config.IsTTY,
		SaveAs:      config.SaveAs,
		Mount:       mounts,
		Command:     asUser + utils.ShellQuoteArg(command),
		LogToStderr: config.LogToStderr,
	}

	cfg.LogStash.Enable = config.LogStash.Enable
	cfg.LogStash.SettleTime, err = time.ParseDuration(config.LogStash.SettleTime)
	if err != nil {
		return 1, err
	}

	cfg.LogStash.IdleFlushTime, err = time.ParseDuration(config.LogStash.IdleFlushTime)
	if err != nil {
		return 1, err
	}

	options := ccconfig.GetOptions()
	cmd, err := shell.StartDocker(&cfg, options.Endpoint, config.ServicedEndpoint, options.DockerRegistry, options.ControllerBinary)
	if err != nil {
		return 1, fmt.Errorf("failed to connect to service: %s", err)
	}

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	dockercli, err := a.connectDocker()
	if err != nil {
		log.WithError(err).Fatal("Unable to connect to Docker")
	}

	log := log.WithFields(logrus.Fields{
		"containername": cfg.SaveAs,
	})

	err = cmd.Start()
	if err != nil {
		log.WithError(err).Fatal("Unable to start container")
	}
	cmdChan := make(chan error)
	go func() {
		cmdChan <- cmd.Wait()
	}()
	log.WithFields(logrus.Fields{
		"command": cmd,
	}).Debug("Started command in container")

	select {
	case <-stopChan:
		log.Debug("Received signal to stop. Stopping container")
		killContainerOpts := dockerclient.KillContainerOptions{ID: cfg.SaveAs}
		err = dockercli.KillContainer(killContainerOpts)
		if err != nil {
			log.WithError(err).Error("Unable to kill container")
		}
		log.Info("Killed container")
		return 1, err
	case err = <-cmdChan:
		if _, ok := utils.GetExitStatus(err); !ok {
			log.WithError(err).Fatal("Abnormal termination from shell command")
		}
	}

	if _, ok := utils.GetExitStatus(err); !ok {
		log.WithError(err).Fatal("Abnormal termination from shell command")
	}

	exitcode, err := dockercli.WaitContainer(config.SaveAs)
	if err != nil {
		log.WithError(err).Fatal("Failure waiting for container")
	}
	container, err := dockercli.InspectContainer(config.SaveAs)
	if err != nil {
		log.WithError(err).Fatal("Unable to acquire information about container")
	}
	log = log.WithFields(logrus.Fields{
		"containerid": container.ID,
	})
	log.Debug("Acquired container information")

	if exitcode == 0 {
		if run.CommitOnSuccess {
			// Commit the container
			label := ""
			log.Info("Committing container")
			if err := client.Snapshot(dao.SnapshotRequest{ContainerID: container.ID, SnapshotSpacePercent: options.SnapshotSpacePercent}, &label); err != nil {
				log.WithError(err).Fatal("Unable to commit container")
			}
		}
	} else {
		// Delete the container
		if err := dockercli.StopContainer(container.ID, 10); err != nil {
			if _, ok := err.(*dockerclient.ContainerNotRunning); !ok {
				log.WithError(err).Warn("Unable to stop container")
			}
		} else if err := dockercli.RemoveContainer(dockerclient.RemoveContainerOptions{ID: container.ID}); err != nil {
			log.WithError(err).Warn("Unable to remove container")
		}
		commitMsg := ""
		if run.CommitOnSuccess {
			commitMsg = " Container not committed."
		}
		return exitcode, fmt.Errorf("Command returned non-zero exit code %d.%s", exitcode, commitMsg)
	}

	return exitcode, nil
}
