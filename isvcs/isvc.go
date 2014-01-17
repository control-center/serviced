package isvcs

import (
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced"

	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"
)

type ISvc struct {
	Name       string
	Repository string
	Tag        string
	Ports      []int
	Volumes    []string
	shutdown   chan chan error
}

func NewISvc(name, repository, tag string, ports []int, volumes []string) (s ISvc) {
	s = ISvc{
		Name:       name,
		Repository: repository,
		Tag:        tag,
		Ports:      ports,
		Volumes:    volumes,
	}
	s.shutdown = make(chan chan error)
	return s
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

	if _, err := os.Stat(imageTar); err != nil {
		if os.IsNotExist(err) {
			glog.Errorf("Could not locate: %s", imageTar)
		}
		return err
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

func (s *ISvc) getContainerId() string {
	return "serviced_" + s.Name
}

func (s *ISvc) Run() {
	go s.RunAndWait()
}

func (s *ISvc) killAndRemove() {
	cmd := exec.Command("docker", "ps", "-a")
	if output, err := cmd.Output(); err == nil {
		for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
			if strings.HasSuffix(strings.TrimSpace(line), s.getContainerId()) {
				fields := strings.Fields(line)
				glog.Infof("About to kill isvc %s, %s", s.Name, fields[0])
				cmd = exec.Command("docker", "kill", strings.TrimSpace(fields[0]))
				cmd.Run()
				cmd = exec.Command("docker", "rm", strings.TrimSpace(fields[0]))
				cmd.Run()
				cmd = exec.Command("docker", "rm", strings.TrimSpace(s.getContainerId()))
				cmd.Run()
			}
		}
	}
}

func (s *ISvc) RunAndWait() {

	s.killAndRemove()

	args := "docker run -rm -name=" + s.getContainerId()
	// add the ports to the arg list
	for _, port := range s.Ports {
		args += fmt.Sprintf(" -p %d:%d", port, port)
	}

	for _, volume := range s.Volumes {
		hostDir := fmt.Sprintf("/tmp/serviced/%s", s.Name)
		if err := os.MkdirAll(hostDir, 0700); err != nil {
			panic(err)
		}
		args += fmt.Sprintf(" -v %s:%s", hostDir, volume)
	}

	// bind mount the resources directory, always make it /usr/local/serviced to simplify the dockerfile commands
	containerServiceDResources := "/usr/local/serviced/resources"
	args += " -v"
	args += fmt.Sprintf(" %s:%s", resourcesDir(), containerServiceDResources)

	// specify the image
	args += fmt.Sprintf(" %s:%s", s.Repository, s.Tag)
	cmd := exec.Command("sh", "-c", args)
	glog.Infof("About to start isvc %s : %s", s.Name, args)
	cmd.Start()

	done := make(chan error)
	go func() {
		done <- cmd.Wait()
	}()
	select {
	case errc := <-s.shutdown:
		done = nil
		glog.Infof("Shutting down process: %s", s.Name)
		errc <- cmd.Process.Kill()
		s.killAndRemove()
	case err := <-done:
		if err != nil {
			glog.Fatalf("Unexpected failure of isvc %s, try docker logs %s", err, s.getContainerId())
		}
		s.killAndRemove()
	}
}

func (s *ISvc) Stop() error {
	errc := make(chan error)
	s.shutdown <- errc
	err := <-errc
	s.killAndRemove()
	return err
}

func (s *ISvc) Kill() error {
	return s.Stop()
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

func localDir(p string) string {
	homeDir := serviced.ServiceDHome()
	if len(homeDir) == 0 {
		_, filename, _, _ := runtime.Caller(1)
		homeDir = path.Dir(filename)
	}
	return path.Join(homeDir, p)
}

func imagesDir() string {
	return localDir("images")
}

func resourcesDir() string {
	return localDir("resources")
}
