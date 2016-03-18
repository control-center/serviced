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
	"time"

	"github.com/control-center/serviced/commons"
	"github.com/control-center/serviced/commons/docker"
	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/domain"
	dockerclient "github.com/fsouza/go-dockerclient"
	"github.com/zenoss/glog"

	"errors"
	"fmt"
	"os"
	"path"
	"sort"
	"sync"
)

// managerOp is a type of manager operation (stop, start, notify)
type managerOp int

// constants for the manager operations
const (
	managerOpStart             managerOp = iota // Start the subservices
	managerOpStop                               // stop the subservices
	managerOpNotify                             // notify config in subservices
	managerOpExit                               // exit the loop of the manager
	managerOpRegisterContainer                  // register a given container
	managerOpInit                               // make sure manager is ready to run containers
	managerOpWipe                               // wipe all data associated with volumes
)

var (
	ErrManagerUnknownOp  = errors.New("manager: unknown operation")
	ErrManagerUnknownArg = errors.New("manager: unknown arg type")
	ErrImageNotExists    = errors.New("manager: image does not exist")
	ErrNotifyFailed      = errors.New("manager: notification failure")
)

type StartError int

func (err StartError) Error() string {
	return fmt.Sprintf("manager: could not start %d isvcs", int(err))
}

type StopError int

func (err StopError) Error() string {
	return fmt.Sprintf("manager: could not stop %d isvcs", int(err))
}

// A managerRequest describes an operation for the manager loop() to perform and a response channel
type managerRequest struct {
	op       managerOp // the operation to perform
	val      interface{}
	response chan error // the response channel
}

// A manager of docker services run in ephemeral containers
type Manager struct {
	imagesDir       string              // local directory where images could be loaded from
	volumesDir      string              // local directory where volumes are stored
	requests        chan managerRequest // the main loops request channel
	services        map[string]*IService
	startGroups     map[int][]string  // map by group number of a list of service names
	dockerLogDriver string            // which log driver to use with containers
	dockerLogConfig map[string]string // options for the log driver
}

// Returns a new Manager struct and starts the Manager's main loop()
func NewManager(imagesDir, volumesDir string, dockerLogDriver string, dockerLogConfig map[string]string) *Manager {
	loadvolumes()

	manager := &Manager{
		imagesDir:       imagesDir,
		volumesDir:      volumesDir,
		requests:        make(chan managerRequest),
		services:        make(map[string]*IService),
		startGroups:     make(map[int][]string),
		dockerLogDriver: dockerLogDriver,
		dockerLogConfig: dockerLogConfig,
	}
	go manager.loop()
	return manager
}

