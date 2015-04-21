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
	"github.com/control-center/serviced/commons"
	"github.com/control-center/serviced/commons/docker"
	"github.com/control-center/serviced/domain"
	"github.com/control-center/serviced/stats/cgroup"
	"github.com/control-center/serviced/utils"
	"github.com/rcrowley/go-metrics"
	"github.com/zenoss/glog"
	dockerclient "github.com/zenoss/go-dockerclient"

	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	isvcsVolumes = map[string]string{
		utils.ResourcesDir(): "/usr/local/serviced/resources",
	}
)

var (
	ErrNotRunning       = errors.New("isvc: not running")
	ErrRunning          = errors.New("isvc: running")
	ErrBadContainerSpec = errors.New("isvc: bad service specification")
)

type ExitError int

func (err ExitError) Error() string {
	return fmt.Sprintf("isvc: service received exit code %d", int(err))
}

type action int

const (
	start action = iota
	stop
	restart
)

const DEFAULT_HEALTHCHECK_NAME = "running"

// FIXME: What should the default interval and timeout be?
const DEFAULT_HEALTHCHECK_INTERVAL = time.Duration(30) * time.Second
const DEFAULT_HEALTHCHECK_TIMEOUT = time.Duration(10) * time.Second

type actionrequest struct {
	action   action
	response chan error
}

type healthCheckDefinition struct {
	healthCheck func() error  // A function to verify the service is healthy
	Interval    time.Duration // The interval at which to execute the script.
	Timeout     time.Duration // A timeout in which to complete the health check.
}

type IServiceDefinition struct {
	Name          string                             // name of the service (used in naming containers)
	Repo          string                             // the service's docker repository
	Tag           string                             // the service's docker repository tag
	Command       func() string                      // the command to run in the container
	Volumes       map[string]string                  // volumes to bind mount to the container
	Ports         []uint16                           // ports to expose to the host
	HealthChecks  map[string]healthCheckDefinition   // a set of functions to verify the service is healthy
	Configuration map[string]interface{}             // service specific configuration
	Notify        func(*IService, interface{}) error // A function to run when notified of a data event
	PostStart     func(*IService) error              // A function to run after the initial start of the service
	HostNetwork   bool                               // enables host network in the container
}

type IService struct {
	IServiceDefinition
	exited         <-chan int
	root           string
	actions        chan actionrequest
	lock           *sync.RWMutex
	healthStatuses map[string]*domain.HealthCheckStatus
}

func NewIService(sd IServiceDefinition) (*IService, error) {
	if strings.TrimSpace(sd.Name) == "" || strings.TrimSpace(sd.Repo) == "" || sd.Command == nil {
		return nil, ErrBadContainerSpec
	}

	if sd.Configuration == nil {
		sd.Configuration = make(map[string]interface{})
	}

	svc := IService{sd, nil, "", make(chan actionrequest), &sync.RWMutex{}, nil}
	if len(svc.HealthChecks) > 0 {
		svc.healthStatuses = make(map[string]*domain.HealthCheckStatus, len(svc.HealthChecks))
		for name, healthCheckDefinition := range svc.HealthChecks {
			svc.healthStatuses[name] = &domain.HealthCheckStatus{
				Name:      name,
				Status:    "unknown",
				Interval:  healthCheckDefinition.Interval.Seconds(),
				Timestamp: 0,
				StartedAt: 0,
			}
		}
	} else {
		name := DEFAULT_HEALTHCHECK_NAME
		svc.healthStatuses = make(map[string]*domain.HealthCheckStatus, 1)
		svc.healthStatuses[name] = &domain.HealthCheckStatus{
			Name:      name,
			Status:    "unknown",
			Interval:  0,
			Timestamp: 0,
			StartedAt: 0,
		}
	}

	envPerService[sd.Name] = make(map[string]string)
	go svc.run()

	return &svc, nil
}

func (svc *IService) SetRoot(root string) {
	svc.root = strings.TrimSpace(root)
}

func (svc *IService) IsRunning() bool {
	return svc.exited != nil
}

func (svc *IService) Start() error {
	response := make(chan error)
	svc.actions <- actionrequest{start, response}
	return <-response
}

