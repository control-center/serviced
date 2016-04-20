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

type Connection struct {
	mock.Mock
}

func (_m *Connection) Put(key datastore.Key, data datastore.JSONMessage) error {
	ret := _m.Called(key, data)

	var r0 error
	if rf, ok := ret.Get(0).(func(datastore.Key, datastore.JSONMessage) error); ok {
		r0 = rf(key, data)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *Connection) Get(key datastore.Key) (datastore.JSONMessage, error) {
	ret := _m.Called(key)

	var r0 datastore.JSONMessage
	if rf, ok := ret.Get(0).(func(datastore.Key) datastore.JSONMessage); ok {
		r0 = rf(key)
	} else {
		r0 = ret.Get(0).(datastore.JSONMessage)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(datastore.Key) error); ok {
		r1 = rf(key)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *Connection) Delete(key datastore.Key) error {
	ret := _m.Called(key)

	var r0 error
	if rf, ok := ret.Get(0).(func(datastore.Key) error); ok {
		r0 = rf(key)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *Connection) Query(query interface{}) ([]datastore.JSONMessage, error) {
	ret := _m.Called(query)

	var r0 []datastore.JSONMessage
	if rf, ok := ret.Get(0).(func(interface{}) []datastore.JSONMessage); ok {
		r0 = rf(query)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]datastore.JSONMessage)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(interface{}) error); ok {
		r1 = rf(query)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
