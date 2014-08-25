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
	"github.com/zenoss/glog"
	"github.com/zenoss/go-dockerclient"

	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"
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

var ErrManagerUnknownOp error
var ErrManagerNotRunning error
var ErrManagerRunning error
var ErrImageNotExists error

func init() {
	ErrManagerUnknownOp = errors.New("manager: unknown operation")
	ErrManagerNotRunning = errors.New("manager: not running")
	ErrManagerRunning = errors.New("manager: already running")
	ErrImageNotExists = errors.New("manager: image does not exist")
}

// A managerRequest describes an operation for the manager loop() to perform and a response channel
type managerRequest struct {
	op       managerOp // the operation to perform
	val      interface{}
	response chan error // the response channel
}

// A manager of docker services run in ephemeral containers
type Manager struct {
	dockerAddress string              // the docker endpoint address to talk to
	imagesDir     string              // local directory where images could be loaded from
	volumesDir    string              // local directory where volumes are stored
	requests      chan managerRequest // the main loops request channel
	containers    map[string]*Container
}

// Returns a new Manager struct and starts the Manager's main loop()
func NewManager(dockerAddress, imagesDir, volumesDir string) *Manager {
	manager := &Manager{
		dockerAddress: dockerAddress,
		imagesDir:     imagesDir,
		volumesDir:    volumesDir,
		requests:      make(chan managerRequest),
		containers:    make(map[string]*Container),
	}
	go manager.loop()
	return manager
}

// newDockerClient is a function pointer to the client contructor so that it can be mocked in tests
var newDockerClient func(address string) (*docker.Client, error)

func init() {
	newDockerClient = docker.NewClient
}

// checks to see if the given repo:tag exists in docker
func (m *Manager) imageExists(repo, tag string) (bool, error) {
	glog.V(1).Infof("Checking for imageExists %s:%s", repo, tag)
	if client, err := newDockerClient(m.dockerAddress); err != nil {
		glog.Errorf("unable to start docker client at docker address: %+v", m.dockerAddress)
		return false, err
	} else {
		repoTag := repo + ":" + tag
		if images, err := client.ListImages(false); err != nil {
			return false, err
		} else {
			for _, image := range images {
				for _, tagi := range image.RepoTags {
					if string(tagi) == repoTag {
						return true, nil
					}
				}
			}
		}
	}
	return false, nil
}

// SetVolumesDir sets the volumes dir for *Manager
func (m *Manager) SetVolumesDir(dir string) {
	m.volumesDir = dir
}

func (m *Manager) SetConfigurationOption(container, key string, value interface{}) error {
	c, found := m.containers[container]
	if !found {
		return errors.New("could not find container")
	}
	glog.Infof("setting %s, %s: %s", container, key, value)
	c.Configuration[key] = value
	return nil
}

