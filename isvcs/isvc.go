package isvcs

import (
	"github.com/zenoss/glog"

	"fmt"
	"os/exec"
	"strings"
)

type ISvc struct {
	Name       string
	Dockerfile string
	Tag        string
}

func (s *ISvc) Exists() (bool, error) {

	cmd := exec.Command("docker", "images", s.Tag)
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}
	return strings.Contains(string(output), s.Tag), nil
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
		cmd = exec.Command("docker", "run", "-d", s.Tag)
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
