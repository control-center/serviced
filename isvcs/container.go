package isvcs

import (
	"github.com/fsouza/go-dockerclient"
	"github.com/zenoss/glog"

	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

type containerOp int

const (
	containerOpStart containerOp = iota
	containerOpStop
)

type containerOpRequest struct {
	op       containerOp
	response chan error
}

var ErrNotRunning error
var ErrRunning error
var ErrBadContainerSpec error
var randomSource string

func init() {
	ErrNotRunning = errors.New("container: not running")
	ErrRunning = errors.New("container: already running")
	ErrBadContainerSpec = errors.New("container: bad container specification")

	randomSource = "/dev/urandom"
}

type ContainerDescription struct {
	Name        string            // name of the container (used for docker named containers)
	Repo        string            // the repository the image for this container uses
	Tag         string            // the repository tag this container uses
	Command     string            // the actual command to run inside the container
	Volumes     map[string]string // Volumes to bind mount in to the containers
	Ports       []int             // Ports to expose to the host
	HealthCheck func() error      // A function to verify that the service is healthy
}

type Container struct {
	ContainerDescription
	ops chan containerOpRequest // channel for communicating to the container's loop
}

func NewContainer(cd ContainerDescription) (*Container, error) {
	if len(cd.Name) == 0 || len(cd.Repo) == 0 || len(cd.Tag) == 0 || len(cd.Command) == 0 {
		return nil, ErrBadContainerSpec
	}
	c := Container{
		ContainerDescription: cd,
		ops:                  make(chan containerOpRequest),
	}
	go c.loop()
	return &c, nil
}

func (c *Container) loop() {

	var exitChan chan error
	var cmd *exec.Cmd

	for {
		select {
		case req := <-c.ops:
			switch req.op {
			case containerOpStop:
				glog.Infof("containerOpStop(): %s", c.Name)
				if exitChan == nil {
					req.response <- ErrNotRunning
					continue
				}
				oldCmd := cmd
				cmd = nil
				exitChan = nil
				c.stop()
				c.rm()
				oldCmd.Process.Kill()
				req.response <- nil

			case containerOpStart:
				glog.Infof("containerOpStart(): %s", c.Name)
				if cmd != nil {
					req.response <- ErrRunning
					continue
				}
				c.stop()
				c.rm()
				cmd, exitChan = c.run()
				if c.HealthCheck != nil {
					req.response <- c.HealthCheck()
				} else {
					req.response <- nil
				}

			}
		case exitErr := <-exitChan:
			docker := exec.Command("docker", "logs", c.Name)
			output, _ := docker.CombinedOutput()
			glog.Errorf("isvc:%s, %s", c.Name, string(output))
			glog.Errorf("Unexpected failure of %s, got %s", c.Name, exitErr)
			time.Sleep(time.Second * 30)
			glog.Fatalf("iscv:%s, process exited: %s", cmd.ProcessState.Exited())
			cmd, exitChan = c.run()
		}
	}
}

func (c *Container) stop() error {
	client, err := newDockerClient("unix:///var/run/docker.sock")
	if err != nil {
		glog.Errorf("Could not create docker client: %s", err)
		return err
	}
	containers, err := client.ListContainers(docker.ListContainersOptions{All: true})
	if err != nil {
		return err
	}
	for _, container := range containers {
		for _, name := range container.Names {
			if strings.HasPrefix(name, "/"+c.Name) {
				client.StopContainer(container.ID, 20)
			}
		}
	}
	return nil
}

func (c *Container) rm() error {
	client, err := newDockerClient("unix:///var/run/docker.sock")
	if err != nil {
		glog.Errorf("Could not create docker client: %s", err)
		return err
	}
	containers, err := client.ListContainers(docker.ListContainersOptions{All: true})
	if err != nil {
		return err
	}
	for _, container := range containers {
		for _, name := range container.Names {
			if strings.HasPrefix(name, "/"+c.Name) {
				err = client.RemoveContainer(container.ID)
			}
		}
	}
	return nil
}

func isDir(path string) (bool, error) {
	stat, err := os.Stat(path)
	if err == nil {
		return stat.IsDir(), nil
	} else {
		if os.IsNotExist(err) {
			return false, nil
		}
	}
	return false, err
}

func uuid() string {
	f, _ := os.Open(randomSource)
	b := make([]byte, 16)
	f.Read(b)
	f.Close()
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])

}

func (c *Container) run() (*exec.Cmd, chan error) {

	containerName := c.Name + "-" + uuid()
	exitChan := make(chan error)
	args := make([]string, 0)
	args = append(args, "run", "-rm", "-name", containerName)
	for _, port := range c.Ports {
		args = append(args, "-p", fmt.Sprintf("%d:%d", port, port))
	}
	for name, volume := range c.Volumes {
		hostDir := fmt.Sprintf("/tmp/serviced/%s/%s", c.Name, name)
		if exists, _ := isDir(hostDir); !exists {
			if err := os.MkdirAll(hostDir, 0777); err != nil {
				glog.Errorf("could not create %s on host: %s", hostDir, err)
				exitChan <- err
				return nil, exitChan
			}
		}
		args = append(args, "-v", hostDir+":"+volume)
	}
	args = append(args, c.Repo+":"+c.Tag, "/bin/sh", "-c", c.Command)

	glog.V(1).Infof("Executing docker %s", args)
	cmd := exec.Command("docker", args...)
	go func() {
		exitChan <- cmd.Run()
	}()
	return cmd, exitChan
}

func (c *Container) Start() error {
	glog.Infof("entering Start() for %s", c.Name)
	defer glog.Infof("leaving Start() for %s", c.Name)
	req := containerOpRequest{
		op:       containerOpStart,
		response: make(chan error),
	}
	c.ops <- req
	return <-req.response
}

func (c *Container) Stop() error {
	glog.Infof("calling Stop() for %s", c.Name)
	req := containerOpRequest{
		op:       containerOpStop,
		response: make(chan error),
	}
	c.ops <- req
	return <-req.response
}