// checks for the existence of all the container images
func (m *Manager) allImagesExist() error {
	for _, c := range m.containers {
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

// loadImage() loads a docker image from a tar export
func loadImage(tarball, dockerAddress, repoTag string) error {

	if file, err := os.Open(tarball); err != nil {
		return err
	} else {
		defer file.Close()
		cmd := exec.Command("docker", "-H", dockerAddress, "import", "-")
		cmd.Stdin = file
		glog.Infof("Loading docker image %+v with docker import cmd: %+v", repoTag, cmd.Args)
		output, err := cmd.CombinedOutput()
		if err != nil {
			glog.Errorf("unable to import docker image %+v with command:%+v output:%s err:%s", repoTag, cmd.Args, output, err)
			return err
		}

		importedImageName := strings.Trim(string(output), "\n")
		tagcmd := exec.Command("docker", "-H", dockerAddress, "tag", importedImageName, repoTag)
		glog.Infof("Tagging imported docker image %s with tag %s using docker tag cmd: %+v", importedImageName, repoTag, tagcmd.Args)
		output, err = tagcmd.CombinedOutput()
		if err != nil {
			glog.Errorf("unable to tag imported image %s using command:%+v output:%s err: %s\n", importedImageName, tagcmd.Args, output, err)
			return err
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
	cmd := exec.Command("docker", "-H", m.dockerAddress,
		"run", "--rm", "-v", m.volumesDir+":/mnt/volumes:rw", "ubuntu", "/bin/sh", "-c", "rm -Rf /mnt/volumes/*")
	return cmd.Run()
}

// loadImages() loads all the images defined in the registered services
func (m *Manager) loadImages() error {
	loadedImages := make(map[string]bool)
	for _, c := range m.containers {
		glog.Infof("Checking isvcs container %+v", c)
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
				if err := loadImage(localTar, m.dockerAddress, imageRepoTag); err != nil {
					return err
				}
				glog.Infof("Loaded %s from %s", imageRepoTag, localTar)
				loadedImages[imageRepoTag] = true
			} else {
				glog.Infof("Pulling image %s", imageRepoTag)
				cmd := exec.Command("docker", "-H", m.dockerAddress, "pull", imageRepoTag)
				if output, err := cmd.CombinedOutput(); err != nil {
					return fmt.Errorf("Failed to pull image:(%s) %s ", err, output)
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

	var running map[string]*Container

	for {
		select {
		case request := <-m.requests:
			switch request.op {
			case managerOpWipe:
				if running != nil {
					request.response <- ErrManagerRunning
					continue
				}
				//TODO: didn't  we just check running for nil?
				responses := make(chan error, len(running))
				for _, c := range running {
					go func(con *Container) {
						responses <- con.Stop()
					}(c)
				}
				runningCount := len(running)
				for i := 0; i < runningCount; i++ {
					<-responses
				}
				running = nil
				request.response <- m.wipe()

			case managerOpNotify:
				var retErr error
				for _, c := range running {
					if c.Notify != nil {
						if err := c.Notify(c, request.val); err != nil {
							retErr = err
						}
					}
				}
				request.response <- retErr
				continue

			case managerOpExit:
				request.response <- nil
				return // this will exit the loop()

			case managerOpStart:
				if running != nil {
					request.response <- ErrManagerRunning
					continue
				}

				if err := m.loadImages(); err != nil {
					request.response <- err
					continue
				}
				if err := m.allImagesExist(); err != nil {
					request.response <- err
				} else {
					// start a map of running containers
					running = make(map[string]*Container)

					// start a channel to track responses
					started := make(chan containerStartResponse, len(m.containers))

					// start containers in parallel
					for _, c := range m.containers {
						running[c.Name] = c
						go func(con *Container, respc chan containerStartResponse) {
							glog.Infof("calling start on %s", con.Name)
							con.SetVolumesDir(m.volumesDir)
							resp := containerStartResponse{
								name: con.Name,
								err:  con.Start(),
							}
							respc <- resp
						}(c, started)
					}

					// wait for containers to respond to start
					var returnErr error
					for _, _ = range m.containers {
						res := <-started
						if res.err != nil {
							returnErr = res.err
							glog.Errorf("%s failed with %s", res.name, res.err)
							delete(running, res.name)
						} else {
							glog.Infof("%s started", res.name)
						}
					}
					request.response <- returnErr
				}
			case managerOpStop:
				if running == nil {
					request.response <- ErrManagerNotRunning
					continue
				}
				responses := make(chan error, len(running))
				for _, c := range running {
					go func(con *Container) {
						responses <- con.Stop()
					}(c)
				}
				runningCount := len(running)
				for i := 0; i < runningCount; i++ {
					<-responses
				}
				running = nil
				request.response <- nil
			case managerOpRegisterContainer:
				if running != nil {
					request.response <- ErrManagerRunning
					continue
				}
				if container, ok := request.val.(*Container); !ok {
					panic(errors.New("manager unknown arg type"))
				} else {
					m.containers[container.Name] = container
					request.response <- nil
				}
				continue
			case managerOpInit:
				request.response <- nil

			default:
				request.response <- ErrManagerUnknownOp
			}
		}
	}
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
func (m *Manager) Register(c *Container) error {
	request := managerRequest{
		op:       managerOpRegisterContainer,
		val:      c,
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
