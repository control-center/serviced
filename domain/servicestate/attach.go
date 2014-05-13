package servicestate

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/zenoss/glog"
)

const (
	nsInitRoot = "/var/lib/docker/execdriver/native"
)

var sudo, nsinit, bash string

func init() {
	exeMap, err := exePaths("bash", "nsinit", "sudo")
	if err != nil {
		panic(err)
	}
	bash = exeMap["bash"]
	nsinit = exeMap["nsinit"]
	sudo = exeMap["sudo"]
}

func exePaths(exes ...string) (map[string]string, error) {
	exeMap := make(map[string]string)

	for _, exe := range exes {
		path, err := exec.LookPath(exe)
		if err != nil {
			glog.Errorf("%s not found: %s", exe, err)
			return nil, err
		}
		exeMap[exe] = path
	}
	return exeMap, nil
}

func (ss *ServiceState) Attach(command ...string) *exec.Cmd {
	return attach(ss.DockerId, command)
}

func attach(containerID string, command []string) *exec.Cmd {
	attachCmd := []string{"--", bash}
	if len(command) > 0 {
		cmd := fmt.Sprintf("cd %s/%s && %s exec %s %s", nsInitRoot, containerID, nsinit, strings.Join(command, " "))
		attachCmd = append(attachCmd, "-c", cmd)
	}
	return exec.Command(sudo, attachCmd...)
}