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

type Context struct {
	mock.Mock
}

func (_m *Context) Connection() (datastore.Connection, error) {
	ret := _m.Called()

	var r0 datastore.Connection
	if rf, ok := ret.Get(0).(func() datastore.Connection); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(datastore.Connection)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
