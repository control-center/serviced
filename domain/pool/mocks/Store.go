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

import "github.com/control-center/serviced/domain/pool"
import "github.com/stretchr/testify/mock"

import "github.com/control-center/serviced/datastore"

type Store struct {
	mock.Mock
}

func (_m *Store) Put(ctx datastore.Context, key datastore.Key, entity datastore.ValidEntity) error {
	ret := _m.Called(ctx, key, entity)

	var r0 error
	if rf, ok := ret.Get(0).(func(datastore.Context, datastore.Key, datastore.ValidEntity) error); ok {
		r0 = rf(ctx, key, entity)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

func (_m *Store) Get(ctx datastore.Context, key datastore.Key, entity datastore.ValidEntity) error {
	ret := _m.Called(ctx, key, entity)

	var r0 error
	if rf, ok := ret.Get(0).(func(datastore.Context, datastore.Key, datastore.ValidEntity) error); ok {
		r0 = rf(ctx, key, entity)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

func (_m *Store) Delete(ctx datastore.Context, key datastore.Key) error {
	ret := _m.Called(ctx, key)

	var r0 error
	if rf, ok := ret.Get(0).(func(datastore.Context, datastore.Key) error); ok {
		r0 = rf(ctx, key)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

func (_m *Store) GetResourcePools(ctx datastore.Context) ([]pool.ResourcePool, error) {
	ret := _m.Called(ctx)

	var r0 []pool.ResourcePool
	if rf, ok := ret.Get(0).(func(datastore.Context) []pool.ResourcePool); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]pool.ResourcePool)
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
func (_m *Store) GetResourcePoolsByRealm(ctx datastore.Context, realm string) ([]pool.ResourcePool, error) {
	ret := _m.Called(ctx, realm)

	var r0 []pool.ResourcePool
	if rf, ok := ret.Get(0).(func(datastore.Context, string) []pool.ResourcePool); ok {
		r0 = rf(ctx, realm)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]pool.ResourcePool)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(datastore.Context, string) error); ok {
		r1 = rf(ctx, realm)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *Store) HasVirtualIP(ctx datastore.Context, poolID string, virtualIP string) (bool, error) {
	ret := _m.Called(ctx, poolID, virtualIP)

	var r0 bool
	if rf, ok := ret.Get(0).(func(datastore.Context, string, string) bool); ok {
		r0 = rf(ctx, poolID, virtualIP)
	} else {
		r0 = ret.Get(0).(bool)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(datastore.Context, string, string) error); ok {
		r1 = rf(ctx, poolID, virtualIP)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
