package isvcs

import (
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced"

	"bufio"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path"
	"runtime"
	"strings"
)

type ISvc struct {
	Name       string
	Repository string
	Tag        string
	Ports      []int
}

func (s *ISvc) exists() (bool, error) {

	cmd := exec.Command("sh", "-c", fmt.Sprintf("docker images -a %s | tail -n +2 | awk '{ print $2 }'", s.Repository))
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}
	tagsFound := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, tag := range tagsFound {
		if tag == s.Tag {
			return true, nil
		}
	}
	return false, nil
}

func (s *ISvc) create() error {
	if exists, err := s.exists(); err != nil || exists {
		return err
	}
	glog.V(1).Infof("Looking for existing tar export of %s:%s", s.Repository, s.Tag)
	imageTar := fmt.Sprintf("%s/%s/%s.tar", imagesDir(), s.Repository, s.Tag)

	if _, err := os.Stat(imageTar); os.IsNotExist(err) {
		glog.Errorf("Could not locate: %s", imageTar)
		return err
	} else {
		if err != nil {
			return err
		}
	}

	file, err := os.Open(imageTar)
	if err != nil {
		glog.Errorf("Could not open %s: %s", imageTar, err)
		return err
	}

	cmd := exec.Command("docker", "load")
	cmd.Stdin = bufio.NewReader(file)
	glog.V(1).Infof("Importing docker image %s", imageTar)
	if err := cmd.Run(); err != nil {
		glog.Errorf("Could not load %s: %s", imageTar, err)
		return err
	}
	return nil
}

func (s *ISvc) Running() (bool, error) {
	containerId, _ := s.getContainerId()
	if len(containerId) == 0 {
		return false, nil
	}
	cmd := exec.Command("docker", "ps")
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}
	return strings.Contains(string(output), containerId), nil
}

func (s *ISvc) Run() error {

	err := s.create()
	if err != nil {
		return err
	}

	running, err := s.Running()
	if err != nil || running {
		return err
	}
	glog.Infof("%s is not running", s.Repository)

	containerId, err := s.getContainerId()
	if err != nil && !IsSvcNotFoundErr(err) {
		return err
	}

	var cmd *exec.Cmd
	if containerId != "" {
		cmd = exec.Command("docker", "start", containerId)
	} else {
		args := "docker run -d "
		// add the ports to the arg list
		for _, port := range s.Ports {
			args += fmt.Sprintf(" -p %d:%d", port, port)
		}
		// bind mount the resources directory, always make it /usr/local/serviced to simplify the dockerfile commands
		containerServiceDResources := "/usr/local/serviced/resources"
		args += " -v"
		args += fmt.Sprintf(" %s:%s", resourcesDir(), containerServiceDResources)

		// specify the image
		args += fmt.Sprintf(" %s:%s", s.Repository, s.Tag)
		cmd = exec.Command("sh", "-c", args)
	}
	glog.Info("Running docker cmd: ", cmd)
	return cmd.Run()
}

func (s *ISvc) Stop() error {
	containerId, err := s.getContainerId()
	if err != nil {
		return err
	}
	cmd := exec.Command("docker", "stop", containerId)
	return cmd.Run()
}

func (s *ISvc) Kill() error {
	containerId, err := s.getContainerId()
	if err != nil {
		return err
	}
	cmd := exec.Command("docker", "kill", containerId)
	return cmd.Run()
}

var SvcNotFoundErr error
var IsSvcNotFoundErr func(error) bool

const msgSvcNotFoundErr = "svc not found"

func init() {
	SvcNotFoundErr = fmt.Errorf(msgSvcNotFoundErr)
	IsSvcNotFoundErr = func(err error) bool {
		if err == nil {
			return false
		}
		return err.Error() == msgSvcNotFoundErr
	}
}

func (s *ISvc) getContainerId() (string, error) {
	cmd := exec.Command("sh", "-c", `docker ps -a | tail -n +2 | awk '{ print $1 " " $2 }'`)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	repoAndName := s.Repository + ":" + s.Tag
	for _, line := range strings.Split(string(output), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[1] == repoAndName {
			return fields[0], nil
		}
	}
	return "", SvcNotFoundErr
}

func localDir(p string) string {
	homeDir := serviced.ServiceDHome()
	if len(homeDir) == 0 {
		_, filename, _, _ := runtime.Caller(1)
		homeDir = path.Dir(filename)
	}
	return path.Join(homeDir, p)
}

func resourcesDir() string {
	return localDir("resources")
}

func imagesDir() string {
	homeDir := serviced.ServiceDHome()
	if len(homeDir) == 0 {
		current, err := user.Current()
		if err != nil {
			panic("Could not get current user info")
		}
		return fmt.Sprintf("/tmp/serviced-%s-tmp/images", current.Username)
	}
	return path.Join(homeDir, "images")
}
