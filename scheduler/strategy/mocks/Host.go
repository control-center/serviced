package mocks

import "github.com/control-center/serviced/scheduler/strategy"
import "github.com/stretchr/testify/mock"

type Host struct {
	mock.Mock
}

func (m *Host) HostID() string {
	ret := m.Called()

	r0 := ret.Get(0).(string)

	return r0
}
func (m *Host) TotalCores() int {
	ret := m.Called()

	r0 := ret.Get(0).(int)

	return r0
}
func (m *Host) TotalMemory() uint64 {
	ret := m.Called()

	r0 := ret.Get(0).(uint64)

	return r0
}
func (m *Host) RunningServices() []strategy.ServiceConfig {
	ret := m.Called()

	var r0 []strategy.ServiceConfig
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]strategy.ServiceConfig)
	}

	return r0
}