func (svc *IService) Stop() error {
	response := make(chan error)
	svc.actions <- actionrequest{stop, response}
	return <-response
}

func (svc *IService) Restart() error {
	response := make(chan error)
	svc.actions <- actionrequest{restart, response}
	return <-response
}

func (svc *IService) Exec(command []string) error {
	ctr, err := docker.FindContainer(svc.name())
	if err != nil {
		return err
	}

	output, err := utils.AttachAndRun(ctr.ID, command)
	if err != nil {
		return err
	}
	os.Stdout.Write(output)
	return nil
}

func (svc *IService) getResourcePath(p string) string {
	const defaultdir string = "isvcs"

	if svc.root == "" {
		if p := strings.TrimSpace(os.Getenv("SERVICED_VARPATH")); p != "" {
			svc.root = filepath.Join(p, defaultdir)
		} else if p := strings.TrimSpace(os.Getenv("SERVICED_HOME")); p != "" {
			svc.root = filepath.Join(p, "var", defaultdir)
		} else if user, err := user.Current(); err == nil {
			svc.root = filepath.Join(os.TempDir(), fmt.Sprintf("serviced-%s", user.Username), "var", defaultdir)
		} else {
			svc.root = filepath.Join(os.TempDir(), "serviced", "var", defaultdir)
		}
	}

	return filepath.Join(svc.root, svc.Name, p)
}

func (svc *IService) name() string {
	return fmt.Sprintf("serviced-isvcs_%s", svc.Name)
}

func (svc *IService) create() (*docker.Container, error) {
	var config dockerclient.Config
	cd := &docker.ContainerDefinition{
		dockerclient.CreateContainerOptions{Name: svc.name(), Config: &config},
		dockerclient.HostConfig{},
	}

	config.Image = commons.JoinRepoTag(svc.Repo, svc.Tag)
	config.Cmd = []string{"/bin/sh", "-c", "trap 'kill 0' 15; " + svc.Command()}

	// set the host network (if enabled)
	if svc.HostNetwork {
		cd.NetworkMode = "host"
	}

	// attach all exported ports
	if svc.Ports != nil && len(svc.Ports) > 0 {
		config.ExposedPorts = make(map[dockerclient.Port]struct{})
		cd.PortBindings = make(map[dockerclient.Port][]dockerclient.PortBinding)
		for _, portno := range svc.Ports {
			port := dockerclient.Port(fmt.Sprintf("%d", portno))
			config.ExposedPorts[port] = struct{}{}
			cd.PortBindings[port] = append(cd.PortBindings[port], dockerclient.PortBinding{HostPort: fmt.Sprintf("%d", portno)})
		}
	}

	// attach all exported volumes
	config.Volumes = make(map[string]struct{})
	cd.Binds = []string{}

	// service-specific volumes
	if svc.Volumes != nil && len(svc.Volumes) > 0 {
		for src, dest := range svc.Volumes {
			hostpath := svc.getResourcePath(src)
			if exists, _ := isDir(hostpath); !exists {
				if err := os.MkdirAll(hostpath, 0777); err != nil {
					glog.Errorf("could not create %s on host: %s", hostpath, err)
					return nil, err
				}
			}
			cd.Binds = append(cd.Binds, fmt.Sprintf("%s:%s", hostpath, dest))
			config.Volumes[dest] = struct{}{}
		}
	}

	// global volumes
	if isvcsVolumes != nil && len(isvcsVolumes) > 0 {
		for src, dest := range isvcsVolumes {
			if exists, _ := isDir(src); !exists {
				glog.Warningf("Could not mount source %s: path does not exist", src)
				continue
			}
			cd.Binds = append(cd.Binds, fmt.Sprintf("%s:%s", src, dest))
			config.Volumes[dest] = struct{}{}
		}
	}

	// attach environment variables
	for key, val := range envPerService[svc.Name] {
		config.Env = append(config.Env, fmt.Sprintf("%s=%s", key, val))
	}

	return docker.NewContainer(cd, false, 5*time.Second, nil, nil)
}

