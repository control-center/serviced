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

import (
	"net"

	"github.com/stretchr/testify/mock"
)

type Dialer struct {
	mock.Mock
}

func (_m *Dialer) Dial(network, address string) (net.Conn, error) {
	ret := _m.Called(network, address)

	var r0 net.Conn
	if rf, ok := ret.Get(0).(func(string, string) net.Conn); ok {
		r0 = rf(network, address)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(net.Conn)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, string) error); ok {
		r1 = rf(network, address)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
