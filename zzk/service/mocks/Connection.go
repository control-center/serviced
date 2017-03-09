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

type Connection struct {
	mock.Mock
}

func (_m *Connection) Close() {
	_m.Called()
}
func (_m *Connection) SetID(_a0 int) {
	_m.Called(_a0)
}
func (_m *Connection) ID() int {
	ret := _m.Called()

	var r0 int
	if rf, ok := ret.Get(0).(func() int); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(int)
	}

	return r0
}
func (_m *Connection) SetOnClose(_a0 func(int)) {
	_m.Called(_a0)
}
func (_m *Connection) NewTransaction() client.Transaction {
	ret := _m.Called()

	var r0 client.Transaction
	if rf, ok := ret.Get(0).(func() client.Transaction); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(client.Transaction)
	}

	return r0
}
func (_m *Connection) Create(path string, node client.Node) error {
	ret := _m.Called(path, node)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, client.Node) error); ok {
		r0 = rf(path, node)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *Connection) CreateDir(path string) error {
	ret := _m.Called(path)

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(path)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *Connection) CreateEphemeral(path string, node client.Node) (string, error) {
	ret := _m.Called(path, node)

	var r0 string
	if rf, ok := ret.Get(0).(func(string, client.Node) string); ok {
		r0 = rf(path, node)
	} else {
		r0 = ret.Get(0).(string)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, client.Node) error); ok {
		r1 = rf(path, node)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

func (_m *Connection) CreateIfExists(path string, node client.Node) error {
	ret := _m.Called(path, node)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, client.Node) error); ok {
		r0 = rf(path, node)
	} else {
		r0 = ret.Get(0).(error)
	}

	return r0
}

func (_m *Connection) CreateEphemeralIfExists(path string, node client.Node) (string, error) {
	ret := _m.Called(path, node)

	var r0 string
	if rf, ok := ret.Get(0).(func(string, client.Node) string); ok {
		r0 = rf(path, node)
	} else {
		r0 = ret.Get(0).(string)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, client.Node) error); ok {
		r1 = rf(path, node)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

func (_m *Connection) EnsurePath(path string) error {
	ret := _m.Called(path)

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(path)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *Connection) Exists(path string) (bool, error) {
	ret := _m.Called(path)

	var r0 bool
	if rf, ok := ret.Get(0).(func(string) bool); ok {
		r0 = rf(path)
	} else {
		r0 = ret.Get(0).(bool)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(path)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *Connection) Delete(path string) error {
	ret := _m.Called(path)

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(path)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

func (_m *Connection) ExistsW(path string, done <-chan struct{}) (bool, <-chan client.Event, error) {
	ret := _m.Called(path, done)

	var r0 bool
	if rf, ok := ret.Get(0).(func(string, <-chan struct{}) bool); ok {
		r0 = rf(path, done)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(bool)
		}
	}

	var r1 <-chan client.Event
	if rf, ok := ret.Get(0).(func(string, <-chan struct{}) <-chan client.Event); ok {
		r1 = rf(path, done)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(<-chan client.Event)
		}
	}
	var r2 error
	if rf, ok := ret.Get(2).(func(string, <-chan struct{}) error); ok {
		r2 = rf(path, done)
	} else {
		if ret.Get(2) != nil {
			r2 = ret.Error(1)
		}
	}

	return r0, r1, r2
}

func (_m *Connection) ChildrenW(path string, done <-chan struct{}) (children []string, event <-chan client.Event, err error) {
	ret := _m.Called(path, done)

	var r0 []string
	if rf, ok := ret.Get(0).(func(string, <-chan struct{}) []string); ok {
		r0 = rf(path, done)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]string)
		}
	}

	var r1 <-chan client.Event
	if rf, ok := ret.Get(0).(func(string, <-chan struct{}) <-chan client.Event); ok {
		r1 = rf(path, done)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(<-chan client.Event)
		}
	}
	var r2 error
	if rf, ok := ret.Get(2).(func(string, <-chan struct{}) error); ok {
		r2 = rf(path, done)
	} else {
		if ret.Get(2) != nil {
			r2 = ret.Error(2)
		}
	}

	return r0, r1, r2
}

func (_m *Connection) Children(p string) (children []string, err error) {
	ret := _m.Called(p)

	//var r0 []string
	if rf, ok := ret.Get(0).(func(string) []string); ok {
		children = rf(p)
	} else {
		if ret.Get(0) != nil {
			children = ret.Get(0).([]string)
		}
	}

	//var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		err = rf(p)
	} else {
		err = ret.Error(1)
	}

	return children, err
}
func (_m *Connection) Get(path string, node client.Node) error {
	ret := _m.Called(path, node)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, client.Node) error); ok {
		r0 = rf(path, node)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *Connection) Set(path string, node client.Node) error {
	ret := _m.Called(path, node)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, client.Node) error); ok {
		r0 = rf(path, node)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *Connection) NewLock(path string) (client.Lock, error) {
	ret := _m.Called(path)

	var r0 client.Lock
	if rf, ok := ret.Get(0).(func(string) client.Lock); ok {
		r0 = rf(path)
	} else {
		r0 = ret.Get(0).(client.Lock)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(path)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Error(1)
		}
	}

	return r0, r1
}

func (_m *Connection) NewLeader(path string) (client.Leader, error) {
	ret := _m.Called(path)

	var r0 client.Leader
	if rf, ok := ret.Get(0).(func(string) client.Leader); ok {
		r0 = rf(path)
	} else {
		r0 = ret.Get(0).(client.Leader)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(path)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Error(1)
		}
	}

	return r0, r1
}

func (_m *Connection) GetW(path string, node client.Node, done <-chan struct{}) (<-chan client.Event, error) {
	ret := _m.Called(path, node, done)

	var r0 <-chan client.Event
	if rf, ok := ret.Get(0).(func(string, client.Node, <-chan struct{}) <-chan client.Event); ok {
		r0 = rf(path, node, done)
	} else {
		r0 = ret.Get(0).(<-chan client.Event)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, client.Node, <-chan struct{}) error); ok {
		r1 = rf(path, node, done)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Error(1)
		}
	}

	return r0, r1
}
