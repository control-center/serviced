package test

import "github.com/control-center/serviced/commons/docker"
import "github.com/stretchr/testify/mock"

type MockEventMonitor struct {
	mock.Mock
}

var _ docker.EventMonitor = &MockEventMonitor{}

func (_m *MockEventMonitor) IsActive() bool {
	ret := _m.Called()

	var r0 bool
	if rf, ok := ret.Get(0).(func() bool); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}
func (_m *MockEventMonitor) Subscribe(ID string) (*docker.Subscription, error) {
	ret := _m.Called(ID)

	var r0 *docker.Subscription
	if rf, ok := ret.Get(0).(func(string) *docker.Subscription); ok {
		r0 = rf(ID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*docker.Subscription)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(ID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *MockEventMonitor) Close() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
