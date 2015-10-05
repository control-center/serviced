// Copyright 2015 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package mocks

import (
	"github.com/control-center/serviced/datastore"
	registrystore "github.com/control-center/serviced/domain/registry"
	"github.com/stretchr/testify/mock"
)

type MockFacade struct {
	mock.Mock
}

func (_m *MockFacade) GetRegistryImage(ctx datastore.Context, image string) (*registrystore.Image, error) {
	ret := _m.Called(ctx, image)

	var r0 *registrystore.Image
	if rf, ok := ret.Get(0).(func(datastore.Context, string) *registrystore.Image); ok {
		r0 = rf(ctx, image)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*registrystore.Image)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(datastore.Context, string) error); ok {
		r1 = rf(ctx, image)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

func (_m *MockFacade) SetRegistryImage(ctx datastore.Context, rImage *registrystore.Image) error {
	ret := _m.Called(ctx, rImage)

	var r0 error
	if rf, ok := ret.Get(0).(func(datastore.Context, *registrystore.Image) error); ok {
		r0 = rf(ctx, rImage)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

func (_m *MockFacade) DeleteRegistryImage(ctx datastore.Context, image string) error {
	ret := _m.Called(ctx, image)

	var r0 error
	if rf, ok := ret.Get(0).(func(datastore.Context, string) error); ok {
		r0 = rf(ctx, image)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

func (_m *MockFacade) GetRegistryImages(ctx datastore.Context) ([]registrystore.Image, error) {
	ret := _m.Called(ctx)

	var r0 []registrystore.Image
	if rf, ok := ret.Get(0).(func(datastore.Context) []registrystore.Image); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]registrystore.Image)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(datastore.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

func (_m *MockFacade) SearchRegistryLibraryByTag(ctx datastore.Context, library string, tag string) ([]registrystore.Image, error) {
	ret := _m.Called(ctx, library, tag)

	var r0 []registrystore.Image
	if rf, ok := ret.Get(0).(func(datastore.Context, string, string) []registrystore.Image); ok {
		r0 = rf(ctx, library, tag)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]registrystore.Image)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(datastore.Context, string, string) error); ok {
		r1 = rf(ctx, library, tag)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
