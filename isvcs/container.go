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
	"github.com/Sirupsen/logrus"
	"github.com/control-center/serviced/commons"
	"github.com/control-center/serviced/commons/docker"
	"github.com/control-center/serviced/config"
	dfsdocker "github.com/control-center/serviced/dfs/docker"
	"github.com/control-center/serviced/domain"
	"github.com/control-center/serviced/utils"
	dockerclient "github.com/fsouza/go-dockerclient"
	metrics "github.com/rcrowley/go-metrics"

	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

var isvcsVolumes map[string]string
var UseServicedLogDir = "UseServicedLogDir"

func loadvolumes() {
	if isvcsVolumes == nil {
		isvcsVolumes = map[string]string{
			utils.ResourcesDir(): utils.RESOURCES_CONTAINER_DIRECTORY,
		}
	}
}

var (
	ErrNotRunning       = errors.New("isvc: not running")
	ErrRunning          = errors.New("isvc: running")
	ErrBadContainerSpec = errors.New("isvc: bad service specification")
	ErrNoDockerIP       = errors.New("isvc: unable to get IP for docker interface")

	loggedHostIPOverride bool
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

// CustomStatFunction is a function to run at an interval to get custom stats for a service
type CustomStatsFunction func(halt <-chan struct{}) error

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
	HealthChecks   []map[string]healthCheckDefinition // a set of functions to verify the service is healthy
	Configuration  map[string]interface{}             // service specific configuration
	Notify         func(*IService, interface{}) error // A function to run when notified of a data event
	PreStart       func(*IService) error              // A function to run before the initial start of the service
	PostStart      func(*IService) error              // A function to run after the initial start of the service
	Recover        func(path string) error            // A recovery step if the service fails to start
	StartupFailed  func()                             // A clean up step just before the service is stopped
	HostNetwork    bool                               // enables host network in the container
	Links          []string                           // List of links to other containers in the form of <name>:<alias>
	StartGroup     uint16                             // Start up group number
	StartupTimeout time.Duration                      // How long to wait for the service to start up (this is the timeout for the initial 'startup' healthcheck)
	CustomStats    CustomStatsFunction                // Function that is run on interval to get custom stats
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
	healthStatuses []map[string]*domain.HealthCheckStatus
	customStats    CustomStatsFunction
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
		customStats:        nil,
	}

	if len(svc.HealthChecks) > 0 {
		svc.healthStatuses = make([]map[string]*domain.HealthCheckStatus, len(svc.HealthChecks))

		for instIndex, checkMap := range svc.HealthChecks {

			svc.healthStatuses[instIndex] = make(map[string]*domain.HealthCheckStatus)

			for name, checkDefinition := range checkMap {
				svc.healthStatuses[instIndex][name] = &domain.HealthCheckStatus{
					Name:      name,
					Status:    "unknown",
					Interval:  checkDefinition.Interval.Seconds(),
					Timestamp: 0,
					StartedAt: 0,
				}
			}
		}

	} else {
		name := DEFAULT_HEALTHCHECK_NAME
		svc.healthStatuses = make([]map[string]*domain.HealthCheckStatus, 1)
		svc.healthStatuses[0] = make(map[string]*domain.HealthCheckStatus)
		svc.healthStatuses[0][name] = &domain.HealthCheckStatus{
			Name:      name,
			Status:    "unknown",
			Interval:  3.156e9,
			Timestamp: 0,
			StartedAt: 0,
		}
	}

	if sd.CustomStats != nil {
		svc.customStats = sd.CustomStats
	}

	envPerService[sd.Name] = make(map[string]string)
	envPerService[sd.Name]["CONTROLPLANE_SERVICE_ID"] = svc.ID

	if sd.PreStart != nil {
		if err := sd.PreStart(&svc); err != nil {
			log.WithFields(logrus.Fields{
				"isvc": sd.Name,
			}).WithError(err).Error("Unable to prestart service")
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
	if svc.root == "" {
		svc.root = config.GetOptions().IsvcsPath
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

	log := log.WithFields(logrus.Fields{
		"isvc":    svc.name(),
		"logtype": hostConfig.LogConfig.Type,
	})

	var config dockerclient.Config
	cd := &dockerclient.CreateContainerOptions{
		Name:       svc.name(),
		Config:     &config,
		HostConfig: &hostConfig,
	}

	config.Image = commons.JoinRepoTag(svc.Repo, svc.Tag)
	config.Cmd = []string{"/bin/sh", "-c", "trap 'kill 0' 15; " + svc.Command()}

	// NOTE: USE WITH CARE!
	// Enabling host networking for an isvc may expose ports
	// of the isvcs to access outside of the serviced host, potentially
	// compromising security.
	if svc.HostNetwork {
		cd.HostConfig.NetworkMode = "host"
		log.Info("Using host network stack for internal service")
	}

	// attach all exported ports
	if svc.PortBindings != nil && len(svc.PortBindings) > 0 {
		config.ExposedPorts = make(map[dockerclient.Port]struct{})
		cd.HostConfig.PortBindings = make(map[dockerclient.Port][]dockerclient.PortBinding)
		for _, binding := range svc.PortBindings {
			port := dockerclient.Port(fmt.Sprintf("%d", binding.HostPort))
			config.ExposedPorts[port] = struct{}{}
			portBinding := dockerclient.PortBinding{
				HostIP:   getHostIp(binding),
				HostPort: port.Port(),
			}

			cd.HostConfig.PortBindings[port] = append(cd.HostConfig.PortBindings[port], portBinding)
			log.WithFields(logrus.Fields{
				"bindaddress": fmt.Sprintf("%s:%s", portBinding.HostIP, portBinding.HostPort),
			}).Debug("Bound internal service port to host")
		}
	}

	// copy any links to other isvcs
	if svc.Links != nil && len(svc.Links) > 0 {
		// To use a link, the source container must be instantiated already, so
		//    the service using a link can't be in the first start group.
		//
		// FIXME: Other sanity checks we could add - make sure that the source
		//        container is not in the same group or a later group
		if svc.StartGroup == 0 {
			log.WithFields(logrus.Fields{
				"startgroup": 0,
			}).Fatal("Internal service in the first start group cannot use Docker links")
		}
		cd.HostConfig.Links = make([]string, len(svc.Links))
		copy(cd.HostConfig.Links, svc.Links)
	}

	// attach all exported volumes
	config.Volumes = make(map[string]struct{})
	cd.HostConfig.Binds = []string{}

	// service-specific volumes
	if svc.Volumes != nil && len(svc.Volumes) > 0 {
		for src, dest := range svc.Volumes {
			var hostpath string
			if src == UseServicedLogDir {
				hostpath = utils.ServicedLogDir()
			} else {
				hostpath = svc.getResourcePath(src)
			}
			log := log.WithFields(logrus.Fields{
				"hostpath":      hostpath,
				"containerpath": dest,
			})
			if exists, _ := isDir(hostpath); !exists {
				if err := os.MkdirAll(hostpath, 0777); err != nil {
					log.WithError(err).Debug("Unable to create volume path on host")
					return nil, err
				}
			}

			//TODO It's ugly but we need non-root user for running es bin and the data folder should be
			//  also with non-root privileges
			if strings.Contains(hostpath, "elasticsearch-serviced/data") {
				log.Info("Set chown to es-serviced data directory")
				os.Chown(hostpath, 1001, 1001)
			}

			cd.HostConfig.Binds = append(cd.HostConfig.Binds, fmt.Sprintf("%s:%s", hostpath, dest))
			config.Volumes[dest] = struct{}{}
		}
	}

	// global volumes
	if isvcsVolumes != nil && len(isvcsVolumes) > 0 {
		for src, dest := range isvcsVolumes {
			log := log.WithFields(logrus.Fields{
				"hostpath":      src,
				"containerpath": dest,
			})
			if exists, _ := isDir(src); !exists {
				log.Warn("Unable to mount host path that does not exist")
				continue
			}
			cd.HostConfig.Binds = append(cd.HostConfig.Binds, fmt.Sprintf("%s:%s", src, dest))
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
	log := log.WithFields(logrus.Fields{
		"isvc": svc.name(),
	})
	ctr, _ := docker.FindContainer(svc.name())
	if ctr != nil {
		log := log.WithFields(logrus.Fields{
			"containerid": ctr.ID,
		})
		notify := make(chan int, 1)
		if !ctr.IsRunning() {
			log.Warn("Internal service container found but not running. Removing.")
			go svc.remove(notify)
		} else if !svc.checkVolumes(ctr) {
			// CC-1550: A reload causes CC to re-read its configuration, which means that the host volumes
			//          mounted into the isvcs containers might change. If that happens, we cannot simply
			//          attach to the existing containers. Instead, we need to stop them and create new ones
			//          using the revised configuration.
			ctr.OnEvent(docker.Die, func(cid string) { svc.remove(notify) })
			svc.stop()
			log.Info("Internal service restarted to use revised configuration")
		} else {
			log.Info("Attaching to internal service container")
			return ctr, nil
		}
		<-notify
	}

	log.Debug("Creating a new container for internal service")
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

	log := log.WithFields(logrus.Fields{
		"isvc":        svc.name(),
		"containerid": ctr.ID,
	})

	// perform an initial healthcheck to verify that the service started successfully
	select {
	case err := <-svc.startupHealthcheck():
		if err != nil {
			log.WithFields(logrus.Fields{
				"timeout": svc.StartupTimeout,
			})
			log.WithError(err).Debug("Internal service failed to check in healthy within the timeout")

			// Dump last 10000 lines of container if possible.
			dockerLogsToFile(ctr.ID, 10000)
			if svc.StartupFailed != nil {
				svc.StartupFailed()
			}
			svc.stop()
			return nil, err
		}
		svc.startTime = time.Now()
	case rc := <-notify:
		log.WithFields(logrus.Fields{
			"exitcode": rc,
		}).Error("Internal service exited on startup")
		svc.setStoppedHealthStatus(ExitError(rc))
		return nil, ExitError(rc)
	}

	return notify, nil
}

func (svc *IService) stop() error {
	log := log.WithFields(logrus.Fields{
		"isvc":        svc.Name,
		"containerid": svc.Name,
	})
	ctr, err := docker.FindContainer(svc.name())
	if err == docker.ErrNoSuchContainer {
		svc.setStoppedHealthStatus(nil)
		return nil
	} else if err != nil {
		log.WithError(err).Debug("Unable to get internal service container")
		svc.setStoppedHealthStatus(err)
		return err
	}
	log.Debug("Stopping internal service container")
	err = ctr.Stop(45 * time.Second)
	log.Info("Stopped internal service container")
	svc.setStoppedHealthStatus(err)
	return err
}

func (svc *IService) remove(notify chan<- int) {
	log := log.WithFields(logrus.Fields{
		"isvc":        svc.Name,
		"containerid": svc.Name,
	})
	defer close(notify)
	ctr, err := docker.FindContainer(svc.name())
	if err == docker.ErrNoSuchContainer {
		return
	} else if err != nil {
		log.WithError(err).Debug("Unable to find internal service container")
		return
	}

	// report the log output
	dockerLogsToFile(ctr.ID, 1000)

	// kill the container if it is running
	if ctr.IsRunning() {
		log.Debug("Internal service container still running; killing")
		ctr.Kill()
		log.Warn("Killed internal service container that wouldn't die")
	}

	// get the exit code
	rc, _ := ctr.Wait(time.Second)
	defer func() { notify <- rc }()

	// delete the container
	if err := ctr.Delete(true); err != nil && err != docker.ErrNoSuchContainer {
		log.WithError(err).Warn("Unable to remove internal service container")
	}
}

func (svc *IService) run() {
	var newExited <-chan int
	var err error
	var collecting bool
	haltStats := make(chan struct{})
	haltHealthChecks := make(chan struct{})
	haltCustomStats := make(chan struct{})

	log := log.WithFields(logrus.Fields{
		"isvc": svc.Name,
	})

	for {
		select {
		case req := <-svc.actions:
			switch req.action {
			case stop:
				log.Debug("Stopping internal service container")
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
					log.WithFields(logrus.Fields{
						"exitcode": rc,
					}).Error("An internal service exited with a non-zero exit code")
				}
				svc.setExitedChannel(nil)
				req.response <- nil
			case start:
				log.Debug("Starting internal service")
				if svc.IsRunning() {
					req.response <- ErrRunning
					continue
				}

				newExited, err = svc.start()
				if err != nil && svc.Recover != nil {
					log.WithError(err).Warn("Internal service failed to start. Attempting to recover.")
					if e := svc.Recover(svc.getResourcePath("")); e != nil {
						log.WithError(e).Error("Unable to recover internal service")
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
					if svc.customStats != nil {
						go svc.customStats(haltCustomStats)
					}
					collecting = true
				}

				req.response <- nil
			case restart:
				log.Debug("Restarting internal service")
				if svc.IsRunning() {

					if collecting {
						haltStats <- struct{}{}
						if len(svc.HealthChecks) > 0 {
							haltHealthChecks <- struct{}{}
						}
						if svc.customStats != nil {
							haltCustomStats <- struct{}{}
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
				log.Info("Restarted internal service")
			}
		case rc := <-svc.exited:
			svc.setStoppedHealthStatus(ExitError(rc))
			log.WithFields(logrus.Fields{
				"exitcode": rc,
			}).Warn("Internal service exited unexpectedly")

			stopService := svc.isFlapping()
			if !stopService {
				log.Debug("Restarting internal service")
				newExited, err = svc.start()
				svc.setExitedChannel(newExited)
				if err != nil {
					log.WithError(err).Error("Unable to restart internal service")
					stopService = true
				}
				log.Info("Restarted internal service")
			}

			if stopService {
				if collecting {
					haltStats <- struct{}{}
					if len(svc.HealthChecks) > 0 {
						haltHealthChecks <- struct{}{}
					}
					if svc.customStats != nil {
						haltCustomStats <- struct{}{}
					}
					collecting = false
				}
				log.Error("Not restarting internal service; failed too many times")
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
		log.WithFields(logrus.Fields{
			"containerid": ctr.ID,
		}).WithError(err).Error("Unable to inspect container")
		return false
	}

	if svc.Volumes != nil {
		for src, dest := range svc.Volumes {
			var mount *dockerclient.Mount
			if mount = findContainerMount(dctr, dest); mount == nil {
				log.WithFields(logrus.Fields{
					"isvc":        svc.Name,
					"volume":      dest,
					"containerid": ctr.ID,
				}).Debug("Volume not found in container")
				return false
			}

			expectedSrc, _ := filepath.EvalSymlinks(svc.getResourcePath(src))
			if rel, _ := filepath.Rel(filepath.Clean(expectedSrc), mount.Source); rel != "." {
				log.WithFields(logrus.Fields{
					"isvc":        svc.Name,
					"volume":      dest,
					"containerid": ctr.ID,
					"expected":    expectedSrc,
					"found":       mount.Source,
				}).Debug("Volume mount in container has changed")
				return false
			}
		}
	}

	if isvcsVolumes != nil {
		for src, dest := range isvcsVolumes {
			var mount *dockerclient.Mount
			if mount = findContainerMount(dctr, dest); mount == nil {
				log.WithFields(logrus.Fields{
					"isvc":        svc.Name,
					"volume":      dest,
					"containerid": ctr.ID,
				}).Debug("Global volume not found in container")
				return false
			}

			if rel, _ := filepath.Rel(src, mount.Source); rel != "." {
				log.WithFields(logrus.Fields{
					"isvc":        svc.Name,
					"volume":      dest,
					"containerid": ctr.ID,
					"expected":    src,
					"found":       mount.Source,
				}).Debug("Global volume mount in container has changed")
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

		if len(svc.HealthChecks) == 0 {
			svc.setHealthStatus(nil, time.Now().Unix(), DEFAULT_HEALTHCHECK_NAME, 0)
			err <- nil
			return
		}

		startCheck := time.Now()

		for {
			healthy := true
			currentTime := time.Now()
			elapsed := time.Since(startCheck)

			for instIndex, checkMap := range svc.HealthChecks {
				for healthCheckName, healthCheckDefinition := range checkMap {
					result = svc.runCheckOrTimeout(healthCheckDefinition)
					svc.setHealthStatus(result, currentTime.Unix(), healthCheckName, instIndex)

					// flag if we are not healthy but keep doing the rest of the health checks
					if result != nil {
						healthy = false
					}
				}
			}

			log := log.WithFields(logrus.Fields{
				"isvc":    svc.Name,
				"elapsed": elapsed,
			})

			if healthy == true {
				break
			} else if elapsed.Seconds() > svc.StartupTimeout.Seconds() {
				log.WithFields(logrus.Fields{
					"lastresult": result,
				}).Warn("Unable to verify health of internal service")
				break
			}

			log.Debug("Waiting for internal service to check in healthy")

			time.Sleep(time.Second)
		}

		err <- result
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
		result = fmt.Errorf("healthcheck timed out")
		halt <- struct{}{}
	}
	return result
}

func (svc *IService) doHealthChecks(halt <-chan struct{}) {
	if len(svc.HealthChecks) == 0 {
		log.WithFields(logrus.Fields{
			"healthcheck": "None",
		}).Debug("Health checks not found")
		return
	}

	tickInterval := DEFAULT_HEALTHCHECK_INTERVAL

	// go through the health checks and find the longest duration interval for our ticker.
	// CURRENT - we have one ticker for the longest of the periods for all health checks.
	for _, checkMap := range svc.HealthChecks {
		for healthCheckName, checkDefinition := range checkMap {
			if tickInterval < checkDefinition.Interval {
				log.WithFields(logrus.Fields{
					"healthcheck": healthCheckName,
				}).Warn("Health check interval difference detected")
				tickInterval = checkDefinition.Interval
			}
		}
	}

	timer := time.Tick(tickInterval)

	for {
		select {
		case <-halt:
			log.Debug("Stopped health checks")
			return

		case currentTime := <-timer:
			for index, checkMap := range svc.HealthChecks {
				for healthCheckName, checkDefinition := range checkMap {
					err := svc.runCheckOrTimeout(checkDefinition)
					svc.setHealthStatus(err, currentTime.Unix(), healthCheckName, index)
					if err != nil {
						log.WithFields(logrus.Fields{
							"healthcheck": healthCheckName,
						}).WithError(err).Warn("Health check failed")
					}
				}
			}
		}
	}
}

func (svc *IService) setHealthStatus(result error, currentTime int64, healthCheckName string, instanceIndex int) {

	if len(svc.healthStatuses) == 0 {
		return
	}

	svc.lock.Lock()
	defer svc.lock.Unlock()

	log := log.WithFields(logrus.Fields{
		"isvc": svc.Name,
	})

	if healthStatus, found := svc.healthStatuses[instanceIndex][healthCheckName]; found {
		if result == nil {
			if healthStatus.Status != "passed" && healthStatus.Status != "unknown" {
				log.WithFields(logrus.Fields{
					"healthcheck": healthCheckName,
				}).Info("Internal service health check passed")
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
		log.WithFields(logrus.Fields{
			"healthcheck": healthCheckName,
		}).Warn("Health check not found")
	}
}

func (svc *IService) setStoppedHealthStatus(stopResult error) {
	if len(svc.healthStatuses) == 0 {
		log.WithFields(logrus.Fields{
			"isvc":        svc.Name,
			"healthcheck": "None",
		}).Debug("Health check not found")
		return
	}

	svc.lock.Lock()
	defer svc.lock.Unlock()

	for _, statusMap := range svc.healthStatuses {
		for _, healthStatus := range statusMap {
			healthStatus.Status = "stopped"

			if stopResult == nil {
				healthStatus.Failure = ""
			} else {
				healthStatus.Failure = stopResult.Error()
			}

			healthStatus.Timestamp = time.Now().Unix()
			healthStatus.StartedAt = 0
		}
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

	log := log.WithFields(logrus.Fields{
		"isvc": svc.Name,
	})

	for {
		select {
		case <-halt:
			log.Debug("Stopped collecting stats")
			return
		case t := <-tc:
			ctr, err := docker.FindContainer(svc.name())
			if err != nil {
				log.Warn("Unable to find container")
				break
			}

			if svc.docker == nil {
				log.Warn("Docker API object has not yet been set")
				break
			}

			dockerstats, err := svc.docker.GetContainerStats(ctr.ID, 30*time.Second)
			if err != nil || dockerstats == nil { //dockerstats may be nil if service is shutting down
				log.Warn("Unable to get stats for internal service")
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
					log.WithFields(logrus.Fields{
						"cputotal":         totalCPU,
						"previouscputotal": previousTotalCPU,
					}).Debug("Change in CPU usage was negative. Skipping update")
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
				log.Debug("No previous values to compare to, so skipping CPU stats update")
			}

			// Memory Stats
			pgFault := int64(dockerstats.MemoryStats.Stats.Pgfault)
			totalRSS := int64(dockerstats.MemoryStats.Stats.TotalRss)
			cache := int64(dockerstats.MemoryStats.Stats.Cache)
			if pgFault < 0 || totalRSS < 0 || cache < 0 {
				log.WithFields(logrus.Fields{
					"pgfault":  dockerstats.MemoryStats.Stats.Pgfault,
					"totalrss": dockerstats.MemoryStats.Stats.TotalRss,
					"cache":    dockerstats.MemoryStats.Stats.Cache,
				}).Warn("Memory metric value overflowed int64")
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
				log.WithError(err).Warn("Unable to serialize stats to JSON")
				break
			}
			url := "http://127.0.0.1:4242/api/put"
			req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
			if err != nil {
				log.WithFields(logrus.Fields{
					"url": url,
				}).WithError(err).Warn("Unable to create stats request")
				break
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				// FIXME: This should be a warning, but it happens alot at startup, so let's keep it
				// on the DL until we have some better startup coordination and/or logging mechanisms.
				log.WithFields(logrus.Fields{
					"url": url,
				}).WithError(err).Debug("Unable to POST container stats")
				break
			}
			defer resp.Body.Close()
			if strings.Contains(resp.Status, "204 No Content") == false {
				log.WithFields(logrus.Fields{
					"url":    url,
					"status": resp.Status,
				}).WithError(err).Debug("POST of container stats failed")
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
			if !loggedHostIPOverride {
				log.WithFields(logrus.Fields{
					"hostip": hostIp,
				}).Info("Using host IP override")
				loggedHostIPOverride = true
			}
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

func dockerLogsToFile(containerid string, numlines int) {
	fname := filepath.Join(os.TempDir(), fmt.Sprintf("%s.container.log", containerid))
	f, e := os.Create(fname)
	if e != nil {
		log.WithError(e).WithFields(logrus.Fields{
			"file": fname,
		}).Debug("Unable to create container log file")
		return
	}
	defer f.Close()
	cmd := exec.Command("docker", "logs", "--tail", fmt.Sprintf("%d", numlines), containerid)
	cmd.Stdout = f
	cmd.Stderr = f
	if err := cmd.Run(); err != nil {
		log.WithError(err).Debug("Unable to get logs for container")
		return
	}
	log.WithFields(logrus.Fields{
		"file":  fname,
		"lines": numlines,
	}).Infof("Container log dumped to file")
}
