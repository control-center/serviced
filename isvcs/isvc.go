package isvcs

import (
	"github.com/zenoss/glog"

	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
)

type ISvc struct {
	Name       string
	Dockerfile string
	Tag        string
	Ports      []int
}

func (s *ISvc) exists() (bool, error) {

	cmd := exec.Command("docker", "images", s.Tag)
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}
	return strings.Contains(string(output), s.Tag), nil
}

func (s *ISvc) create() error {
	exists, err := s.exists()
	if err != nil || exists {
		return err
	}
	glog.Infof("Creating temp directory for building image: %s", s.Tag)
	tdir, err := ioutil.TempDir("", "isvc_")
	if err != nil {
		return err
	}
	dockerfile := tdir + "/Dockerfile"
	ioutil.WriteFile(dockerfile, []byte(s.Dockerfile), 0660)
	glog.Infof("building %s with dockerfile in %s", s.Tag, dockerfile)
	cmd := exec.Command("docker", "build", "-t", s.Tag, tdir)
	output, returnErr := cmd.CombinedOutput()
	if returnErr != nil {
		glog.Errorf("Problem running docker build: %s", string(output))
	}
	err = os.RemoveAll(tdir)
	if err != nil {
		glog.Warningf("Failed to cleanup directory :%s ", err)
	}
	return returnErr
}

func (s *ISvc) Running() (bool, error) {
	cmd := exec.Command("docker", "ps")
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}
	return strings.Contains(string(output), s.Tag+":latest"), nil
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

	containerId, err := s.getContainerId()
	if err != nil && !IsSvcNotFoundErr(err) {
		return err
	}

	var cmd *exec.Cmd
	if containerId != "" {
		cmd = exec.Command("docker", "start", containerId)
	} else {
		args := make([]string, len(s.Ports) * 2 + 3)
		glog.Errorf("About to build.")
		args[0] = "run"
		args[1] = "-d"
		for i, port := range(s.Ports) {
			args[2 + i] = "-p"
			args[2 + i + 1] = fmt.Sprintf("%d:%d", port, port)
		}
		args[len(s.Ports) * 2 + 2] = s.Tag
		cmd = exec.Command("docker", args...)
	}
	glog.Infof("Running docker cmd: %v", cmd)
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
	cmd := exec.Command("docker", "ps", "-a")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(output), "\n") {
		if strings.Contains(line, s.Tag+":latest") {
			fields := strings.Fields(line)
			return fields[0], nil
		}
	}
	return "", SvcNotFoundErr
}
