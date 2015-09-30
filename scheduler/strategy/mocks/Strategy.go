package mocks

import "github.com/control-center/serviced/scheduler/strategy"
import "github.com/stretchr/testify/mock"

type Strategy struct {
	mock.Mock
}

func (m *Strategy) Name() string {
	ret := m.Called()

	r0 := ret.Get(0).(string)

	return r0
}
func (m *Strategy) SelectHost(svc strategy.ServiceConfig, hosts []strategy.Host) (strategy.Host, error) {
	ret := m.Called(svc, hosts)

	r0 := ret.Get(0).(strategy.Host)
	r1 := ret.Error(1)

	return r0, r1
}
