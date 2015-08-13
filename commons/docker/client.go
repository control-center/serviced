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
	dockerclient "github.com/fsouza/go-dockerclient"
)

// The function used to get an instance of ClientInterface
type DockerClientGetter func() (ClientInterface, error)

// The default method used to get a new instance of ClientInterface.
// The default instance is a thin shim around dockerclient.Client
var defaultDockerClientGetter DockerClientGetter = func() (ClientInterface, error) { return NewClient() }

var getDockerClient DockerClientGetter = defaultDockerClientGetter

// Used by tests that need to inject a mock or stub implementation of Client
func SetDockerClientGetter(getterOverride DockerClientGetter) {
	getDockerClient = getterOverride
}

// This Client/ClientInterface is a shim for dockerclient.Client.
// All references to dockerclient.Client shoudl be isolated in Client/ClientInterface
type Client struct {
	dc *dockerclient.Client
}

// An interface respresentation of dockerclient.Client
type ClientInterface interface {
	CommitContainer(opts dockerclient.CommitContainerOptions) (*dockerclient.Image, error)

	CreateContainer(opts dockerclient.CreateContainerOptions) (*dockerclient.Container, error)

	ExportContainer(opts dockerclient.ExportContainerOptions) error

	ImportImage(opts dockerclient.ImportImageOptions) error

	ExportImages(opts dockerclient.ExportImagesOptions) error

	LoadImage(opts dockerclient.LoadImageOptions) error

	InspectContainer(id string) (*dockerclient.Container, error)

	InspectImage(name string) (*dockerclient.Image, error)

	KillContainer(opts dockerclient.KillContainerOptions) error

	ListContainers(opts dockerclient.ListContainersOptions) ([]dockerclient.APIContainers, error)

	ListImages(opts dockerclient.ListImagesOptions) ([]dockerclient.APIImages, error)

	MonitorEvents() (EventMonitor, error)

	PullImage(opts dockerclient.PullImageOptions, auth dockerclient.AuthConfiguration) error

	PushImage(opts dockerclient.PushImageOptions, auth dockerclient.AuthConfiguration) error

	RemoveContainer(opts dockerclient.RemoveContainerOptions) error

	RemoveImage(name string) error

	RestartContainer(id string, timeout uint) error

	StartContainer(id string, hostConfig *dockerclient.HostConfig) error

	StopContainer(id string, timeout uint) error

	TagImage(name string, opts dockerclient.TagImageOptions) error

	WaitContainer(id string) (int, error)
	
	Version() (*dockerclient.Env, error)
}

// assert the interface
var _ ClientInterface = &Client{}

func NewClient() (ClientInterface, error) {
	dc, err := dockerclient.NewClient(dockerep)
	if err != nil {
		return nil, err
	}
	return &Client{dc: dc}, nil
}

func (c *Client) CommitContainer(opts dockerclient.CommitContainerOptions) (*dockerclient.Image, error) {
	return c.dc.CommitContainer(opts)
}

func (c *Client) CreateContainer(opts dockerclient.CreateContainerOptions) (*dockerclient.Container, error) {
	return c.dc.CreateContainer(opts)
}

func (c *Client) ExportContainer(opts dockerclient.ExportContainerOptions) error {
	return c.dc.ExportContainer(opts)
}

func (c *Client) KillContainer(opts dockerclient.KillContainerOptions) error {
	return c.dc.KillContainer(opts)
}

func (c *Client) ImportImage(opts dockerclient.ImportImageOptions) error {
	return c.dc.ImportImage(opts)
}

func (c *Client) ExportImages(opts dockerclient.ExportImagesOptions) error {
	return c.dc.ExportImages(opts)
}

func (c *Client) LoadImage(opts dockerclient.LoadImageOptions) error {
	return c.dc.LoadImage(opts)
}

func (c *Client) InspectContainer(id string) (*dockerclient.Container, error) {
	return c.dc.InspectContainer(id)
}

func (c *Client) InspectImage(name string) (*dockerclient.Image, error) {
	return c.dc.InspectImage(name)
}

func (c *Client) ListContainers(opts dockerclient.ListContainersOptions) ([]dockerclient.APIContainers, error) {
	return c.dc.ListContainers(opts)
}

func (c *Client) ListImages(opts dockerclient.ListImagesOptions) ([]dockerclient.APIImages, error) {
	return c.dc.ListImages(opts)
}

func (c *Client) MonitorEvents() (EventMonitor, error) {
	return c.monitorEvents()
}

func (c *Client) PullImage(opts dockerclient.PullImageOptions, auth dockerclient.AuthConfiguration) error {
	return c.dc.PullImage(opts, auth)
}

func (c *Client) PushImage(opts dockerclient.PushImageOptions, auth dockerclient.AuthConfiguration) error {
	return c.dc.PushImage(opts, auth)
}

func (c *Client) RemoveContainer(opts dockerclient.RemoveContainerOptions) error {
	return c.dc.RemoveContainer(opts)
}

func (c *Client) RemoveImage(name string) error {
	return c.dc.RemoveImage(name)
}

func (c *Client) RestartContainer(id string, timeout uint) error {
	return c.dc.RestartContainer(id, timeout)
}

func (c *Client) StartContainer(id string, hostConfig *dockerclient.HostConfig) error {
	return c.dc.StartContainer(id, hostConfig)
}

func (c *Client) StopContainer(id string, timeout uint) error {
	return c.dc.StopContainer(id, timeout)
}

func (c *Client) TagImage(name string, opts dockerclient.TagImageOptions) error {
	return c.dc.TagImage(name, opts)
}

func (c *Client) WaitContainer(id string) (int, error) {
	return c.dc.WaitContainer(id)
}

func (c *Client) Version() (*dockerclient.Env, error) {
	return c.dc.Version()
}
