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
	"time"

	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/domain/registry"
	dockerclient "github.com/fsouza/go-dockerclient"
	"github.com/stretchr/testify/mock"
)

type Registry struct {
	mock.Mock
}

func (_m *Registry) SetConnection(conn client.Connection) {
	ret := _m.Called(conn)

	if rf, ok := ret.Get(0).(func(client.Connection)); ok {
		rf(conn)
	}
	return
}
func (_m *Registry) PullImage(cancel <-chan time.Time, image string) error {
	ret := _m.Called(cancel, image)

	var r0 error
	if rf, ok := ret.Get(0).(func(<-chan time.Time, string) error); ok {
		r0 = rf(cancel, image)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *Registry) ImagePath(image string) (string, error) {
	ret := _m.Called(image)

	var r0 string
	if rf, ok := ret.Get(0).(func(string) string); ok {
		r0 = rf(image)
	} else {
		r0 = ret.Get(0).(string)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(image)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *Registry) FindImage(rImg *registry.Image) (*dockerclient.Image, error) {
	ret := _m.Called(rImg)

	var r0 *dockerclient.Image
	if rf, ok := ret.Get(0).(func(*registry.Image) *dockerclient.Image); ok {
		r0 = rf(rImg)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*dockerclient.Image)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*registry.Image) error); ok {
		r1 = rf(rImg)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
