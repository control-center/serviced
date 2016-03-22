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

package isvcs

import (
	"github.com/control-center/serviced/commons"
	"github.com/control-center/serviced/commons/docker"
	dfsdocker "github.com/control-center/serviced/dfs/docker"
	"github.com/control-center/serviced/domain"
	"github.com/control-center/serviced/utils"
	dockerclient "github.com/fsouza/go-dockerclient"
	"github.com/rcrowley/go-metrics"
	"github.com/zenoss/glog"

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

var isvcsVolumes map[string]string

func loadvolumes() {
	if isvcsVolumes == nil {
		isvcsVolumes = map[string]string{
			utils.ResourcesDir(): "/usr/local/serviced/resources",
		}
	}
}

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

const DEFAULT_HEALTHCHECK_INTERVAL = time.Duration(30) * time.Second
const DEFAULT_HEALTHCHECK_TIMEOUT = time.Duration(10) * time.Second

const WAIT_FOR_INITIAL_HEALTHCHECK = time.Duration(2) * time.Minute

type actionrequest struct {
	action   action
	response chan error
}

// HealthCheckFunction- A function to verify the service is healthy
type HealthCheckFunction func(halt <-chan struct{}) error

type healthCheckDefinition struct {
	healthCheck HealthCheckFunction
	Interval    time.Duration // The interval at which to execute the script.
	Timeout     time.Duration // A timeout in which to complete the health check.
}

// portBinding defines how a port in the container is bound to port on the master host.
// The port number in the container and the port exposed on the master host are always
// same, but the IP on the master host can vary.
//
// HostIp - use blank or "0.0.0.0" to bind to a port that will be public on the master host IP;
//          use "127.0.0.1" to bind to a port that will be restricted to "localhost" on the master host
// HostIpOverride - if not blank, assumed to be an environment variable containing a value that
//          overrides HostIp. The naming convention for the env var should be
//               SERVICED_ISVC_<NAME>_PORT_<NUMBER>_HOSTIP
//          where <NAME> is the name if the isvc (e.g. "opentsdb")
//                <PORT> is the port number to be overriden (e.g. 12181)
// HostPort - the port number on the master host to bind to.
type portBinding struct {
	HostIp         string
	HostIpOverride string
	HostPort       uint16
}

type IServiceDefinition struct {
	ID             string                             // the ID of the service associated
	Name           string                             // name of the service (used in naming containers)
	Repo           string                             // the service's docker repository
	Tag            string                             // the service's docker repository tag
	Command        func() string                      // the command to run in the container
	Volumes        map[string]string                  // volumes to bind mount to the container
	PortBindings   []portBinding                      // defines how ports are exposed on the host
	HealthChecks   map[string]healthCheckDefinition   // a set of functions to verify the service is healthy
	Configuration  map[string]interface{}             // service specific configuration
	Notify         func(*IService, interface{}) error // A function to run when notified of a data event
	PreStart       func(*IService) error              // A function to run before the initial start of the service
	PostStart      func(*IService) error              // A function to run after the initial start of the service
	Recover        func(path string) error            // A recovery step if the service fails to start
	HostNetwork    bool                               // enables host network in the container
	Links          []string                           // List of links to other containers in the form of <name>:<alias>
	StartGroup     uint16                             // Start up group number
	StartupTimeout time.Duration                      // How long to wait for the service to start up (this is the timeout for the initial 'startup' healthcheck)
}

type IService struct {
	IServiceDefinition
	root            string
	actions         chan actionrequest
	startTime       time.Time
	restartCount    int
	dockerLogDriver string            // which log driver to use with containers
	dockerLogConfig map[string]string // options for the log driver
	docker          dfsdocker.Docker  // Docker API, needed to get stats

	channelLock *sync.RWMutex
	exited      <-chan int

	lock           *sync.RWMutex
	healthStatuses map[string]*domain.HealthCheckStatus
}

func NewIService(sd IServiceDefinition) (*IService, error) {
	if strings.TrimSpace(sd.Name) == "" || strings.TrimSpace(sd.Repo) == "" || strings.TrimSpace(sd.ID) == "" || sd.Command == nil {
		return nil, ErrBadContainerSpec
	}

	if sd.Configuration == nil {
		sd.Configuration = make(map[string]interface{})
	}

	if sd.StartupTimeout == 0 { //Initialize startup timeout to Default for all IServices if not specified
		sd.StartupTimeout = WAIT_FOR_INITIAL_HEALTHCHECK
	}
	svc := IService{
		IServiceDefinition: sd,
		root:               "",
		actions:            make(chan actionrequest),
		startTime:          time.Time{},
		restartCount:       0,
		dockerLogDriver:    "",
		dockerLogConfig:    nil,
		channelLock:        &sync.RWMutex{},
		exited:             nil,
		lock:               &sync.RWMutex{},
		healthStatuses:     nil,
	}

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
			Interval:  3.156e9,
			Timestamp: 0,
			StartedAt: 0,
		}
	}

	envPerService[sd.Name] = make(map[string]string)
	envPerService[sd.Name]["CONTROLPLANE_SERVICE_ID"] = svc.ID

	if sd.PreStart != nil {
		if err := sd.PreStart(&svc); err != nil {
			glog.Errorf("Could not prestart service %s: %s", sd.Name, err)
			return nil, err
		}
	}
	go svc.run()

	return &svc, nil
}

