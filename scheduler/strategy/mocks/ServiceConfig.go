package mocks

import "github.com/stretchr/testify/mock"

type ServiceConfig struct {
	mock.Mock
}

func (m *ServiceConfig) GetHostID() string {
	ret := m.Called()

	r0 := ret.Get(0).(string)

	return r0
}
func (m *ServiceConfig) GetServiceID() string {
	ret := m.Called()

	r0 := ret.Get(0).(string)

	return r0
}
func (m *ServiceConfig) RequestedCores() int {
	ret := m.Called()

	r0 := ret.Get(0).(int)

	return r0
}
func (m *ServiceConfig) RequestedMemory() uint64 {
	ret := m.Called()

	r0 := ret.Get(0).(uint64)

	return r0
}
