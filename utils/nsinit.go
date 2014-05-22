package utils

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/zenoss/glog"
)

var BASH_SCRIPT = `
DIR="$(cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"
export SEENFILE="${DIR}/$$.found"
trap "rm -f ${SEENFILE} ${BASH_SOURCE[0]}" EXIT
for i in {1..10}; do
	rm -f ${SEENFILE}
	{{{{COMMAND}}}} 2> >(awk '/setns /{print >ENVIRON["SEENFILE"];next} {print}' >&2)
	RESULT=$?
	sleep 0.1  # allow time to flush for awk writefile
	[ -s "${SEENFILE}" ] || exit ${RESULT}
done
{{{{COMMAND}}}}
exit $?
`

func createWrapperScript(cmd []string) ([]string, error) {
	f, err := ioutil.TempFile("", "nsinit")
	if err != nil {
		return nil, err
	}
	defer f.Close()
	script := strings.Replace(BASH_SCRIPT, "{{{{COMMAND}}}}", strings.Join(cmd, " "), -1)
	if _, err := f.WriteString(script); err != nil {
		return nil, err
	}
	if err := f.Sync(); err != nil {
		return nil, err
	}
	command := []string{"/usr/bin/sudo", "/bin/bash", f.Name()}
	return command, nil
}

func ExecNSInitWithRetry(containerID string, bashcmd []string) error {
	cmd, err := generateAttachCommand(containerID, bashcmd)
	if err != nil {
		return err
	}
	command, err := createWrapperScript(cmd)
	if err != nil {
		return err
	}
	return syscall.Exec(command[0], command[0:], os.Environ())
}

func RunNSInitWithRetry(containerID string, bashcmd []string) ([]byte, error) {
	cmd, err := generateAttachCommand(containerID, bashcmd)
	if err != nil {
		return nil, err
	}
	command, err := createWrapperScript(cmd)
	if err != nil {
		return nil, err
	}
	thecmd := exec.Command(command[0], command[1:]...)
	output, err := thecmd.CombinedOutput()
	if err != nil {
		glog.Errorf("Error running command:'%s' output: %s  error: %s\n", cmd, output, err)
		return output, err
	}
	glog.V(1).Infof("Successfully ran command:'%s' output: %s\n", cmd, output)
	return output, nil
}

// generateAttachCommand returns a slice containing nsinit command to exec
func generateAttachCommand(containerID string, bashcmd []string) ([]string, error) {
	if containerID == "" {
		return []string{}, fmt.Errorf("will not attach to container with empty containerID")
	}

	exeMap, err := exePaths([]string{"sudo", "nsinit"})
	if err != nil {
		return []string{}, err
	}

	nsInitRoot := "/var/lib/docker/execdriver/native" // has container.json

	attachCmd := fmt.Sprintf("cd %s/%s && %s exec %s", nsInitRoot, containerID,
		exeMap["nsinit"], strings.Join(bashcmd, " "))
	fullCmd := []string{attachCmd}
	glog.V(1).Infof("attach command for container:%v command: %v\n", containerID, fullCmd)
	return fullCmd, nil
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