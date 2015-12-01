// Copyright 2015 The Serviced Authors.
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

import "github.com/control-center/serviced/volume"
import "github.com/stretchr/testify/mock"

var DriverName volume.DriverType = "mock"

type Driver struct {
	mock.Mock
}

func (m *Driver) Root() string {
	ret := m.Called()

	r0 := ret.Get(0).(string)

	return r0
}

func (m *Driver) DriverType() volume.DriverType {
	return DriverName
}

func (m *Driver) Create(volumeName string) (volume.Volume, error) {
	ret := m.Called(volumeName)

	r0 := ret.Get(0).(volume.Volume)
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Driver) Remove(volumeName string) error {
	ret := m.Called(volumeName)

	r0 := ret.Error(0)

	return r0
}
func (m *Driver) Resize(volumeName string, size uint64) error {
	ret := m.Called(volumeName, size)

	r0 := ret.Error(0)

	return r0
}
func (m *Driver) GetTenant(volumeName string) (volume.Volume, error) {
	ret := m.Called(volumeName)

	r0 := ret.Get(0).(volume.Volume)
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Driver) Get(volumeName string) (volume.Volume, error) {
	ret := m.Called(volumeName)

	r0 := ret.Get(0).(volume.Volume)
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Driver) Release(volumeName string) error {
	ret := m.Called(volumeName)

	r0 := ret.Error(0)

	return r0
}
func (m *Driver) List() []string {
	ret := m.Called()

	var r0 []string
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]string)
	}

	return r0
}
func (m *Driver) Exists(volumeName string) bool {
	ret := m.Called(volumeName)

	r0 := ret.Get(0).(bool)

	return r0
}
func (m *Driver) Cleanup() error {
	ret := m.Called()

	r0 := ret.Error(0)

	return r0
}
func (m *Driver) Status() (*volume.Status, error) {
	ret := m.Called()

	var r0 *volume.Status
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*volume.Status)
	}
	r1 := ret.Error(1)

	return r0, r1
}
