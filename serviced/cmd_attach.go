package main

import (
	"github.com/zenoss/glog"
	//"github.com/zenoss/serviced/dao"

	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"syscall"
)

// exePaths returns the full path to the given executables in a map
func exePaths(exes []string) (map[string]string, error) {
	exeMap := map[string]string{}

	for _, exe := range exes {
		path, err := exec.LookPath(exe)
		if err != nil {
			glog.Errorf("exe:'%v' not found err: %v\n", exe, err)
			return nil, err
		}

		exeMap[exe] = path
	}

	return exeMap, nil
}

// attachContainerAndExec connects to a container and executes an arbitrary bash command
func attachContainerAndExec(containerId string, cmd []string) error {
	exeMap, err := exePaths([]string{"sudo", "nsinit"})
	if err != nil {
		return err
	}

	NSINIT_ROOT := "/var/lib/docker/execdriver/native" // has container.json

	attachCmd := fmt.Sprintf("cd %s/%s && %s exec %s", NSINIT_ROOT, containerId,
		exeMap["nsinit"], strings.Join(cmd, " "))
	fullCmd := []string{exeMap["sudo"], "--", "/bin/bash", "-c", attachCmd}
	glog.V(1).Infof("exec cmd: %v\n", fullCmd)
	return syscall.Exec(fullCmd[0], fullCmd[0:], os.Environ())
}

// findContainerIdFromDocker returns the containerID that matches docker ps output
func findContainerIdFromDocker(pattern string) (string, error) {
	// algorithm - perform following shell one-liner in this short 40 line go function :)
	//   docker ps --no-trunc | awk '/serviced.*redis/{print $1;exit}'
	exeMap, err := exePaths([]string{"docker"})
	if err != nil {
		return "", err
	}

	CMD := exeMap["docker"]
	argv := []string{"ps", "--no-trunc"}
	glog.V(1).Infof("running command: '%s %s'", CMD, argv)
	cmd := exec.Command(CMD, argv...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		glog.Errorf("Error setting up command: '%v %v'  err: %s", CMD, argv, err)
		return "", err
	}

	if err := cmd.Start(); err != nil {
		glog.Errorf("Error starting command: '%v %v'  err: %s", CMD, argv, err)
		return "", err
	}

	glog.V(1).Infof("looking for container matching pattern: %s", pattern)
	reader := bufio.NewReader(stdout)
	for {
		line, err := reader.ReadString('\n')
		if err == io.EOF && len(line) == 0 {
			break
		}
		matched, err := regexp.MatchString(pattern, line)
		if matched {
			return strings.Split(line, " ")[0], nil
		}
	}

	return "", errors.New(fmt.Sprintf("could not find container from pattern: %v", pattern))
}

// CmdAttach attaches to a service container and runs the given arbitrary bash command
func (cli *ServicedCli) CmdAttach(args ...string) error {
	cmd := Subcmd("attach", "BASH_CMD ...", "attach to a service container and run command")

	var containerID string
	cmd.StringVar(&containerID, "containerID", "", "attach to container given containerID")

	var pattern string
	cmd.StringVar(&pattern, "pattern", "", "attach to first container found by matching pattern in docker ps")

	if err := cmd.Parse(args); err != nil {
		return err
	}

	if len(cmd.Args()) < 1 {
		return errors.New(fmt.Sprintf("missing bash command to run\n"))
	}

	if len(containerID) <= 0 {
		if len(pattern) > 0 {
			var err error
			// TODO: find containerID from service state that matches user supplied pattern
			containerID, err = findContainerIdFromDocker(pattern)
			if err != nil {
				return err
			}
		} else {
			return errors.New(fmt.Sprintf("neither containerID nor pattern is specified\n"))
		}
	}

	bashCommand := cmd.Args()[0:]

	if err := attachContainerAndExec(containerID, bashCommand); err != nil {
		return errors.New(fmt.Sprintf("error running bash command:%v  err:%v\n", cmd.Args(), err))
	}

	return nil
}