func (svc *IService) attach() (*docker.Container, error) {
	ctr, _ := docker.FindContainer(svc.name())
	if ctr != nil {
		notify := make(chan int, 1)
		if !ctr.IsRunning() {
			glog.Infof("isvc %s found but not running; removing container %s", svc.name(), ctr.ID)
			go svc.remove(notify)
		} else if !svc.checkvolumes(ctr) {
			glog.Infof("isvc %s found but volumes are missing or incomplete; removing container %s", svc.name(), ctr.ID)
			ctr.OnEvent(docker.Die, func(cid string) { svc.remove(notify) })
			svc.stop()
		} else {
			glog.Infof("Attaching to isvc %s at %s", svc.name(), ctr.ID)
			return ctr, nil
		}
		<-notify
	}

	glog.Infof("Creating a new container for isvc %s", svc.name())
	return svc.create()
}

func (svc *IService) start() (<-chan int, error) {
	ctr, err := svc.attach()
	if err != nil {
		svc.setStoppedHealthStatus(fmt.Errorf("could not start service: %s", err))
		return nil, err
	}

	// destroy the container when it dies
	notify := make(chan int, 1)
	ctr.OnEvent(docker.Die, func(cid string) { svc.remove(notify) })

	// start the container
	if err := ctr.Start(10 * time.Second); err != nil && err != docker.ErrAlreadyStarted {
		svc.setStoppedHealthStatus(fmt.Errorf("could not start service: %s", err))
		return nil, err
	}

	// perform an initial healthcheck to verify that the service started succeessfully
	select {
	case err := <-svc.healthcheck(0):
		if err != nil {
			svc.stop()
			return nil, err
		}
	case rc := <-notify:
		svc.setStoppedHealthStatus(ExitError(rc))
		return nil, ExitError(rc)
	}

	return notify, nil
}

func (svc *IService) stop() error {
	ctr, err := docker.FindContainer(svc.name())
	if err == docker.ErrNoSuchContainer {
		svc.setStoppedHealthStatus(nil)
		return nil
	} else if err != nil {
		glog.Errorf("Could not get isvc container %s", svc.Name)
		svc.setStoppedHealthStatus(err)
		return err
	}

	glog.Warningf("Stopping isvc container %s", svc.name())
	err = ctr.Stop(45 * time.Second)
	svc.setStoppedHealthStatus(err)
	return err
}

func (svc *IService) remove(notify chan<- int) {
	defer close(notify)
	ctr, err := docker.FindContainer(svc.name())
	if err == docker.ErrNoSuchContainer {
		return
	} else if err != nil {
		glog.Errorf("Could not get isvc container %s", svc.Name)
		return
	}

	// report the log output
	if output, err := exec.Command("docker", "logs", "--tail", "1000", ctr.ID).CombinedOutput(); err != nil {
		glog.Warningf("Could not get logs for container %s", ctr.Name)
	} else {
		glog.V(1).Infof("Exited isvc %s:\n %s", svc.Name, string(output))
	}

	// kill the container if it is running
	if ctr.IsRunning() {
		glog.Warningf("isvc %s is still running; killing", svc.Name)
		ctr.Kill()
	}

	// get the exit code
	rc, _ := ctr.Wait(time.Second)
	defer func() { notify <- rc }()

	// delete the container
	if err := ctr.Delete(true); err != nil {
		glog.Errorf("Could not remove isvc %s: %s", ctr.Name, err)
	}
}

