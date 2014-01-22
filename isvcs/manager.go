/*******************************************************************************
* Copyright (C) Zenoss, Inc. 2013, 2014, all rights reserved.
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
	"os"
	"os/exec"
	"path"
)

// managerOp is a type of manager operation (stop, start, reload)
type managerOp int

// constants for the manager operations
const (
	managerOpStart             managerOp = iota // Start the subservices
	managerOpStop                               // stop the subservices
	managerOpReload                             // reload config in subservices
	managerOpExit                               // exit the loop of the manager
	managerOpRegisterContainer                  // register a given container
	managerOpInit                               // make sure manager is ready to run containers
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
	requests      chan managerRequest // the main loops request channel
	containers    map[string]*Container
}

// Returns a new Manager struct and starts the Manager's main loop()
func NewManager(dockerAddress, imagesDir string) *Manager {
	manager := &Manager{
		dockerAddress: dockerAddress,
		imagesDir:     imagesDir,
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
	if client, err := newDockerClient(m.dockerAddress); err != nil {
		return false, err
	} else {
		if images, err := client.ListImages(false); err != nil {
			return false, err
		} else {
			for _, image := range images {
				repoTag := repo + ":" + tag
				for _, tagi := range image.RepoTags {
					if tagi == repoTag {
						return true, nil
					}
				}
			}
		}
	}
	return false, nil
}

// checks for the existence of all the container images
func (m *Manager) processStart() error {
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

// load a docker image fron a tar export
func loadImage(tarball, dockerAddress string) error {

	if file, err := os.Open(tarball); err != nil {
		return err
	} else {
		defer file.Close()
		cmd := exec.Command("docker", "-H", dockerAddress, "load")
		cmd.Stdin = file
		glog.Infof("Loading docker image")
		return cmd.Run()
	}
	return nil
}

// load all the images defined in the registered services
func (m *Manager) loadImages() error {
	loadedImages := make(map[string]bool)
	for _, c := range m.containers {
		if exists, err := m.imageExists(c.Repo, c.Tag); err != nil {
			return err
		} else {
			if exists {
				continue
			}
			localTar := path.Join(m.imagesDir, c.Repo, c.Tag+".tar")
			glog.Infof("Looking for %s", localTar)
			if _, exists := loadedImages[localTar]; exists {
				continue
			}
			if _, err := os.Stat(localTar); err == nil {
				if err := loadImage(localTar, m.dockerAddress); err != nil {
					return err
				}
				loadedImages[localTar] = true
			} else {

			}
		}
	}
	return nil
}

type containerStartResponse struct {
	name string
	err  error
}

func (m *Manager) loop() {

	var running map[string]*Container

	for {
		select {
		case request := <-m.requests:
			switch request.op {
			case managerOpReload:
				var retErr error
				for _, c := range running {
					if c.Reload != nil {
						if err := c.Reload(c, request.val); err != nil {
							retErr = err
						}
					}
				}
				request.response <- retErr
				continue

			case managerOpExit:
				request.response <- nil
				return
			case managerOpStart:
				if running != nil {
					request.response <- ErrManagerRunning
					continue
				}
				if err := m.loadImages(); err != nil {
					request.response <- err
					continue
				}
				if err := m.processStart(); err != nil {
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
							glog.Errorf("%s started with %s", res.name, res.err)
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

func (m *Manager) makeRequest(op managerOp) error {
	request := managerRequest{
		op:       op,
		response: make(chan error),
	}
	m.requests <- request
	return <-request.response
}

func (m *Manager) Register(c *Container) error {
	request := managerRequest{
		op:       managerOpRegisterContainer,
		val:      c,
		response: make(chan error),
	}
	m.requests <- request
	return <-request.response
}

func (m *Manager) Stop() error {
	glog.V(2).Infof("manager sending stop request")
	defer glog.V(2).Infof("received stop response")
	return m.makeRequest(managerOpStop)
}

func (m *Manager) Start() error {
	glog.V(2).Infof("manager sending start request")
	defer glog.V(2).Infof("received start response")
	return m.makeRequest(managerOpStart)
}

func (m *Manager) Reload(val interface{}) error {
	glog.V(2).Infof("manager sending reload request")
	defer glog.V(2).Infof("received reload response")
	request := managerRequest{
		op:       managerOpReload,
		val:      val,
		response: make(chan error),
	}
	m.requests <- request
	return <-request.response
}

func (m *Manager) TearDown() error {
	glog.V(2).Infof("manager sending exit request")
	defer glog.V(2).Infof("received exit response")
	return m.makeRequest(managerOpExit)
}
