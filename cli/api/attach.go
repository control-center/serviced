package api

import (
	"github.com/zenoss/glog"

	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

// ServiceAttachConfig is the deserialized object from the command-line
type ServiceAttachConfig struct {
	DockerId string
	Command  []string
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

// ServiceAttach runs an arbitrary shell command in a running service container
func (a *api) ServiceAttach(config ServiceAttachConfig) error {
	containerID := config.DockerId
	command := config.Command
	if err := attachContainerAndExec(containerID, command); err != nil {
		fmt.Fprintf(os.Stderr, "error running bash command:'%v'  error:%v\n", command, err)
		return err
	}

	return nil
}
