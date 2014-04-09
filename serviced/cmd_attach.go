package main

import (
	"github.com/zenoss/glog"
	//"github.com/zenoss/serviced/dao"

	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

// exePaths returns the full path to the given executables in a map
func exePaths(exes []string) (map[string]string, error) {
	exeMap := map[string]string{}

	for _, exe := range exes {
		path, err := exec.LookPath(exe)
		if err != nil {
			fmt.Printf("exe:'%v' not found err: %v\n", exe, err)
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
	glog.Infof("exec cmd: %v\n", fullCmd)
	return syscall.Exec(fullCmd[0], fullCmd[0:], os.Environ())
}

// CmdAttach attaches to a service container and runs the given arbitrary bash command
func (cli *ServicedCli) CmdAttach(args ...string) error {

	cmd := Subcmd("attach", "CONTAINER_ID BASH_CMD ...", "attach to a service container and run command")

	if err := cmd.Parse(args); err != nil {
		return err
	}

	if len(cmd.Args()) < 2 {
		glog.Errorf("len(args) = %d; missing command to run\n", len(cmd.Args()))
		os.Exit(255)
	}

	container := cmd.Arg(0)
	command := cmd.Args()[1:]

	if err := attachContainerAndExec(container, command); err != nil {
		glog.Errorf("error running command:%v  err:%v\n", cmd.Args(), err)
		os.Exit(254)
	}

	return nil
}
