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

package utils

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/zenoss/glog"
)

// An error type for failed docker exec attempts.
type DockerExecError struct {
	Command string
	ExecErr error
}

func (err DockerExecError) Error() string {
	return fmt.Sprintf("Error running command: %s: %s", err.Command, err.ExecErr)
}

// ExecDockerExec execs the command using docker exec
func ExecDockerExec(containerID string, bashcmd []string) error {
	command, err := generateDockerExecCommand(containerID, bashcmd, false)
	if err != nil {
		return err
	}
	glog.V(1).Infof("exec command for container:%v command: %v\n", containerID, command)
	return syscall.Exec(command[0], command[0:], os.Environ())
}

// RunDockerExec runs the command using docker exec
func RunDockerExec(containerID string, bashcmd []string) ([]byte, error) {
	oldStdin := os.Stdin
	os.Stdin = nil // temporary stdin=nil https://github.com/docker/docker/pull/9537
	command, err := generateDockerExecCommand(containerID, bashcmd, true)
	os.Stdin = oldStdin
	if err != nil {
		return nil, err
	}
	thecmd := exec.Command(command[0], command[1:]...)
	output, err := thecmd.CombinedOutput()
	if err != nil {
		err = DockerExecError{strings.Join(command, " "), err}
		return output, err
	}
	glog.V(1).Infof("Successfully ran command:'%s' output: %s\n", command, output)
	return output, nil
}

// generateDockerExecCommand returns a slice containing docker exec command to exec
func generateDockerExecCommand(containerID string, bashcmd []string, prependBash bool) ([]string, error) {
	if containerID == "" {
		return []string{}, fmt.Errorf("will not attach to container with empty containerID")
	}

	exeMap, err := exePaths([]string{"docker"})
	if err != nil {
		return []string{}, err
	}

	// TODO: add '-h' hostname to specify the container hostname when that
	// feature becomes available
	attachCmd := []string{exeMap["docker"], "exec"}

	if Isatty(os.Stdin) {
		attachCmd = append(attachCmd, "-t")
	}
	if Isatty(os.Stdout) && Isatty(os.Stdin) {
		attachCmd = append(attachCmd, "-i")
	}
	attachCmd = append(attachCmd, containerID)

	if prependBash {
		attachCmd = append(attachCmd, "/bin/bash", "-c", fmt.Sprintf("%s", strings.Join(bashcmd, " ")))
	} else {
		attachCmd = append(attachCmd, bashcmd...)
	}
	glog.V(1).Infof("attach command for container:%v command: %v\n", containerID, attachCmd)
	return attachCmd, nil
}

// hasFeatureDockerExec returns true if docker exec is supported
func hasFeatureDockerExec() bool {
	command := []string{"docker", "exec"}

	thecmd := exec.Command(command[0], command[1:]...)
	output, err := thecmd.CombinedOutput()
	// when docker exec is supported, we expect above 'docker exec' to fail,
	// but provide usage
	glog.V(1).Infof("Successfully ran command:'%s'  err: %s  output: %s\n", command, err, output)

	return strings.Contains(string(output), "Usage: docker exec")
}

// AttachAndRun attaches to a container and runs the command
func AttachAndRun(containerID string, bashcmd []string) ([]byte, error) {
	return RunDockerExec(containerID, bashcmd)
}

// AttachAndExec attaches to a container and execs the command
func AttachAndExec(containerID string, bashcmd []string) error {
	return ExecDockerExec(containerID, bashcmd)
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
