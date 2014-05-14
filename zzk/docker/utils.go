package docker

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/zenoss/serviced/coordinator/client"
)

const (
	zkDocker = "/docker"
	zkAttach = "/attach"
	zkShell  = "/shell"

	nsInitRoot      = "/var/lib/docker/execdriver/native"
	urandomFilename = "/dev/urandom"
)

var bash, nsinit, sudo string

func newuuid() string {
	f, err := os.Open(urandomFilename)
	if err != nil {
		panic(err)
	}
	b := make([]byte, 16)
	defer f.Close()
	f.Read(b)
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

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

func attach(containerID string, command []string) *exec.Cmd {
	attachCmd := []string{"--", bash}
	if len(command) > 0 {
		cmd := fmt.Sprintf("cd %s/%s && %s exec %s %s", nsInitRoot, containerID, nsinit, strings.Join(command, " "))
		attachCmd = append(attachCmd, "-c", cmd)
	}
	return exec.Command(sudo, attachCmd...)
}

func shell(imageID string, command []string) *exec.Cmd {
	return nil
}

func init() {
	var err error
	bash, err = exec.LookPath("bash")
	if err != nil {
		panic(fmt.Errorf("bash not found: %s", err))
	}
	nsinit, err = exec.LookPath("nsinit")
	if err != nil {
		panic(fmt.Errorf("nsinit not found: %s", err))
	}
	sudo, err = exec.LookPath("sudo")
	if err != nil {
		panic(fmt.Errorf("sudo not found: %s", err))
	}
}