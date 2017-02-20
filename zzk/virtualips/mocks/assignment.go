// Copyright 2017 The Serviced Authors.
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
	"github.com/control-center/serviced/coordinator/client"
	"github.com/stretchr/testify/mock"
)

type AssignmentHandler struct {
	mock.Mock
}

func (_m *AssignmentHandler) Assign(poolID, ipAddress, netmask, binding string, cancel <-chan interface{}) error {
	ret := _m.Called(poolID, ipAddress, netmask, binding, cancel)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string, string, string, <-chan interface{}) error); ok {
		r0 = rf(poolID, ipAddress, netmask, binding, cancel)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

func (_m *AssignmentHandler) Unassign(poolID, ipAddress string) error {
	ret := _m.Called(poolID, ipAddress)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string) error); ok {
		r0 = rf(poolID, ipAddress)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

func (_m *AssignmentHandler) Watch(poolID, ipAddress string, stop <-chan struct{}) (<-chan client.Event, error) {
	ret := _m.Called(poolID, ipAddress, stop)

	var r0 <-chan client.Event
	if rf, ok := ret.Get(0).(func(string, string, <-chan struct{}) <-chan client.Event); ok {
		r0 = rf(poolID, ipAddress, stop)
	} else {
		r0 = ret.Get(0).(<-chan client.Event)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, string, <-chan struct{}) error); ok {
		r1 = rf(poolID, ipAddress, stop)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Error(1)
		}
	}

	return r0, r1
}
