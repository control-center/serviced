package mocks

import "github.com/control-center/serviced/volume"
import "github.com/stretchr/testify/mock"

type Volume struct {
	mock.Mock
}

func (m *Volume) Name() string {
	ret := m.Called()

	r0 := ret.Get(0).(string)

	return r0
}
func (m *Volume) Path() string {
	ret := m.Called()

	r0 := ret.Get(0).(string)

	return r0
}
func (m *Volume) Driver() volume.Driver {
	ret := m.Called()

	r0 := ret.Get(0).(volume.Driver)

	return r0
}
func (m *Volume) Snapshot(label string) error {
	ret := m.Called(label)

	r0 := ret.Error(0)

	return r0
}
func (m *Volume) Snapshots() ([]string, error) {
	ret := m.Called()

	var r0 []string
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]string)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Volume) RemoveSnapshot(label string) error {
	ret := m.Called(label)

	r0 := ret.Error(0)

	return r0
}
func (m *Volume) Rollback(label string) error {
	ret := m.Called(label)

	r0 := ret.Error(0)

	return r0
}
func (m *Volume) Export(label string, parent string, filename string) error {
	ret := m.Called(label, parent, filename)

	r0 := ret.Error(0)

	return r0
}
func (m *Volume) Import(label string, filename string) error {
	ret := m.Called(label, filename)

	r0 := ret.Error(0)

	return r0
}
func (m *Volume) Tenant() string {
	ret := m.Called()

	r0 := ret.Get(0).(string)

	return r0
}
