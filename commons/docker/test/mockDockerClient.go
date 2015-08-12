// Copyright 2015 The Serviced Authors.
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

package test

import (
	"github.com/control-center/serviced/commons/docker"

	dockerclient "github.com/fsouza/go-dockerclient"
	"github.com/stretchr/testify/mock"
)

// assert the interface
var _ docker.ClientInterface = &MockDockerClient{}

type MockDockerClient struct {
	mock.Mock
}

func (mdc *MockDockerClient) CommitContainer(opts dockerclient.CommitContainerOptions) (*dockerclient.Image, error) {
	args := mdc.Mock.Called(opts)
	return args.Get(0).(*dockerclient.Image), args.Error(1)
}

func (mdc *MockDockerClient) CreateContainer(opts dockerclient.CreateContainerOptions) (*dockerclient.Container, error) {
	args := mdc.Mock.Called(opts)
	return args.Get(0).(*dockerclient.Container), args.Error(1)
}

func (mdc *MockDockerClient) ExportContainer(opts dockerclient.ExportContainerOptions) error {
	return mdc.Mock.Called(opts).Error(0)
}

func (mdc *MockDockerClient) KillContainer(opts dockerclient.KillContainerOptions) error {
	return mdc.Mock.Called(opts).Error(0)
}

func (mdc *MockDockerClient) ImportImage(opts dockerclient.ImportImageOptions) error {
	return mdc.Mock.Called(opts).Error(0)
}

func (mdc *MockDockerClient) ExportImages(opts dockerclient.ExportImagesOptions) error {
	return mdc.Mock.Called(opts).Error(0)
}

func (mdc *MockDockerClient) LoadImage(opts dockerclient.LoadImageOptions) error {
	return mdc.Mock.Called(opts).Error(0)
}

func (mdc *MockDockerClient) InspectContainer(id string) (*dockerclient.Container, error) {
	args := mdc.Mock.Called(id)
	return args.Get(0).(*dockerclient.Container), args.Error(1)
}

func (mdc *MockDockerClient) InspectImage(name string) (*dockerclient.Image, error) {
	args := mdc.Mock.Called(name)
	return args.Get(0).(*dockerclient.Image), args.Error(1)
}

func (mdc *MockDockerClient) ListContainers(opts dockerclient.ListContainersOptions) ([]dockerclient.APIContainers, error) {
	args := mdc.Mock.Called(opts)
	return args.Get(0).([]dockerclient.APIContainers), args.Error(1)
}

func (mdc *MockDockerClient) ListImages(opts dockerclient.ListImagesOptions) ([]dockerclient.APIImages, error) {
	args := mdc.Mock.Called(opts)
	var images []dockerclient.APIImages
	if arg0 := args.Get(0); arg0 != nil {
		images = arg0.([]dockerclient.APIImages)
	}
	return images, args.Error(1)
}

func (mdc *MockDockerClient) MonitorEvents() (docker.EventMonitor, error) {
	args := mdc.Mock.Called()
	return args.Get(0).(docker.EventMonitor), args.Error(1)
}

func (mdc *MockDockerClient) PullImage(opts dockerclient.PullImageOptions, auth dockerclient.AuthConfiguration) error {
	return mdc.Mock.Called(opts, auth).Error(0)
}

func (mdc *MockDockerClient) PushImage(opts dockerclient.PushImageOptions, auth dockerclient.AuthConfiguration) error {
	return mdc.Mock.Called(opts, auth).Error(0)
}

func (mdc *MockDockerClient) RemoveContainer(opts dockerclient.RemoveContainerOptions) error {
	return mdc.Mock.Called(opts).Error(0)
}

func (mdc *MockDockerClient) RemoveImage(name string) error {
	return mdc.Mock.Called(name).Error(0)
}

func (mdc *MockDockerClient) RestartContainer(id string, timeout uint) error {
	return mdc.Mock.Called(id, timeout).Error(0)
}

func (mdc *MockDockerClient) StartContainer(id string, hostConfig *dockerclient.HostConfig) error {
	return mdc.Mock.Called(id, hostConfig).Error(0)
}

func (mdc *MockDockerClient) StopContainer(id string, timeout uint) error {
	return mdc.Mock.Called(id, timeout).Error(0)
}

func (mdc *MockDockerClient) TagImage(name string, opts dockerclient.TagImageOptions) error {
	return mdc.Mock.Called(name, opts).Error(0)
}

func (mdc *MockDockerClient) WaitContainer(id string) (int, error) {
	args := mdc.Mock.Called(id)
	return args.Int(0), args.Error(1)
}

func (mdc *MockDockerClient) Version() (*dockerclient.Env, error) {
	return nil, nil
}
