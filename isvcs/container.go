/*******************************************************************************
* Copyright (C) Zenoss, Inc. 2013, 2014 all rights reserved.
*
* This content is made available according to terms specified in
* License.zenoss under the directory where your Zenoss product is installed.
*
*******************************************************************************/

package isvcs

import (
	"github.com/fsouza/go-dockerclient"
	"github.com/zenoss/glog"

	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path"
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

func init() {
	ErrNotRunning = errors.New("container: not running")
	ErrRunning = errors.New("container: already running")
	ErrBadContainerSpec = errors.New("container: bad container specification")
}

type ContainerDescription struct {
	Name          string                              // name of the container (used for docker named containers)
	Repo          string                              // the repository the image for this container uses
	Tag           string                              // the repository tag this container uses
	Command       string                              // the actual command to run inside the container
	Volumes       map[string]string                   // Volumes to bind mount in to the containers
	Ports         []int                               // Ports to expose to the host
	HealthCheck   func() error                        // A function to verify that the service is healthy
	Configuration interface{}                         // A container specific configuration
	Notify        func(*Container, interface{}) error // A function to run when notified of a data event
	volumesDir    string                              // directory to store volume data
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

func (c *Container) SetVolumesDir(volumesDir string) {
	c.volumesDir = volumesDir
}

func (c *Container) VolumesDir() string {
	if len(c.volumesDir) > 0 {
		return c.volumesDir
	}
	if user, err := user.Current(); err == nil {
		return fmt.Sprintf("/tmp/serviced-%s/isvcs_volumes", user.Username)
	}
	return "/tmp/serviced/isvcs_volumes"
}

// loop maintains the state of the container; it handles requests to start() &
// stop() containers as well as detect container failures.
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
				exitChan = nil        // setting extChan to nil will disable reading from it in the select()
				oldCmd.Process.Kill() // kill the docker run() wrapper
				c.stop()              // stop the container if it's not already stopped
				c.rm()                // remove the container if it's not already gone
				req.response <- nil

			case containerOpStart:
				glog.Infof("containerOpStart(): %s", c.Name)
				if cmd != nil {
					req.response <- ErrRunning
					continue
				}
				c.stop()                // stop the container, if it's not stoppped
				c.rm()                  // remove it if it was not already removed
				cmd, exitChan = c.run() // run the actual container
				if c.HealthCheck != nil {
					req.response <- c.HealthCheck() // run the HealthCheck if it exists
				} else {
					req.response <- nil
				}

			}
		case exitErr := <-exitChan:
			glog.Errorf("Unexpected failure of %s, got %s", c.Name, exitErr)
			c.stop()                // stop the container, if it's not stoppped
			c.rm()                  // remove it if it was not already removed
			cmd, exitChan = c.run() // run the actual container
		}
	}
}

// getMatchingContainersIds
func (c *Container) getMatchingContainersIds(client *docker.Client) (*[]string, error) {
	containers, err := client.ListContainers(docker.ListContainersOptions{All: true})
	if err != nil {
		return nil, err
	}
	matching := make([]string, 0)
	for _, container := range containers {
		for _, name := range container.Names {
			if strings.HasPrefix(name, "/"+c.Name) {
				matching = append(matching, container.ID)
			}
		}
	}
	return &matching, nil
}

// attempt to stop all matching containers
func (c *Container) stop() error {
	client, err := newDockerClient("unix:///var/run/docker.sock")
	if err != nil {
		glog.Errorf("Could not create docker client: %s", err)
		return err
	}
	if ids, err := c.getMatchingContainersIds(client); err != nil {
		return err
	} else {
		for _, id := range *ids {
			client.StopContainer(id, 20)
		}
	}
	return nil
}

// attempt to remove all matching containers
func (c *Container) rm() error {
	client, err := newDockerClient("unix:///var/run/docker.sock")
	if err != nil {
		glog.Errorf("Could not create docker client: %s", err)
		return err
	}
	if ids, err := c.getMatchingContainersIds(client); err != nil {
		return err
	} else {
		for _, id := range *ids {
			client.RemoveContainer(id)
		}
	}
	return nil
}

// Run() an instance of this container and return it's exec.Command reference and a
// channel that sends the exit code, when the container exits
func (c *Container) run() (*exec.Cmd, chan error) {

	// the container name is semi random because containers can get wedged
	// in docker and can not be removed until a reboot (or aufs trickery)
	containerName := c.Name + "-" + uuid()

	exitChan := make(chan error, 1)
	args := make([]string, 0)
	args = append(args, "run", "-rm", "-name", containerName)

	// attach all exported ports
	for _, port := range c.Ports {
		args = append(args, "-p", fmt.Sprintf("%d:%d", port, port))
	}

	// attach resources directory to all containers
	args = append(args, "-v", resourcesDir()+":"+"/usr/local/serviced/resources")

	// attach all exported volumes
	for name, volume := range c.Volumes {
		hostDir := path.Join(c.VolumesDir(), c.Name, name)
		if exists, _ := isDir(hostDir); !exists {
			if err := os.MkdirAll(hostDir, 0777); err != nil {
				glog.Errorf("could not create %s on host: %s", hostDir, err)
				exitChan <- err
				return nil, exitChan
			}
		}
		args = append(args, "-v", hostDir+":"+volume)
	}

	// set the image and command to run
	args = append(args, c.Repo+":"+c.Tag, "/bin/sh", "-c", c.Command)

	glog.V(1).Infof("Executing docker %s", args)
	var cmd *exec.Cmd
	tries := 5
	for {
		var err error
		if tries > 0 {
			cmd = exec.Command("docker", args...)
			if err := cmd.Start(); err != nil {
				glog.Errorf("Could not start: %s", c.Name)
				c.stop()
				c.rm()
				time.Sleep(time.Second * 1)
			} else {
				break
			}
		} else {
			exitChan <- err
			return cmd, exitChan
		}
		tries = -1
	}
	go func() {
		exitChan <- cmd.Wait()
	}()
	return cmd, exitChan
}

// Start() a container by sending the loop() a request
func (c *Container) Start() error {
	req := containerOpRequest{
		op:       containerOpStart,
		response: make(chan error),
	}
	c.ops <- req
	return <-req.response
}

// Stop() a container by sending the loop() a request
func (c *Container) Stop() error {
	req := containerOpRequest{
		op:       containerOpStop,
		response: make(chan error),
	}
	c.ops <- req
	return <-req.response
}