func (svc *IService) run() {
	var err error
	var collecting bool
	haltStats := make(chan struct{})
	haltHealthChecks := make(chan struct{})

	for {
		select {
		case req := <-svc.actions:
			switch req.action {
			case stop:
				glog.Infof("Stopping isvc %s", svc.Name)
				if svc.exited == nil {
					req.response <- ErrNotRunning
					continue
				}

				if collecting {
					haltStats <- struct{}{}
					if len(svc.HealthChecks) > 0 {
						haltHealthChecks <- struct{}{}
					}
					collecting = false
				}

				if err := svc.stop(); err != nil {
					req.response <- err
					continue
				}

				if rc := <-svc.exited; rc != 0 {
					glog.Warningf("isvc %s received exit code %d", svc.Name, rc)
				}
				svc.exited = nil
				req.response <- nil
			case start:
				glog.Infof("Starting isvc %s", svc.Name)
				if svc.exited != nil {
					req.response <- ErrRunning
					continue
				}

				if svc.exited, err = svc.start(); err != nil {
					req.response <- err
					continue
				}

				if !collecting {
					go svc.stats(haltStats)
					go svc.doHealthChecks(haltHealthChecks)
					collecting = true
				}

				req.response <- nil
			case restart:
				glog.Infof("Restarting isvc %s", svc.Name)
				if svc.exited != nil {

					if collecting {
						haltStats <- struct{}{}
						if len(svc.HealthChecks) > 0 {
							haltHealthChecks <- struct{}{}
						}
						collecting = false
					}

					if err := svc.stop(); err != nil {
						req.response <- err
						continue
					}
					<-svc.exited
				}
				if svc.exited, err = svc.start(); err != nil {
					req.response <- err
					continue
				}
				req.response <- nil
			}
		case rc := <-svc.exited:
			glog.Errorf("isvc %s unexpectedly received exit code %d", svc.Name, rc)
			if svc.exited, err = svc.start(); err != nil {
				glog.Errorf("Error restarting isvc %s: %s", svc.Name, err)
			}
		}
	}
}

func (svc *IService) checkvolumes(ctr *docker.Container) bool {
	dctr, err := ctr.Inspect()
	if err != nil {
		return false
	}

	if svc.Volumes != nil {
		for src, dest := range svc.Volumes {
			if p, ok := dctr.Volumes[dest]; ok {
				src, _ = filepath.EvalSymlinks(svc.getResourcePath(src))
				if rel, _ := filepath.Rel(filepath.Clean(src), p); rel != "." {
					return false
				}
			} else {
				return false
			}
		}
	}

	if isvcsVolumes != nil {
		for src, dest := range isvcsVolumes {
			if p, ok := dctr.Volumes[dest]; ok {
				if rel, _ := filepath.Rel(src, p); rel != "." {
					return false
				}
			} else {
				return false
			}
		}
	}

	return true
}

func (svc *IService) healthcheck(currentTime int64) <-chan error {
	err := make(chan error, 1)
	go func() {
		if len(svc.HealthChecks) > 0 {
			var result error
			if checkDefinition, found := svc.HealthChecks[DEFAULT_HEALTHCHECK_NAME]; found {
				result := svc.runCheckOrTimeout(checkDefinition)
				svc.setHealthStatus(result, currentTime)
			} else {
				glog.Warningf("Default healthcheck %q not found for isvc %s", DEFAULT_HEALTHCHECK_NAME, svc.Name)
			}
			err <- result
		} else {
			svc.setHealthStatus(nil, currentTime)
			err <- nil
		}
	}()
	return err
}

func (svc *IService) runCheckOrTimeout(checkDefinition healthCheckDefinition) error {
	finished := make(chan error, 1)
	go func() {
		finished <- checkDefinition.healthCheck()
	}()

	var result error
	select {
	case err := <-finished:
		result = err
	case <-time.After(checkDefinition.Timeout):
		glog.Errorf("healthcheck timed out for %s", svc.name())
		result = fmt.Errorf("healthcheck timed out")
	}
	return result
}

func (svc *IService) doHealthChecks(halt <-chan struct{}) {

	if len(svc.HealthChecks) == 0 {
		return
	}

	var found bool
	var healthCheck healthCheckDefinition
	if healthCheck, found = svc.HealthChecks[DEFAULT_HEALTHCHECK_NAME]; !found {
		glog.Warningf("Default healthcheck %q not found for isvc %s", DEFAULT_HEALTHCHECK_NAME, svc.name())
		return
	}

	timer := time.Tick(healthCheck.Interval)
	healthCheckRunning := false
	for {
		select {
		case <-halt:
			glog.Infof("Stopped healthchecks for %s", svc.name())
			return

		case currentTime := <-timer:
			if healthCheckRunning {
				glog.Warning("Healthcheck %q for isvc %s still running", DEFAULT_HEALTHCHECK_NAME, svc.name())
				return
			}

			healthCheckRunning = true
			err := <-svc.healthcheck(currentTime.Unix())
			healthCheckRunning = false
			if err != nil {
				glog.Errorf("Healthcheck for isvc %s failed: %s", svc.name(), err)
				svc.stop()
				return
			}
		}
	}
}

