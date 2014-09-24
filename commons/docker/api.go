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

package docker

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"time"

	"github.com/control-center/serviced/commons"
	"github.com/zenoss/glog"
	dockerclient "github.com/zenoss/go-dockerclient"
)

// Container represents a Docker container.
type Container struct {
	*dockerclient.Container
	dockerclient.HostConfig
}

// ContainerDefinition records all the information necessary to create a Docker container.
type ContainerDefinition struct {
	dockerclient.CreateContainerOptions
	dockerclient.HostConfig
}

// ContainerActionFunc instances are used to handle container action notifications.
type ContainerActionFunc func(id string)

// Strings identifying the Docker lifecycle events.
const (
	Create  = dockerclient.Create
	Delete  = dockerclient.Delete
	Destroy = dockerclient.Destroy
	Die     = dockerclient.Die
	Export  = dockerclient.Export
	Kill    = dockerclient.Kill
	Restart = dockerclient.Restart
	Start   = dockerclient.Start
	Stop    = dockerclient.Stop
	Untag   = dockerclient.Untag
)

// Container subsystem error types
var (
	ErrAlreadyStarted  = errors.New("docker: container already started")
	ErrRequestTimeout  = errors.New("docker: request timed out")
	ErrKernelShutdown  = errors.New("docker: kernel shutdown")
	ErrNoSuchContainer = errors.New("docker: no such container")
	ErrNoSuchImage     = errors.New("docker: no such image")
)

// NewContainer creates a new container and returns its id. The supplied create action, if
// any, will be executed on successful creation of the container. If a start action is specified
// it will be executed after the container has been started. Note, if the start parameter is
// false the container won't be started and the start action will not be executed.
func NewContainer(cd *ContainerDefinition, start bool, timeout time.Duration, oncreate ContainerActionFunc, onstart ContainerActionFunc) (*Container, error) {
	ec := make(chan error, 1)
	rc := make(chan *dockerclient.Container)

	cmds.Create <- createreq{
		request{ec},
		struct {
			containerOptions *dockerclient.CreateContainerOptions
			hostConfig       *dockerclient.HostConfig
			start            bool
			createaction     ContainerActionFunc
			startaction      ContainerActionFunc
		}{&cd.CreateContainerOptions, &cd.HostConfig, start, oncreate, onstart},
		rc,
	}

	glog.V(2).Info("sent create container request: ", *cd)

	select {
	case <-time.After(timeout):
		glog.V(2).Infof("create container request timedout (%+v)", *cd)
		return nil, ErrRequestTimeout
	case <-done:
		return nil, ErrKernelShutdown
	case err, ok := <-ec:
		switch {
		case !ok:
			return &Container{<-rc, cd.HostConfig}, nil
		default:
			return nil, fmt.Errorf("docker: request failed: %v", err)
		}
	}
}

// FindContainer looks up a container using its id.
func FindContainer(id string) (*Container, error) {
	cl, err := Containers()
	if err != nil {
		return nil, fmt.Errorf("docker: unable to find container %s: %v", id, err)
	}

	for _, c := range cl {
		if c.ID == id {
			return c, nil
		}
	}

	return nil, ErrNoSuchContainer
}

// Containers retrieves a list of all the Docker containers.
func Containers() ([]*Container, error) {
	dc, err := dockerclient.NewClient(dockerep)
	if err != nil {
		return nil, err
	}
	apictrs, err := dc.ListContainers(dockerclient.ListContainersOptions{All: true})
	if err != nil {
		return nil, err
	}
	resp := []*Container{}
	for _, apictr := range apictrs {
		ctr, err := dc.InspectContainer(apictr.ID)
		if err != nil {
			continue
		}
		resp = append(resp, &Container{ctr, dockerclient.HostConfig{}})
	}
	return resp, nil
}

// CancelOnEvent cancels the action associated with the specified event.
func (c *Container) CancelOnEvent(event string) error {
	return cancelOnContainerEvent(event, c.ID)
}

