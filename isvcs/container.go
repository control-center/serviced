// Copyright 2014 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package agent implements a service that runs on a serviced node. It is
// responsible for ensuring that a particular node is running the correct services
// and reporting the state and health of those services back to the master
// serviced.

package isvcs

import (
	"github.com/control-center/serviced/commons/circular"
	"github.com/control-center/serviced/stats/cgroup"
	"github.com/control-center/serviced/utils"
	"github.com/rcrowley/go-metrics"
	"github.com/zenoss/glog"
	dockerclient "github.com/zenoss/go-dockerclient"

	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"path"
	"strconv"
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
	Command       func() string                       // the actual command to run inside the container
	Volumes       map[string]string                   // Volumes to bind mount in to the containers
	Ports         []int                               // Ports to expose to the host
	HealthCheck   func() error                        // A function to verify that the service is healthy
	Configuration map[string]interface{}              // A container specific configuration
	Notify        func(*Container, interface{}) error // A function to run when notified of a data event
	volumesDir    string                              // directory to store volume data
}

type Container struct {
	ContainerDescription
	client *dockerclient.Client
	ops    chan containerOpRequest // channel for communicating to the container's loop
}

func NewContainer(cd ContainerDescription) (*Container, error) {
	client, err := dockerclient.NewClient("unix:///var/run/docker.sock")
	if err != nil {
		glog.Errorf("Could not create docker client: %s", err)
		return nil, err
	}

	if len(cd.Name) == 0 || len(cd.Repo) == 0 || len(cd.Tag) == 0 || cd.Command == nil {
		return nil, ErrBadContainerSpec
	}

	if cd.Configuration == nil {
		cd.Configuration = make(map[string]interface{})
	}
	c := Container{
		ContainerDescription: cd,

		ops:    make(chan containerOpRequest),
		client: client,
	}

	envPerService[cd.Name] = make(map[string]string)

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
		return fmt.Sprintf("/tmp/serviced-%s/var/isvcs", user.Username)
	}
	return "/tmp/serviced/var/isvcs"
}

