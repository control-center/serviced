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
	"github.com/control-center/serviced/domain/host"
	"github.com/stretchr/testify/mock"
)

type RegisteredHostHandler struct {
	mock.Mock
}

func (_m *RegisteredHostHandler) GetRegisteredHosts(cancel <-chan interface{}) ([]host.Host, error) {
	ret := _m.Called(cancel)

	var r0 []host.Host
	if rf, ok := ret.Get(0).(func(<-chan interface{}) []host.Host); ok {
		r0 = rf(cancel)
	} else {
		r0 = ret.Get(0).([]host.Host)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(<-chan interface{}) error); ok {
		r1 = rf(cancel)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