// Commit creates a new Image from the containers changes.
func (c *Container) Commit(iidstr string) (*Image, error) {
	iid, err := commons.ParseImageID(iidstr)
	if err != nil {
		return nil, err
	}

	ec := make(chan error, 1)
	rc := make(chan *Image)

	cmds.Commit <- commitreq{
		request{ec},
		struct {
			containerID string
			imageID     *commons.ImageID
		}{c.ID, iid},
		rc,
	}

	select {
	case <-done:
		return nil, ErrKernelShutdown
	case err, ok := <-ec:
		switch {
		case !ok:
			return <-rc, nil
		default:
			return nil, fmt.Errorf("docker: request failed: %v", err)
		}
	}
}

// Delete removes the container.
func (c *Container) Delete(volumes bool) error {
	dc, err := dockerclient.NewClient(dockerep)
	if err != nil {
		return err
	}
	return dc.RemoveContainer(dockerclient.RemoveContainerOptions{ID: c.ID, RemoveVolumes: volumes})
}

// Export writes the contents of the container's filesystem as a tar archive to outfile.
func (c *Container) Export(outfile *os.File) error {
	dc, err := dockerclient.NewClient(dockerep)
	if err != nil {
		return err
	}
	return dc.ExportContainer(dockerclient.ExportContainerOptions{c.ID, outfile})
}

// Kill sends a SIGKILL signal to the container. If the container is not started
// no action is taken.
func (c *Container) Kill() error {
	dc, err := dockerclient.NewClient(dockerep)
	if err != nil {
		return err
	}
	return dc.KillContainer(dockerclient.KillContainerOptions{ID: c.ID, Signal: dockerclient.SIGKILL})
}

// Inspect returns information about the container specified by id.
func (c *Container) Inspect() (*dockerclient.Container, error) {
	dc, err := dockerclient.NewClient(dockerep)
	if err != nil {
		return nil, err
	}
	return dc.InspectContainer(c.ID)
}

// IsRunning inspects the container and returns true if it is running
func (c *Container) IsRunning() bool {
	cc, err := c.Inspect()
	if err != nil {
		return false
	}

	return cc.State.Running
}

// OnEvent adds an action for the specified event.
func (c *Container) OnEvent(event string, action ContainerActionFunc) error {
	return onContainerEvent(event, c.ID, action)
}

// Restart stops and then restarts a container.
func (c *Container) Restart(timeout time.Duration) error {
	ec := make(chan error, 1)

	cmds.Restart <- restartreq{
		request{ec},
		struct {
			id      string
			timeout uint
		}{c.ID, uint(timeout.Seconds())},
	}

	select {
	case <-time.After(timeout):
		return ErrRequestTimeout
	case <-done:
		return ErrKernelShutdown
	case err, ok := <-ec:
		switch {
		case !ok:
			return nil
		default:
			return fmt.Errorf("docker: request failed: %v", err)
		}
	}
}

// Start uses the information provided in the container definition cd to start a new Docker
// container. If a container can't be started before the timeout expires an error is returned. After the container
// is successfully started the onstart action function is executed.
func (c *Container) Start(timeout time.Duration, onstart ContainerActionFunc) error {
	if c.State.Running != false {
		return nil
	}

	ec := make(chan error, 1)
	rc := make(chan *dockerclient.Container)

	cmds.Start <- startreq{
		request{ec},
		struct {
			id         string
			hostConfig *dockerclient.HostConfig
			action     ContainerActionFunc
		}{c.ID, &c.HostConfig, onstart},
		rc,
	}

	select {
	case <-time.After(timeout):
		return ErrRequestTimeout
	case <-done:
		return ErrKernelShutdown
	case err, ok := <-ec:
		switch {
		case !ok:
			dcc := <-rc
			c.Container = dcc
			return nil
		default:
			return fmt.Errorf("docker: request failed: %v", err)
		}
	}
}

// Stop stops the container specified by the id. If the container can't be stopped before the timeout
// expires an error is returned.
func (c *Container) Stop(timeout time.Duration) error {
	dc, err := dockerclient.NewClient(dockerep)
	if err != nil {
		return err
	}
	return dc.StopContainer(c.ID, uint(timeout.Seconds()))
}

