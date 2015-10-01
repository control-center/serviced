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
	"github.com/control-center/serviced/domain/registry"
	"github.com/stretchr/testify/mock"
)

type Registry struct {
	mock.Mock
}

func (_m *Registry) GetImage(image string) *registry.Image {
	ret := _m.Called(image)

	var r0 *registry.Image
	if rf, ok := ret.Get(0).(func(string) *registry.Image); ok {
		r0 = rf(image)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*registry.Image)
		}
	}

	return r0
}

func (_m *Registry) PushImage(image string, uuid string) (string, error) {
	ret := _m.Called(image, uuid)

	var r0 string
	if rf, ok := ret.Get(0).(func(string, string) string); ok {
		r0 = rf(image, uuid)
	} else {
		r0 = ret.Get(0).(string)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, string) error); ok {
		r1 = rf(image, uuid)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

func (_m *Registry) PullImage(image string) (string, error) {
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

func (_m *Registry) DeleteImage(image string) error {
	ret := _m.Called(image)

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(image)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

func (_m *Registry) SearchLibraryByTag(library string, tag string) ([]registry.Image, error) {
	ret := _m.Called(library, tag)

	var r0 []registry.Image
	if rf, ok := ret.Get(0).(func(string, string) []registry.Image); ok {
		r0 = rf(library, tag)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]registry.Image)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, string) error); ok {
		r1 = rf(library, tag)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
