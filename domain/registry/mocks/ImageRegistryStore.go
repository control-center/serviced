// Copyright 2016 The Serviced Authors.
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

import "github.com/control-center/serviced/domain/registry"
import "github.com/stretchr/testify/mock"

import "github.com/control-center/serviced/datastore"

type ImageRegistryStore struct {
	mock.Mock
}

func (_m *ImageRegistryStore) Get(ctx datastore.Context, id string) (*registry.Image, error) {
	ret := _m.Called(ctx, id)

	var r0 *registry.Image
	if rf, ok := ret.Get(0).(func(datastore.Context, string) *registry.Image); ok {
		r0 = rf(ctx, id)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*registry.Image)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(datastore.Context, string) error); ok {
		r1 = rf(ctx, id)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *ImageRegistryStore) Put(ctx datastore.Context, image *registry.Image) error {
	ret := _m.Called(ctx, image)

	var r0 error
	if rf, ok := ret.Get(0).(func(datastore.Context, *registry.Image) error); ok {
		r0 = rf(ctx, image)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *ImageRegistryStore) Delete(ctx datastore.Context, id string) error {
	ret := _m.Called(ctx, id)

	var r0 error
	if rf, ok := ret.Get(0).(func(datastore.Context, string) error); ok {
		r0 = rf(ctx, id)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *ImageRegistryStore) GetImages(ctx datastore.Context) ([]registry.Image, error) {
	ret := _m.Called(ctx)

	var r0 []registry.Image
	if rf, ok := ret.Get(0).(func(datastore.Context) []registry.Image); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]registry.Image)
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
func (_m *ImageRegistryStore) SearchLibraryByTag(ctx datastore.Context, library string, tag string) ([]registry.Image, error) {
	ret := _m.Called(ctx, library, tag)

	var r0 []registry.Image
	if rf, ok := ret.Get(0).(func(datastore.Context, string, string) []registry.Image); ok {
		r0 = rf(ctx, library, tag)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]registry.Image)
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