// Wait blocks until the container stops or the timeout expires and then returns its exit code.
func (c *Container) Wait(timeout time.Duration) (int, error) {
	ec := make(chan error, 1)
	rc := make(chan int)

	cmds.Wait <- waitreq{
		request{ec},
		struct {
			id string
		}{c.ID},
		rc,
	}

	select {
	case <-time.After(timeout):
		return -127, ErrRequestTimeout
	case <-done:
		return -127, ErrKernelShutdown
	case err, ok := <-ec:
		switch {
		case !ok:
			return <-rc, nil
		default:
			return -127, fmt.Errorf("docker: request failed: %v", err)
		}
	}
}

// OnContainerCreated associates a containter action with the specified container. The action will be triggered when
// that container is created; since we can't know before it's created what a containers id will be the only really
// useful id is docker.Wildcard which will cause the action to be triggered for every container docker creates.
func OnContainerCreated(id string, action ContainerActionFunc) error {
	return onContainerEvent(dockerclient.Create, id, action)
}

// CancelOnContainerCreated cancels any OnContainerCreated action associated with the specified id - docker.Wildcard is
// the only id that really makes sense.
func CancelOnContainerCreated(id string) error {
	return cancelOnContainerEvent(dockerclient.Create, id)
}

// Image represents a Docker image
type Image struct {
	UUID string
	ID   commons.ImageID
}

// Images returns a list of all the named images in the local repository
func Images() ([]*Image, error) {
	dc, err := dockerclient.NewClient(dockerep)
	if err != nil {
		return nil, err
	}
	imgs, err := dc.ListImages(false)
	if err != nil {
		return nil, err
	}

	re := regexp.MustCompile("<none>:<none>")

	resp := []*Image{}
	for _, img := range imgs {
		for _, repotag := range img.RepoTags {
			if len(re.FindString(repotag)) > 0 {
				continue
			}

			iid, err := commons.ParseImageID(repotag)
			if err != nil {
				return resp, err
			}
			resp = append(resp, &Image{img.ID, *iid})
		}
	}
	return resp, nil
}

// ImportImage creates a new image in the local repository from a file system archive.
func ImportImage(repotag, filename string) error {
	ec := make(chan error, 1)

	cmds.ImageImport <- impimgreq{
		request{ec},
		struct {
			repotag  string
			filename string
		}{repotag, filename},
	}

	select {
	case <-done:
		return ErrKernelShutdown
	case err, ok := <-ec:
		switch {
		case !ok:
			return nil
		default:
			return fmt.Errorf("docker: request failed: %v", err)
		}
	}
}

// FindImage looks up an image by repotag, e.g., zenoss/devimg, from the local repository
// TODO: add a FindImageByFilter that returns collections of images
func FindImage(repotag string, pull bool) (*Image, error) {
	glog.V(1).Infof("looking up image: %s (pull if neccessary %t)", repotag, pull)
	if pull && useRegistry {
		err := PullImage(repotag)
		if err != nil {
			return nil, err
		}
	}

	return lookupImage(repotag)
}

// Delete remove the image from the local repository
func (img *Image) Delete() error {
	dc, err := dockerclient.NewClient(dockerep)
	if err != nil {
		return err
	}
	return dc.RemoveImage(img.ID.String())
}

// Tag tags an image in the local repository
func (img *Image) Tag(tag string) (*Image, error) {
	ec := make(chan error, 1)
	rc := make(chan *Image)

	iid, err := commons.ParseImageID(tag)
	if err != nil {
		return nil, err
	}

	cmds.TagImage <- tagimgreq{
		request{ec},
		struct {
			uuid     string
			name     string
			repo     string
			registry string
			tag      string
		}{img.UUID, img.ID.String(), iid.BaseName(), iid.Registry(), iid.Tag},
		rc,
	}

	select {
	case <-done:
		return nil, ErrKernelShutdown
	case err, ok := <-ec:
		switch {
		case !ok:
			return <-rc, nil
		default:
			return nil, fmt.Errorf("docker: request failed: %v", err)
		}
	}
}

