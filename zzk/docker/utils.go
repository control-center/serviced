package docker

import (
	"fmt"
	"os/exec"
	"path"
	"strings"

	"github.com/zenoss/serviced/coordinator/client"
)

const (
	zkDocker = "/docker"
	zkAction = "/action"
	zkShell  = "/shell"

	nsInitRoot      = "/var/lib/docker/execdriver/native"
	urandomFilename = "/dev/urandom"
)

func mkdir(conn client.Connection, dirpath string) error {
	if exists, err := conn.Exists(dirpath); err != nil && err != client.ErrNoNode {
		return err
	} else if exists {
		return nil
	} else if err := mkdir(conn, path.Dir(dirpath)); err != nil {
		return err
	}
	return conn.CreateDir(dirpath)
}

func attach(containerID string, command []string) (*exec.Cmd, error) {
	bash, err := exec.LookPath("bash")
	if err != nil {
		return nil, fmt.Errorf("bash not found: %s", err)
	}
	nsinit, err := exec.LookPath("nsinit")
	if err != nil {
		return nil, fmt.Errorf("nsinit not found: %s", err)
	}
	sudo, err := exec.LookPath("sudo")
	if err != nil {
		return nil, fmt.Errorf("sudo not found: %s", err)
	}

	attachCmd := []string{sudo, "--", bash, "-c"}
	bashcmd := fmt.Sprintf("cd %s/%s && %s exec bash", nsInitRoot, containerID, nsinit)
	if len(command) > 0 {
		bashcmd = bashcmd + fmt.Sprintf(" -c \"%s\"", strings.Join(command, " "))
	}
	attachCmd = append(attachCmd, bashcmd)
	fmt.Println(attachCmd)
	return exec.Command(sudo, attachCmd...), nil
}

func shell(imageID string, command []string) *exec.Cmd {
	return nil
}