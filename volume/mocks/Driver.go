package mocks

import "github.com/control-center/serviced/volume"
import "github.com/stretchr/testify/mock"

type Driver struct {
	mock.Mock
}

func (m *Driver) Root() string {
	ret := m.Called()

	r0 := ret.Get(0).(string)

	return r0
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