func InspectImage(uuid string) (*dockerclient.Image, error) {
	dc, err := dockerclient.NewClient(dockerep)
	if err != nil {
		return nil, err
	}
	return dc.InspectImage(uuid)
}

func (img *Image) Inspect() (*dockerclient.Image, error) {
	return InspectImage(img.UUID)
}

func ImageHistory(uuid string) ([]*dockerclient.Image, error) {
	layers := make([]*dockerclient.Image, 0, 64)
	for uuid != "" {
		imageInfo, err := InspectImage(uuid)
		if err != nil {
			return layers, err
		}
		layers = append(layers, imageInfo)
		uuid = imageInfo.Parent
	}
	return layers, nil
}

func (img *Image) History() ([]*dockerclient.Image, error) {
	return ImageHistory(img.UUID)
}

func onContainerEvent(event, id string, action ContainerActionFunc) error {
	ec := make(chan error, 1)

	cmds.AddAction <- addactionreq{
		request{ec},
		struct {
			id     string
			event  string
			action ContainerActionFunc
		}{id, event, action},
	}

	select {
	case <-done:
		return ErrKernelShutdown
	case err, ok := <-ec:
		switch {
		case !ok:
			return nil
		default:
			return fmt.Errorf("docker: request failed: %v", err)
		}
	}
}

func cancelOnContainerEvent(event, id string) error {
	ec := make(chan error, 1)

	cmds.CancelAction <- cancelactionreq{
		request{ec},
		struct {
			id    string
			event string
		}{id, event},
	}

	select {
	case <-done:
		return ErrKernelShutdown
	case err, ok := <-ec:
		switch {
		case !ok:
			return nil
		default:
			return fmt.Errorf("docker: request failed: %v", err)
		}
	}
}

func lookupImage(repotag string) (*Image, error) {
	imgs, err := Images()
	if err != nil {
		return nil, err
	}

	iid, err := commons.ParseImageID(repotag)
	if err != nil {
		return nil, err
	}

	if len(iid.Tag) == 0 {
		repotag = repotag + ":latest"
	}

	for _, img := range imgs {
		if img.ID.String() == repotag {
			glog.V(1).Info("found: ", repotag)
			return img, nil
		}
	}

	return nil, ErrNoSuchImage
}

func PullImage(repotag string) error {
	iid, err := commons.ParseImageID(repotag)
	if err != nil {
		return err
	}
	return pullImage(iid.BaseName(), iid.Registry(), iid.Tag)
}

func pullImage(repo, registry, tag string) error {

	dc, err := dockerclient.NewClient(dockerep)
	if err != nil {
		return err
	}

	glog.V(2).Info("pulling image: ", repo)
	opts := dockerclient.PullImageOptions{
		Repository: repo,
		Registry:   registry,
		Tag:        tag,
	}

	err = dc.PullImage(opts, dockerclient.AuthConfiguration{})
	if err != nil {
		glog.V(2).Infof("failed to pull %s: %v", repo, err)
		return err
	}
	return nil
}

// PushImage pushes an image by repotag to local registry, e.g., zenoss/devimg, from the local docker repository
func PushImage(repotag string) error {
	iid, err := commons.ParseImageID(repotag)
	if err != nil {
		return err

	}
	return pushImage(iid.BaseName(), iid.Registry(), iid.Tag)
}

func pushImage(repo, registry, tag string) error {
	dc, err := dockerclient.NewClient(dockerep)
	if err != nil {
		return err
	}

	glog.V(2).Infof("pushing image from repo: %s to registry: %s with tag: %s", repo, registry, tag)
	opts := dockerclient.PushImageOptions{
		Name:     repo,
		Registry: registry,
		Tag:      tag,
	}

	err = dc.PushImage(opts, dockerclient.AuthConfiguration{})
	if err != nil {
		glog.V(2).Infof("failed to push %s: %v", repo, err)
		return err
	}
	return nil
}