func (svc *IService) setHealthStatus(result error, currentTime int64) {
	if len(svc.healthStatuses) == 0 {
		return
	}

	svc.lock.Lock()
	defer svc.lock.Unlock()

	if healthStatus, found := svc.healthStatuses[DEFAULT_HEALTHCHECK_NAME]; found {
		if result == nil {
			healthStatus.Status = "passed"
			healthStatus.Failure = ""
		} else {
			healthStatus.Status = "failed"
			healthStatus.Failure = result.Error()
		}
		healthStatus.Timestamp = currentTime
		if healthStatus.StartedAt == 0 {
			healthStatus.StartedAt = currentTime
		}
	} else {
		glog.Errorf("isvc %s does have the default health check %s", svc.Name, DEFAULT_HEALTHCHECK_NAME)
	}
}

func (svc *IService) setStoppedHealthStatus(stopResult error) {
	if len(svc.healthStatuses) == 0 {
		return
	}

	svc.lock.Lock()
	defer svc.lock.Unlock()

	if healthStatus, found := svc.healthStatuses[DEFAULT_HEALTHCHECK_NAME]; found {
		if stopResult == nil {
			healthStatus.Status = "stopped"
			healthStatus.Failure = ""
		} else {
			healthStatus.Status = "failed"
			healthStatus.Failure = stopResult.Error()
		}
		healthStatus.Timestamp = time.Now().Unix()
		healthStatus.StartedAt = 0
	} else {
		glog.Errorf("isvc %s does have the default health check %s", svc.Name, DEFAULT_HEALTHCHECK_NAME)
	}
}

type containerStat struct {
	Metric    string            `json:"metric"`
	Value     string            `json:"value"`
	Timestamp int64             `json:"timestamp"`
	Tags      map[string]string `json:"tags"`
}

func (svc *IService) stats(halt <-chan struct{}) {
	registry := metrics.NewRegistry()
	tc := time.Tick(10 * time.Second)

	for {
		select {
		case <-halt:
			glog.Infof("stop collecting stats for %s", svc.name())
			return
		case t := <-tc:
			ctr, err := docker.FindContainer(svc.name())
			if err != nil {
				glog.Warningf("Could not find container for isvc %s: %s", svc.Name, err)
				break
			}

			if cpuacctStat, err := cgroup.ReadCpuacctStat(cgroup.GetCgroupDockerStatsFilePath(ctr.ID, cgroup.Cpuacct)); err != nil {
				glog.Warningf("Could not read CpuacctStat for isvc %s: %s", svc.Name, err)
				break
			} else {
				metrics.GetOrRegisterGauge("CpuacctStat.system", registry).Update(cpuacctStat.System)
				metrics.GetOrRegisterGauge("CpuacctStat.user", registry).Update(cpuacctStat.User)
			}

			if memoryStat, err := cgroup.ReadMemoryStat(cgroup.GetCgroupDockerStatsFilePath(ctr.ID, cgroup.Memory)); err != nil {
				glog.Warningf("Could not read MemoryStat for isvc %s: %s", svc.Name, err)
				break
			} else {
				metrics.GetOrRegisterGauge("cgroup.memory.pgmajfault", registry).Update(memoryStat.Pgfault)
				metrics.GetOrRegisterGauge("cgroup.memory.totalrss", registry).Update(memoryStat.TotalRss)
				metrics.GetOrRegisterGauge("cgroup.memory.cache", registry).Update(memoryStat.Cache)
			}
			// Gather the stats
			stats := []containerStat{}
			registry.Each(func(name string, i interface{}) {
				if metric, ok := i.(metrics.Gauge); ok {
					tagmap := make(map[string]string)
					tagmap["isvcname"] = svc.Name
					stats = append(stats, containerStat{name, strconv.FormatInt(metric.Value(), 10), t.Unix(), tagmap})
				}
				if metricf64, ok := i.(metrics.GaugeFloat64); ok {
					tagmap := make(map[string]string)
					tagmap["isvcname"] = svc.Name
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