// loop maintains the state of the container; it handles requests to start() &
// stop() containers as well as detect container failures.
func (c *Container) loop() {

	var exitChan chan error
	var cmd *exec.Cmd
	statsExitChan := make(chan bool)

	for {
		select {
		case req := <-c.ops:
			switch req.op {
			case containerOpStop:
				glog.Infof("containerOpStop(): %s", c.Name)
				statsExitChan <- true
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

				healthCheckChan := make(chan error)
				go func() {
					if c.HealthCheck != nil {
						healthCheckChan <- c.HealthCheck() // run the HealthCheck if it exists
					} else {
						healthCheckChan <- nil
					}
				}()
				select {
				case exited := <-exitChan:
					req.response <- exited
				case healthCheck := <-healthCheckChan:
					if healthCheck == nil {
						go c.doStats(statsExitChan)
					}
					req.response <- healthCheck
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

type containerStat struct {
	Metric    string            `json:"metric"`
	Value     string            `json:"value"`
	Timestamp int64             `json:"timestamp"`
	Tags      map[string]string `json:"tags"`
}

// doStats
func (c *Container) doStats(exitChan chan bool) {
	registry := metrics.NewRegistry()
	id := ""
	tc := time.Tick(10 * time.Second)
	for {
		select {
		case _ = <-exitChan:
			return
		case t := <-tc:
			if id == "" {
				ids, err := c.getMatchingContainersIds()
				if err != nil {
					glog.Warningf("Error collecting isvc container IDs.")
				}
				if len(*ids) < 1 {
					break
				}
				id = (*ids)[0]
			}
			if cpuacctStat, err := cgroup.ReadCpuacctStat(cgroup.GetCgroupDockerStatsFilePath(id, cgroup.Cpuacct)); err != nil {
				glog.Warningf("Couldn't read CpuacctStat:", err)
				id = ""
				break
			} else {
				metrics.GetOrRegisterGauge("CpuacctStat.system", registry).Update(cpuacctStat.System)
				metrics.GetOrRegisterGauge("CpuacctStat.user", registry).Update(cpuacctStat.User)
			}
			if memoryStat, err := cgroup.ReadMemoryStat(cgroup.GetCgroupDockerStatsFilePath(id, cgroup.Memory)); err != nil {
				glog.Warningf("Couldn't read MemoryStat:", err)
				id = ""
				break
			} else {
				metrics.GetOrRegisterGauge("cgroup.memory.pgmajfault", registry).Update(memoryStat.Pgfault)
				metrics.GetOrRegisterGauge("cgroup.memory.totalrss", registry).Update(memoryStat.TotalRss)
				metrics.GetOrRegisterGauge("cgroup.memory.cache", registry).Update(memoryStat.Cache)
			}
			// Gather the stats.
			stats := []containerStat{}
			registry.Each(func(name string, i interface{}) {
				if metric, ok := i.(metrics.Gauge); ok {
					tagmap := make(map[string]string)
					tagmap["isvcname"] = c.Name
					stats = append(stats, containerStat{name, strconv.FormatInt(metric.Value(), 10), t.Unix(), tagmap})
				}
				if metricf64, ok := i.(metrics.GaugeFloat64); ok {
					tagmap := make(map[string]string)
					tagmap["isvcname"] = c.Name
					stats = append(stats, containerStat{name, strconv.FormatFloat(metricf64.Value(), 'f', -1, 32), t.Unix(), tagmap})
				}
			})
			// Post the stats.
			data, err := json.Marshal(stats)
			if err != nil {
				glog.Warningf("Error marshalling isvc stats json.")
				break
			}
			req, err := http.NewRequest("POST", "http://127.0.0.1:4242/api/put", bytes.NewBuffer(data))
			if err != nil {
				glog.Warningf("Error creating isvc stats request.")
				break
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				glog.V(4).Infof("Error making isvc stats request.")
				break
			}
			if strings.Contains(resp.Status, "204 No Content") == false {
				glog.Warningf("Couldn't post stats:", resp.Status)
				break
			}
		}
	}
}

// getMatchingContainersIds
func (c *Container) getMatchingContainersIds() (*[]string, error) {
	containers, err := c.client.ListContainers(dockerclient.ListContainersOptions{All: true})
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
	if ids, err := c.getMatchingContainersIds(); err != nil {
		return err
	} else {
		for _, id := range *ids {
			c.client.StopContainer(id, 20)
		}
	}
	return nil
}

// attempt to remove all matching containers
func (c *Container) rm() error {
	if ids, err := c.getMatchingContainersIds(); err != nil {
		return err
	} else {
		for _, id := range *ids {
			c.client.RemoveContainer(dockerclient.RemoveContainerOptions{ID: id})
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
	args = append(args, "run", "--rm", "--name", containerName)

	// attach all exported ports
	for _, port := range c.Ports {
		args = append(args, "-p", fmt.Sprintf("%d:%d", port, port))
	}

	// attach resources directory to all containers
	args = append(args, "-v", utils.ResourcesDir()+":"+"/usr/local/serviced/resources")

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

	for key, val := range envPerService[c.Name] {
		args = append(args, "-e", key+"="+val)
	}

	// set the image and command to run
	args = append(args, c.Repo+":"+c.Tag, "/bin/sh", "-c", c.Command())

	glog.V(1).Infof("Executing docker %s", args)
	var cmd *exec.Cmd
	tries := 5
	var err error
	const bufferSize = 1000
	lastStdout := circular.NewBuffer(bufferSize)
	lastStderr := circular.NewBuffer(bufferSize)
	for {
		if tries > 0 {
			cmd = exec.Command("docker", args...)
			if stdout, err := cmd.StdoutPipe(); err != nil {
				glog.Fatal("Could not open stdout pipe for launching isvc %s: %s", c.Name, err)
			} else {
				go io.Copy(lastStdout, stdout)
			}
			if stderr, err := cmd.StderrPipe(); err != nil {
				glog.Fatal("Could not open stderr pipe for launching isvc %s: %s", c.Name, err)
			} else {
				go io.Copy(lastStderr, stderr)
			}
			if err := cmd.Start(); err != nil {
				glog.Errorf("Could not start: %s", c.Name)
				c.stop()
				c.rm()
				time.Sleep(time.Second * 5)
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
		results := make([]byte, bufferSize)
		if n, err := lastStdout.Read(results); err == nil && n > 0 {
			glog.V(1).Infof("Stdout exited isvc %s: %s", c.Name, string(results))
		}
		if n, err := lastStderr.Read(results); err == nil && n > 0 {
			glog.V(1).Infof("Stdout exited isvc %s: %s", c.Name, string(results))
		}
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

// RunCommand runs a command inside the container.
func (c *Container) RunCommand(command []string) error {
	var id string
	ids, err := c.getMatchingContainersIds()
	if err != nil {
		glog.Warningf("Error collecting isvc container IDs.")
	}
	if len(*ids) == 0 {
		// Container hasn't started yet
		return fmt.Errorf("No docker container found for %s", c.Name)
	}
	id = (*ids)[0]
	output, err := utils.AttachAndRun(id, command)
	if err != nil {
		return err
	}
	os.Stdout.Write(output)
	return nil
}
