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

package mocks

import (
	"io"

	dockerclient "github.com/fsouza/go-dockerclient"
	"github.com/stretchr/testify/mock"
)

type Docker struct {
	mock.Mock
}

func (_m *Docker) FindImage(image string) (*dockerclient.Image, error) {
	ret := _m.Called(image)

	var r0 *dockerclient.Image
	if rf, ok := ret.Get(0).(func(string) *dockerclient.Image); ok {
		r0 = rf(image)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*dockerclient.Image)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(image)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

func (_m *Docker) SaveImages(images []string, writer io.Writer) error {
	ret := _m.Called(images, writer)

	var r0 error
	if rf, ok := ret.Get(0).(func([]string, io.Writer) error); ok {
		r0 = rf(images, writer)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

func (_m *Docker) LoadImage(reader io.Reader) error {
	ret := _m.Called(reader)

	var r0 error
	if rf, ok := ret.Get(0).(func(io.Reader) error); ok {
		r0 = rf(reader)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

func (_m *Docker) PushImage(image string) error {
	ret := _m.Called(image)

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(image)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

func (_m *Docker) PullImage(image string) error {
	ret := _m.Called(image)

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(image)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

func (_m *Docker) TagImage(oldImage string, newImage string) {
	_m.Called(oldImage, newImage)
}

func (_m *Docker) RemoveImage(image string) error {
	ret := _m.Called(image)

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(image)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

func (_m *Docker) ImageHistory(image string) ([]dockerclient.ImageHistory, error) {
	ret := _m.Called(image)

	var r0 []dockerclient.ImageHistory
	if rf, ok := ret.Get(0).(func(string) []dockerclient.ImageHistory); ok {
		r0 = rf(image)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]dockerclient.ImageHistory)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(image)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

func (_m *Docker) FindContainer(ctr string) (*dockerclient.Container, error) {
	ret := _m.Called(ctr)

	var r0 *dockerclient.Container
	if rf, ok := ret.Get(0).(func(string) *dockerclient.Container); ok {
		r0 = rf(ctr)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*dockerclient.Container)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(ctr)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

func (_m *Docker) CommitContainer(ctr string, image string) (*dockerclient.Image, error) {
	ret := _m.Called(ctr, image)

	var r0 *dockerclient.Image
	if rf, ok := ret.Get(0).(func(string, string) *dockerclient.Image); ok {
		r0 = rf(ctr, image)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*dockerclient.Image)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, string) error); ok {
		r1 = rf(ctr, image)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