// checks to see if the given repo:tag exists in docker
func (m *Manager) imageExists(repo, tag string) (bool, error) {
	if _, err := docker.FindImage(commons.JoinRepoTag(repo, tag), false); docker.IsImageNotFound(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, nil
}

// SetVolumesDir sets the volumes dir for *Manager
func (m *Manager) SetVolumesDir(dir string) {
	m.volumesDir = dir
}

func (m *Manager) SetConfigurationOption(name, key string, value interface{}) error {
	svc, found := m.services[name]
	if !found {
		return errors.New("could not find isvc")
	}
	glog.Infof("setting %s, %s: %s", name, key, value)
	svc.Configuration[key] = value
	return nil
}

// Returns a list of iservice names in sorted order
func (m *Manager) GetServiceNames() []string {
	names := make([]string, 0, len(m.services))
	for name := range m.services {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (m *Manager) GetHealthStatus(name string) (dao.IServiceHealthResult, error) {
	result := dao.IServiceHealthResult{
		ServiceName:    name,
		ContainerName:  "",
		ContainerID:    "",
		HealthStatuses: make([]domain.HealthCheckStatus, 0),
	}

	svc, found := m.services[name]
	if !found {
		glog.Errorf("Internal service %q not found", name)
		return dao.IServiceHealthResult{}, fmt.Errorf("could not find isvc %q", name)
	}

	if ctr, err := docker.FindContainer(svc.name()); err == nil {
		result.ContainerID = ctr.ID
	}

	svc.lock.RLock()
	defer svc.lock.RUnlock()

	result.ContainerName = svc.name()
	for _, value := range svc.healthStatuses {
		result.HealthStatuses = append(result.HealthStatuses, *value)
	}
	return result, nil
}

// checks for the existence of all the container images
func (m *Manager) allImagesExist() error {
	for _, c := range m.services {
		if exists, err := m.imageExists(c.Repo, c.Tag); err != nil {
			return err
		} else {
			if !exists {
				return ErrImageNotExists
			}
		}
	}
	return nil
}

// wipe() removes the data directory associate with the manager
func (m *Manager) wipe() error {

	if err := os.RemoveAll(m.volumesDir); err != nil {
		glog.V(2).Infof("could not remove %s: %v", m.volumesDir, err)
	}
	//nothing to wipe if the volumesDir doesn't exist
	if _, err := os.Stat(m.volumesDir); os.IsNotExist(err) {
		glog.V(2).Infof("Not using docker to remove directories as %s doesn't exist", m.volumesDir)
		return nil
	}
	glog.Infof("Using docker to remove directories in %s", m.volumesDir)

	// remove volumeDir by running a container as root
	// FIXME: detect if already root and avoid running docker
	var config dockerclient.Config
	cd := &docker.ContainerDefinition{
		dockerclient.CreateContainerOptions{Config: &config},
		dockerclient.HostConfig{},
	}

	config.Image = "ubuntu"
	config.Cmd = []string{"/bin/sh", "-c", "rm -Rf /mnt/volumes/*"}
	config.Volumes = map[string]struct{}{
		"/mnt/volumes": struct{}{},
	}

	cd.Binds = []string{m.volumesDir + ":/mnt/volumes"}
	ctr, err := docker.NewContainer(cd, false, 5*time.Second, nil, nil)
	if err != nil {
		return err
	}

	ctr.OnEvent(docker.Die, func(cid string) {
		ctr.Delete(true)
	})

	return ctr.Start()
}

// loadImages() loads all the images defined in the registered services
func (m *Manager) loadImages() error {
	loadedImages := make(map[string]bool)
	for _, c := range m.services {
		glog.V(2).Infof("Checking isvcs container %+v", c)
		if exists, err := m.imageExists(c.Repo, c.Tag); err != nil {
			return err
		} else {
			if exists {
				continue
			}
			localTar := path.Join(m.imagesDir, c.Repo, c.Tag+".tar.gz")
			imageRepoTag := c.Repo + ":" + c.Tag
			glog.Infof("Looking for image %s in tar %s", imageRepoTag, localTar)
			if _, exists := loadedImages[imageRepoTag]; exists {
				continue
			}
			if _, err := os.Stat(localTar); err == nil {
				if err := docker.ImportImage(imageRepoTag, localTar); err != nil {
					return err
				}
				glog.Infof("Loaded %s from %s", imageRepoTag, localTar)
				loadedImages[imageRepoTag] = true
			} else {
				glog.Infof("Pulling image %s", imageRepoTag)
				if err := docker.PullImage(imageRepoTag); err != nil {
					return fmt.Errorf("Failed to pull image %s: %s", imageRepoTag, err)
				}
				glog.Infof("Pulled %s", imageRepoTag)
				loadedImages[imageRepoTag] = true
			}
		}
	}
	return nil
}

type containerStartResponse struct {
	name string
	err  error
}

// loop() maitainers the Manager's state
func (m *Manager) loop() {

	var once sync.Once

	for {
		select {
		case request := <-m.requests:
			switch request.op {
			case managerOpWipe:
				// stop all iservices
				var wg sync.WaitGroup
				for name, svc := range m.services {
					if svc.IsRunning() {
						wg.Add(1)
						go func(svc *IService) {
							defer wg.Done()
							if err := svc.Stop(); err != nil {
								glog.Errorf("Error stopping isvc %s: %s", svc.Name, err)
							}
						}(m.services[name])
					}
				}
				wg.Wait()
				request.response <- m.wipe()
			case managerOpNotify:
				var failed bool
				for _, svc := range m.services {
					if svc.Notify != nil && svc.IsRunning() {
						if err := svc.Notify(svc, request.val); err != nil {
							glog.Errorf("Could not notify isvc %s: %s", svc.Name, err)
							failed = true
						}
					}
				}
				if failed {
					request.response <- ErrNotifyFailed
				} else {
					request.response <- nil
				}
			case managerOpExit:
				request.response <- nil
				return // this will exit the loop()
			case managerOpStart:
				var err error
				once.Do(func() {
					if err = m.loadImages(); err != nil {
						return
					} else if err = m.allImagesExist(); err != nil {
						return
					}
				})
				if err != nil {
					request.response <- err
					continue
				}

				// Start each group of services in group-number order
				unstartedServices := 0
				for _, group := range m.orderedStartGroups() {
					unstartedServices += m.startServiceGroup(group)
				}

				if unstartedServices > 0 {
					request.response <- StartError(unstartedServices)
				} else {
					request.response <- nil
				}

			case managerOpStop:
				// track the number of services that haven't stopped
				var noStop = make([]int, len(m.services))

				// stop services in parallel
				var wg sync.WaitGroup
				index := 0
				for name, svc := range m.services {
					if svc.IsRunning() {
						wg.Add(1)
						go func(svc *IService, i int) {
							defer wg.Done()
							if err := svc.Stop(); err != nil {
								glog.Errorf("Error stopping isvc %s: %s", svc.Name, err)
								noStop[i] = 1
								return
							}
						}(m.services[name], index)
					}
					index++
				}
				wg.Wait()
				count := 0
				for _, i := range noStop {
					count += i
				}
				if count > 0 {
					request.response <- StopError(count)
				} else {
					request.response <- nil
				}
			case managerOpRegisterContainer:
				svc, ok := request.val.(*IService)
				if !ok {
					request.response <- ErrManagerUnknownArg
				} else {
					m.services[svc.Name] = svc
					m.addServiceToStartGroup(svc)
					request.response <- nil
				}
			case managerOpInit:
				request.response <- nil
			default:
				request.response <- ErrManagerUnknownOp
			}
		}
	}
}

func (m *Manager) addServiceToStartGroup(svc *IService) {
	startGroup := int(svc.StartGroup)
	if m.startGroups[startGroup] == nil {
		m.startGroups[startGroup] = make([]string, 0)
	}
	m.startGroups[startGroup] = append(m.startGroups[startGroup], svc.Name)
}

// Returns a list of start groups in numeric order (lowest to highest)
func (m *Manager) orderedStartGroups() []int {
	result := make([]int, len(m.startGroups))
	i := 0
	for key, _ := range m.startGroups {
		result[i] = int(key)
		i++
	}
	sort.Ints(result)
	return result
}

// Start all of the services in the specified start group and wait for all of
// them to finish. Returns the number of services which failed to start.
func (m *Manager) startServiceGroup(group int) int {
	glog.V(1).Infof("Starting isvcs in group %d: %v", group, m.startGroups[group])

	// track the number of services that haven't started
	var noStart = make([]int, len(m.startGroups[group]))

	// start all of the services for this group in parallel
	var wg sync.WaitGroup
	index := 0
	for _, name := range m.startGroups[group] {
		svc := m.services[name]
		if !svc.IsRunning() {
			wg.Add(1)
			go func(svc *IService, i int) {
				defer wg.Done()
				if err := svc.Start(); err != nil {
					glog.Errorf("Error starting isvc %s: %s", svc.Name, err)
					noStart[i] = 1
					return
				}
			}(m.services[name], index)
		}
		index++
	}
	wg.Wait()

	unstartedServices := 0
	for _, i := range noStart {
		unstartedServices += i
	}
	return unstartedServices
}

// makeRequest sends a manager operation request to the *Manager's loop()
func (m *Manager) makeRequest(op managerOp) error {
	request := managerRequest{
		op:       op,
		response: make(chan error),
	}
	m.requests <- request
	return <-request.response
}

// Register() registers a container to be managed by the *Manager
func (m *Manager) Register(svc *IService) error {
	svc.dockerLogDriver = m.dockerLogDriver
	svc.dockerLogConfig = m.dockerLogConfig

	request := managerRequest{
		op:       managerOpRegisterContainer,
		val:      svc,
		response: make(chan error),
	}
	m.requests <- request
	return <-request.response
}

// Wipe() removes the data directory associated with the Manager
func (m *Manager) Wipe() error {
	glog.V(2).Infof("manager sending wipe request")
	defer glog.V(2).Infof("received wipe response")
	return m.makeRequest(managerOpWipe)
}

// Stop() stops all the containers currently registered to the *Manager
func (m *Manager) Stop() error {
	glog.V(2).Infof("manager sending stop request")
	defer glog.V(2).Infof("received stop response")
	return m.makeRequest(managerOpStop)
}

// Start() starts all the containers managed by the *Manager
func (m *Manager) Start() error {
	glog.V(2).Infof("manager sending start request")
	defer glog.V(2).Infof("received start response")
	return m.makeRequest(managerOpStart)
}

// Notify() sends a notify() message to all the containers with the given data val
func (m *Manager) Notify(val interface{}) error {
	glog.V(2).Infof("manager sending notify request")
	defer glog.V(2).Infof("received notify response")
	request := managerRequest{
		op:       managerOpNotify,
		val:      val,
		response: make(chan error),
	}
	m.requests <- request
	return <-request.response
}

// TearDown() causes the *Manager's loop() to exit
func (m *Manager) TearDown() error {
	glog.V(2).Infof("manager sending exit request")
	defer glog.V(2).Infof("received exit response")
	return m.makeRequest(managerOpExit)
}
