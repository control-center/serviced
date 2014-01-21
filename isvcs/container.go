package isvcs

import (
	"github.com/zenoss/glog"

	"errors"
	"os/exec"
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

func init() {
	ErrNotRunning = errors.New("container: not running")
	ErrRunning = errors.New("container: already running")
	ErrBadContainerSpec = errors.New("container: bad container specification")
}

type ContainerDescription struct {
	Name        string       // name of the container (used for docker named containers)
	Repo        string       // the repository the image for this container uses
	Tag         string       // the repository tag this container uses
	Command     string       // the actual command to run inside the container
	Volumes     []string     // Volumes to bind mount in to the containers
	Ports       []int        // Ports to expose to the host
	HealthCheck func() error // A function to verify that the service is healthy
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

	c.stop()
	c.rm()
	var exitChan chan error
	var cmd *exec.Cmd

	for {
		select {
		case req := <-c.ops:
			switch req.op {
			case containerOpStop:
				glog.Info("loop_OpStop()")
				if exitChan == nil {
					req.response <- ErrNotRunning
					continue
				}
				c.stop()
				c.rm()
				cmd.Process.Kill()
				cmd = nil
				exitChan = nil
				req.response <- nil

			case containerOpStart:
				glog.Info("loop_OpStart()")
				if cmd != nil {
					req.response <- ErrRunning
					continue
				}
				cmd, exitChan = c.run()
				req.response <- nil

			}
		case exitErr := <-exitChan:
			glog.Errorf("Unexpected failure of %s, got %s", c.Name, exitErr)
			exitChan = nil
		}
	}
}

func (c *Container) stop() error {
	cmd := exec.Command("docker", "stop", c.Name)
	return cmd.Run()
}

func (c *Container) rm() error {
	cmd := exec.Command("docker", "rm", c.Name)
	return cmd.Run()
}

func (c *Container) run() (*exec.Cmd, chan error) {
	cmd := exec.Command("docker", "run", "-rm", "-name", c.Name, c.Repo+":"+c.Tag, "/bin/sh", "-c", c.Command)
	exitChan := make(chan error)
	go func() {
		exitChan <- cmd.Run()
	}()
	return cmd, exitChan
}

func (c *Container) Start() error {
	glog.Info("calling Start()")
	errc := make(chan error)
	req := containerOpRequest{
		op:       containerOpStart,
		response: errc,
	}
	c.ops <- req
	return <-req.response
}

func (c *Container) Stop() error {
	errc := make(chan error)
	req := containerOpRequest{
		op:       containerOpStop,
		response: errc,
	}
	c.ops <- req
	return <-req.response
}