func (svc *IService) SetRoot(root string) {
	svc.root = strings.TrimSpace(root)
}

func (svc *IService) IsRunning() bool {
	svc.channelLock.RLock()
	defer svc.channelLock.RUnlock()

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

func (svc *IService) Exec(command []string) ([]byte, error) {
	ctr, err := docker.FindContainer(svc.name())
	if err != nil {
		return nil, err
	}

	output, err := utils.AttachAndRun(ctr.ID, command)
	if err != nil {
		return output, err
	}
	os.Stdout.Write(output)
	return output, nil
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

func (svc *IService) setExitedChannel(newChan <-chan int) {
	svc.channelLock.Lock()
	defer svc.channelLock.Unlock()

	svc.exited = newChan
}

func (svc *IService) name() string {
	return fmt.Sprintf("serviced-isvcs_%s", svc.Name)
}

func (svc *IService) create() (*docker.Container, error) {
	hostConfig := dockerclient.HostConfig{}
	hostConfig.LogConfig.Type = svc.dockerLogDriver
	hostConfig.LogConfig.Config = svc.dockerLogConfig
	// CC-1848: set core limit to 0
	hostConfig.Ulimits = []dockerclient.ULimit{
		{
			Name: "core",
			Soft: 0,
			Hard: 0,
		},
	}

	glog.Infof("hostConfig.LogConfig.Type=%s", hostConfig.LogConfig.Type)
	glog.Infof("hostConfig.LogConfig.Config=%v", hostConfig.LogConfig.Config)

	var config dockerclient.Config
	cd := &docker.ContainerDefinition{
		dockerclient.CreateContainerOptions{Name: svc.name(), Config: &config},
		hostConfig,
	}

	config.Image = commons.JoinRepoTag(svc.Repo, svc.Tag)
	config.Cmd = []string{"/bin/sh", "-c", "trap 'kill 0' 15; " + svc.Command()}

	// NOTE: USE WITH CARE!
	// Enabling host networking for an isvc may expose ports
	// of the isvcs to access outside of the serviced host, potentially
	// compromising security.
	if svc.HostNetwork {
		cd.NetworkMode = "host"
		glog.Warningf("Host networking enabled for isvc %s", svc.Name)
	}

	// attach all exported ports
	if svc.PortBindings != nil && len(svc.PortBindings) > 0 {
		config.ExposedPorts = make(map[dockerclient.Port]struct{})
		cd.PortBindings = make(map[dockerclient.Port][]dockerclient.PortBinding)
		for _, binding := range svc.PortBindings {
			port := dockerclient.Port(fmt.Sprintf("%d", binding.HostPort))
			config.ExposedPorts[port] = struct{}{}
			portBinding := dockerclient.PortBinding{
				HostIP:   getHostIp(binding),
				HostPort: port.Port(),
			}

			cd.PortBindings[port] = append(cd.PortBindings[port], portBinding)
		}
	}
	glog.V(1).Infof("Bindings for %s = %v", svc.Name, cd.PortBindings)

	// copy any links to other isvcs
	if svc.Links != nil && len(svc.Links) > 0 {
		// To use a link, the source container must be instantiated already, so
		//    the service using a link can't be in the first start group.
		//
		// FIXME: Other sanity checks we could add - make sure that the source
		//        container is not in the same group or a later group
		if svc.StartGroup == 0 {
			glog.Fatalf("isvc %s can not use docker Links with StartGroup=0", svc.Name)
		}
		cd.Links = make([]string, len(svc.Links))
		copy(cd.Links, svc.Links)
		glog.V(1).Infof("Links for %s = %v", svc.Name, cd.Links)
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
		} else if !svc.checkVolumes(ctr) {
			// CC-1550: A reload causes CC to re-read its configuration, which means that the host volumes
			//          mounted into the isvcs containers might change. If that happens, we cannot simply
			//          attach to the existing containers. Instead, we need to stop them and create new ones
			//          using the revised configuration.
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
	if err := ctr.Start(); err != nil && err != docker.ErrAlreadyStarted {
		svc.setStoppedHealthStatus(fmt.Errorf("could not start service: %s", err))
		return nil, err
	}

	// perform an initial healthcheck to verify that the service started successfully
	select {
	case err := <-svc.startupHealthcheck():
		if err != nil {
			glog.Errorf("Healthcheck for %s failed: %s", svc.Name, err)
			svc.stop()
			return nil, err
		}
		svc.startTime = time.Now()
	case rc := <-notify:
		glog.Errorf("isvc %s exited on startup, rc=%d", svc.Name, rc)
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
	if err := ctr.Delete(true); err != nil && err != docker.ErrNoSuchContainer {
		glog.Errorf("Could not remove isvc %s: %s", ctr.Name, err)
	}
}

func (svc *IService) run() {
	var newExited <-chan int
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
				if !svc.IsRunning() {
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
					svc.setStoppedHealthStatus(ExitError(rc))
					glog.Errorf("isvc %s received exit code %d", svc.Name, rc)
				}
				svc.setExitedChannel(nil)
				req.response <- nil
			case start:
				glog.Infof("Starting isvc %s", svc.Name)
				if svc.IsRunning() {
					req.response <- ErrRunning
					continue
				}

				newExited, err = svc.start()
				if err != nil && svc.Recover != nil {
					glog.Warningf("ISVC %s failed to start; attempting recovery", svc.name())
					if e := svc.Recover(svc.getResourcePath("")); e != nil {
						glog.Errorf("Could not recover service %s: %s", svc.name(), e)
					} else {
						newExited, err = svc.start()
					}
				}
				svc.setExitedChannel(newExited)
				if err != nil {
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
				if svc.IsRunning() {

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
				newExited, err = svc.start()
				svc.setExitedChannel(newExited)
				if err != nil {
					req.response <- err
					continue
				}

				if !collecting {
					go svc.stats(haltStats)
					go svc.doHealthChecks(haltHealthChecks)
					collecting = true
				}
				req.response <- nil
			}
		case rc := <-svc.exited:
			svc.setStoppedHealthStatus(ExitError(rc))
			glog.Errorf("isvc %s exited unexpectedly; rc=%d", svc.Name, rc)

			stopService := svc.isFlapping()
			if !stopService {
				glog.Infof("Restarting isvc %s ", svc.Name)
				newExited, err = svc.start()
				svc.setExitedChannel(newExited)
				if err != nil {
					glog.Errorf("Error restarting isvc %s: %s", svc.Name, err)
					stopService = true
				}
			}

			if stopService {
				if collecting {
					haltStats <- struct{}{}
					if len(svc.HealthChecks) > 0 {
						haltHealthChecks <- struct{}{}
					}
					collecting = false
				}
				glog.Errorf("isvc %s not restarted; failed too many times", svc.Name)
				svc.stop()
				svc.setExitedChannel(nil)
			}
		}
	}
}

// isFlapping returns true if the service is flapping (starting/stopping repeatedly).
//
// A service is considered flappping, if has been running for less than 60 seconds
// in 3 consecutive calls to this function.
func (svc *IService) isFlapping() bool {
	const MINIMUM_VIABLE_LIFESPAN = 60
	const FLAPPING_THRESHOLD = 3

	upTime := time.Since(svc.startTime)
	if upTime.Seconds() < MINIMUM_VIABLE_LIFESPAN {
		svc.restartCount += 1
	} else {
		svc.restartCount = 0
	}

	return svc.restartCount >= FLAPPING_THRESHOLD
}

func (svc *IService) checkVolumes(ctr *docker.Container) bool {
	dctr, err := ctr.Inspect()
	if err != nil {
		glog.Errorf("Unable to inspect container %s: %s", ctr.ID, err)
		return false
	}

	glog.V(2).Infof("checkVolumes for isvcs %s containerID=%s:\ndctr=%#v", svc.Name, ctr.ID, dctr)

	if svc.Volumes != nil {
		for src, dest := range svc.Volumes {
			var mount *dockerclient.Mount
			if mount = findContainerMount(dctr, dest); mount == nil {
				glog.V(2).Infof("checkVolumes for isvcs %s, volume %s not found in containerID=%s", svc.Name, dest, ctr.ID)
				return false
			}

			expectedSrc, _ := filepath.EvalSymlinks(svc.getResourcePath(src))
			if rel, _ := filepath.Rel(filepath.Clean(expectedSrc), mount.Source); rel != "." {
				glog.V(2).Infof("checkVolumes for isvcs %s, the mount for volume %s has changed in containerID=%s; expected %s, found %s",
					svc.Name, dest, ctr.ID, expectedSrc, mount.Source)
				return false
			}
		}
	}

	if isvcsVolumes != nil {
		for src, dest := range isvcsVolumes {
			var mount *dockerclient.Mount
			if mount = findContainerMount(dctr, dest); mount == nil {
				glog.V(2).Infof("checkVolumes for isvcs %s, global volume %s not found in containerID=%s", svc.Name, dest, ctr.ID)
				return false
			}

			if rel, _ := filepath.Rel(src, mount.Source); rel != "." {
				glog.V(2).Infof("checkVolumes for isvcs %s, the mount for global volume %s has changed in containerID=%s; expected %s, found %s",
					svc.Name, dest, ctr.ID, src, mount.Source)
				return false
			}
		}
	}

	return true
}

// startupHealthcheck runs the default healthchecks (if any) and the return the result.
// If the healthcheck fails, then this method will sleep 1 second, and then
// repeat the healthcheck, continuing that sleep/retry pattern until
// the healthcheck succeeds or 2 minutes has elapsed.
//
// An error is returned if the no healtchecks succeed in the 2 minute interval,
// otherwise nil is returned
func (svc *IService) startupHealthcheck() <-chan error {
	err := make(chan error, 1)
	go func() {
		var result error
		if len(svc.HealthChecks) > 0 {
			checkDefinition, found := svc.HealthChecks[DEFAULT_HEALTHCHECK_NAME]
			if !found {
				glog.Warningf("Default healthcheck %q not found for isvc %s", DEFAULT_HEALTHCHECK_NAME, svc.Name)
				err <- nil
				return
			}

			startCheck := time.Now()
			for {
				currentTime := time.Now()
				result = svc.runCheckOrTimeout(checkDefinition)
				svc.setHealthStatus(result, currentTime.Unix())
				elapsed := time.Since(startCheck)
				if result == nil {
					glog.Infof("Verified health status of %s after %s", svc.Name, elapsed)
					break
				} else if elapsed.Seconds() > svc.StartupTimeout.Seconds() {
					glog.Errorf("Could not verify health status of %s after %s. Last health check returned %#v",
						svc.Name, svc.StartupTimeout, result)
					break
				}

				glog.Infof("waiting for %s to start, checking health status again in 1 second", svc.Name)
				time.Sleep(time.Second)
			}
			err <- result
		} else {
			svc.setHealthStatus(nil, time.Now().Unix())
			err <- nil
		}
	}()
	return err
}

func (svc *IService) runCheckOrTimeout(checkDefinition healthCheckDefinition) error {
	finished := make(chan error, 1)
	halt := make(chan struct{}, 1)

	go func() {
		finished <- checkDefinition.healthCheck(halt)
	}()

	var result error
	select {
	case err := <-finished:
		result = err
	case <-time.After(checkDefinition.Timeout):
		glog.Errorf("healthcheck timed out for %s", svc.Name)
		result = fmt.Errorf("healthcheck timed out")
		halt <- struct{}{}
	}
	return result
}

func (svc *IService) doHealthChecks(halt <-chan struct{}) {

	if len(svc.HealthChecks) == 0 {
		return
	}

	var found bool
	var checkDefinition healthCheckDefinition
	if checkDefinition, found = svc.HealthChecks[DEFAULT_HEALTHCHECK_NAME]; !found {
		glog.Warningf("Default healthcheck %q not found for isvc %s", DEFAULT_HEALTHCHECK_NAME, svc.Name)
		return
	}

	timer := time.Tick(checkDefinition.Interval)
	for {
		select {
		case <-halt:
			glog.Infof("Stopped healthchecks for %s", svc.Name)
			return

		case currentTime := <-timer:
			err := svc.runCheckOrTimeout(checkDefinition)
			svc.setHealthStatus(err, currentTime.Unix())
			if err != nil {
				glog.Errorf("Healthcheck for isvc %s failed: %s", svc.Name, err)
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
			if healthStatus.Status != "passed" && healthStatus.Status != "unknown" {
				glog.Infof("Health status for %s returned to 'passed'", svc.Name)
			}
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
		healthStatus.Status = "stopped"
		if stopResult == nil {
			healthStatus.Failure = ""
		} else {
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

	// previous stats holds some of the stats gathered in the previous sample, currently used for computing CPU %
	previousStats := make(map[string]uint64)
	//The docker container ID the last time we gathered stats, used to detect when the container has changed to avoid invalid CPU stats
	previousDockerID := ""

	for {
		select {
		case <-halt:
			glog.Infof("stop collecting stats for %s", svc.Name)
			return
		case t := <-tc:
			ctr, err := docker.FindContainer(svc.name())
			if err != nil {
				glog.Warningf("Could not find container for isvc %s: %s", svc.Name, err)
				break
			}

			if svc.docker == nil {
				glog.Warningf("Docker API object has not been set yet for Isvcs %s", svc.Name)
				break
			}

			dockerstats, err := svc.docker.GetContainerStats(ctr.ID, 30*time.Second)
			if err != nil || dockerstats == nil { //dockerstats may be nil if service is shutting down
				glog.Warningf("Couldn't get stats for IService %s: %v", svc.Name, err)
				break
			}

			// We store and check the docker ID because we can't use the previous value to compute percentage if the container has changed
			usePreviousStats := true
			if ctr.ID != previousDockerID {
				usePreviousStats = false //docker ID has changed, this service was restarted
			}
			previousDockerID = ctr.ID

			// CPU Stats
			// TODO: Consolidate this into a single object that both ISVCS and non-ISVCS can use
			var (
				kernelCPUPercent float64
				userCPUPercent   float64
				totalCPUChange   uint64
			)

			kernelCPU := dockerstats.CPUStats.CPUUsage.UsageInKernelmode
			userCPU := dockerstats.CPUStats.CPUUsage.UsageInUsermode
			totalCPU := dockerstats.CPUStats.SystemCPUUsage

			// Total CPU Cycles
			previousTotalCPU, found := previousStats["totalCPU"]
			if found {
				if totalCPU <= previousTotalCPU {
					glog.Warningf("Change in total CPU usage was nonpositive, skipping CPU stats update.")
					usePreviousStats = false
				} else {
					totalCPUChange = totalCPU - previousTotalCPU
				}
			} else {
				usePreviousStats = false
			}
			previousStats["totalCPU"] = totalCPU

			// CPU Cycles in Kernel mode
			if previousKernelCPU, found := previousStats["kernelCPU"]; found && usePreviousStats {
				kernelCPUChange := kernelCPU - previousKernelCPU
				kernelCPUPercent = (float64(kernelCPUChange) / float64(totalCPUChange)) * float64(len(dockerstats.CPUStats.CPUUsage.PercpuUsage)) * 100.0
			} else {
				usePreviousStats = false
			}
			previousStats["kernelCPU"] = kernelCPU

			// CPU Cycles in User mode
			if previousUserCPU, found := previousStats["userCPU"]; found && usePreviousStats {
				userCPUChange := userCPU - previousUserCPU
				userCPUPercent = (float64(userCPUChange) / float64(totalCPUChange)) * float64(len(dockerstats.CPUStats.CPUUsage.PercpuUsage)) * 100.0
			} else {
				usePreviousStats = false
			}
			previousStats["userCPU"] = userCPU

			// Update CPU metrics
			if usePreviousStats {
				metrics.GetOrRegisterGaugeFloat64("docker.usageinkernelmode", registry).Update(kernelCPUPercent)
				metrics.GetOrRegisterGaugeFloat64("docker.usageinusermode", registry).Update(userCPUPercent)
			} else {
				glog.V(4).Infof("Skipping CPU stats for IService %s, no previous values to compare to", svc.Name)
			}

			// Memory Stats
			pgFault := int64(dockerstats.MemoryStats.Stats.Pgfault)
			totalRSS := int64(dockerstats.MemoryStats.Stats.TotalRss)
			cache := int64(dockerstats.MemoryStats.Stats.Cache)
			if pgFault < 0 || totalRSS < 0 || cache < 0 {
				glog.Warningf("Memory metric value for IService %s too big for int64", svc.Name)
			}
			metrics.GetOrRegisterGauge("cgroup.memory.pgmajfault", registry).Update(pgFault)
			metrics.GetOrRegisterGauge("cgroup.memory.totalrss", registry).Update(totalRSS)
			metrics.GetOrRegisterGauge("cgroup.memory.cache", registry).Update(cache)

			// Gather the stats
			stats := []containerStat{}
			registry.Each(func(name string, i interface{}) {
				tagmap := make(map[string]string)
				tagmap["isvc"] = "true"
				tagmap["isvcname"] = svc.Name
				tagmap["controlplane_service_id"] = svc.ID
				if metric, ok := i.(metrics.Gauge); ok {
					stats = append(stats, containerStat{name, strconv.FormatInt(metric.Value(), 10), t.Unix(), tagmap})
				} else if metricf64, ok := i.(metrics.GaugeFloat64); ok {
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
				// FIXME: This should be a warning, but it happens alot at startup, so let's keep it
				// on the DL until we have some better startup coordination and/or logging mechanisms.
				glog.V(1).Infof("Couldn't post isvc container stats: %s", err)
				break
			}
			defer resp.Body.Close()
			if strings.Contains(resp.Status, "204 No Content") == false {
				glog.Warningf("Post for isvcs container stats failed: %s", resp.Status)
				break
			}
		}
	}
}

func getHostIp(binding portBinding) string {
	hostIp := binding.HostIp
	if binding.HostIpOverride != "" {
		if override := strings.TrimSpace(os.Getenv(binding.HostIpOverride)); override != "" {
			hostIp = override
			glog.Infof("Using HostIp override %s = %v", binding.HostIpOverride, hostIp)
		}
	}
	return hostIp
}

func findContainerMount(dctr *dockerclient.Container, dest string) *dockerclient.Mount {
	for _, mount := range dctr.Mounts {
		if mount.Destination == dest {
			return &mount
		}
	}
	return nil
}
