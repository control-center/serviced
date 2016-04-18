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

import "github.com/control-center/serviced/datastore"
import "github.com/stretchr/testify/mock"

type EntityStore struct {
	mock.Mock
}

func (_m *EntityStore) Put(ctx datastore.Context, key datastore.Key, entity datastore.ValidEntity) error {
	ret := _m.Called(ctx, key, entity)

	var r0 error
	if rf, ok := ret.Get(0).(func(datastore.Context, datastore.Key, datastore.ValidEntity) error); ok {
		r0 = rf(ctx, key, entity)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *EntityStore) Get(ctx datastore.Context, key datastore.Key, entity datastore.ValidEntity) error {
	ret := _m.Called(ctx, key, entity)

	var r0 error
	if rf, ok := ret.Get(0).(func(datastore.Context, datastore.Key, datastore.ValidEntity) error); ok {
		r0 = rf(ctx, key, entity)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *EntityStore) Delete(ctx datastore.Context, key datastore.Key) error {
	ret := _m.Called(ctx, key)

	var r0 error
	if rf, ok := ret.Get(0).(func(datastore.Context, datastore.Key) error); ok {
		r0 = rf(ctx, key)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
