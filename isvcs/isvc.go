package isvcs

import (
	"fmt"
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"
)

type ISvc struct {
	Name       string
	Dockerfile string
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
	exists, err := s.exists()
	if err != nil || exists {
		return err
	}
	glog.V(1).Infof("Creating temp directory for building image: %s:%s", s.Repository, s.Tag)
	tdir, err := ioutil.TempDir("", "isvc_")
	if err != nil {
		return err
	}
	dockerfile := tdir + "/Dockerfile"
	ioutil.WriteFile(dockerfile, []byte(s.Dockerfile), 0660)
	glog.V(0).Infof("building %s:%s with dockerfile in %s", s.Repository, s.Tag, dockerfile)
	cmd := exec.Command("docker", "build", "-t", s.Repository + ":" + s.Tag, tdir)
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

func resourcesDir() string {
	homeDir := serviced.ServiceDHome()
	if len(homeDir) == 0 {
		_, filename, _, _ := runtime.Caller(1)
		homeDir = path.Dir(filename)
	}
	return path.Join(homeDir, "resources")
}
