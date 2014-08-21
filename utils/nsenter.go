// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package utils

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/zenoss/glog"
	docker "github.com/zenoss/go-dockerclient"
)

// connection to the docker client
var dockerClient *docker.Client

// Opens a connection to docker if not already connected
func connectDocker() (*docker.Client, error) {
	if dockerClient == nil {
		const DockerEndpoint string = "unix:///var/run/docker.sock"
		var err error
		if dockerClient, err = docker.NewClient(DockerEndpoint); err != nil {
			return nil, fmt.Errorf("could not create a client to docker: %s", err)
		}
	}
	return dockerClient, nil
}

// getPIDFromDockerID returns the pid of a docker container
func getPIDFromDockerID(containerID string) (string, error) {
	// retrieve host PID from containerID
	dockerClient, err := connectDocker()
	if err != nil {
		glog.Errorf("could not attach to docker client error:%v\n\n", err)
		return "", err
	}
	container, err := dockerClient.InspectContainer(containerID)
	if err != nil {
		glog.Errorf("could not inspect container error:%v\n\n", err)
		return "", err
	}

	pid := fmt.Sprintf("%d", container.State.Pid)
	return pid, nil
}

// ExecNSEnter execs the command using nsenter
func ExecNSEnter(containerID string, bashcmd []string) error {
	command, err := generateNSEnterCommand(containerID, bashcmd, false)
	if err != nil {
		return err
	}
	glog.V(1).Infof("exec command for container:%v command: %v\n", containerID, command)
	return syscall.Exec(command[0], command[0:], os.Environ())
}

// RunNSEnter runs the command using nsenter
func RunNSEnter(containerID string, bashcmd []string) ([]byte, error) {
	command, err := generateNSEnterCommand(containerID, bashcmd, true)
	if err != nil {
		return nil, err
	}
	thecmd := exec.Command(command[0], command[1:]...)
	output, err := thecmd.CombinedOutput()
	if err != nil {
		glog.Errorf("Error running command:'%s' output: %s  error: %s\n", command, output, err)
		return output, err
	}
	glog.V(1).Infof("Successfully ran command:'%s' output: %s\n", command, output)
	return output, nil
}

// generateNSEnterCommand returns a slice containing nsenter command to exec
func generateNSEnterCommand(containerID string, bashcmd []string, prependBash bool) ([]string, error) {
	if containerID == "" {
		return []string{}, fmt.Errorf("will not attach to container with empty containerID")
	}

	exeMap, err := exePaths([]string{"sudo", "nsenter"})
	if err != nil {
		return []string{}, err
	}

	pid, err := getPIDFromDockerID(containerID)
	if err != nil {
		return []string{}, err
	}

	attachCmd := []string{exeMap["sudo"], exeMap["nsenter"], "-m", "-u", "-i", "-n", "-p", "-t", pid, "--"}
	if prependBash {
		attachCmd = append(attachCmd, "/bin/bash", "-c", fmt.Sprintf("%s", strings.Join(bashcmd, " ")))
	} else {
		attachCmd = append(attachCmd, bashcmd...)
	}
	glog.V(1).Infof("attach command for container:%v command: %v\n", containerID, attachCmd)
	return attachCmd, nil
}

// AttachAndRun attaches to a container and runs the command
func AttachAndRun(containerID string, bashcmd []string) ([]byte, error) {
	_, err := exec.LookPath("nsenter")
	if err != nil {
		return nil, fmt.Errorf("unable to find nsenter exe:%v", err)
	}

	return RunNSEnter(containerID, bashcmd)
}

// AttachAndExec attaches to a container and execs the command
func AttachAndExec(containerID string, bashcmd []string) error {
	_, err := exec.LookPath("nsenter")
	if err != nil {
		return fmt.Errorf("unable to find nsenter exe:%v", err)
	}

	return ExecNSEnter(containerID, bashcmd)
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