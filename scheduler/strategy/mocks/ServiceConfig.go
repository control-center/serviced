package mocks

import "github.com/stretchr/testify/mock"

import "github.com/control-center/serviced/domain/servicedefinition"

type ServiceConfig struct {
	mock.Mock
}

func (m *ServiceConfig) GetServiceID() string {
	ret := m.Called()

	r0 := ret.Get(0).(string)

	return r0
}
func (m *ServiceConfig) RequestedCorePercent() int {
	ret := m.Called()

	r0 := ret.Get(0).(int)

	return r0
}
func (m *ServiceConfig) RequestedMemoryBytes() uint64 {
	ret := m.Called()

	r0 := ret.Get(0).(uint64)

	return r0
}
func (m *ServiceConfig) HostPolicy() servicedefinition.HostPolicy {
	ret := m.Called()

	r0 := ret.Get(0).(servicedefinition.HostPolicy)

	return r0
}
