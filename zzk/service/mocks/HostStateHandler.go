package mocks

import zk "github.com/control-center/serviced/zzk/service"
import "github.com/stretchr/testify/mock"

import "time"

type HostStateHandler struct {
	mock.Mock
}

func (_m *HostStateHandler) StopContainer(serviceID string, instanceID int) error {
	ret := _m.Called(serviceID, instanceID)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, int) error); ok {
		r0 = rf(serviceID, instanceID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *HostStateHandler) AttachContainer(state *zk.ServiceState, serviceID string, instanceID int) (<-chan time.Time, error) {
	ret := _m.Called(state, serviceID, instanceID)

	var r0 <-chan time.Time
	if rf, ok := ret.Get(0).(func(*zk.ServiceState, string, int) <-chan time.Time); ok {
		r0 = rf(state, serviceID, instanceID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(<-chan time.Time)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*zk.ServiceState, string, int) error); ok {
		r1 = rf(state, serviceID, instanceID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *HostStateHandler) StartContainer(cancel <-chan interface{}, serviceID string, instanceID int) (*zk.ServiceState, <-chan time.Time, error) {
	ret := _m.Called(cancel, serviceID, instanceID)

	var r0 *zk.ServiceState
	if rf, ok := ret.Get(0).(func(<-chan interface{}, string, int) *zk.ServiceState); ok {
		r0 = rf(cancel, serviceID, instanceID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*zk.ServiceState)
		}
	}

	var r1 <-chan time.Time
	if rf, ok := ret.Get(1).(func(<-chan interface{}, string, int) <-chan time.Time); ok {
		r1 = rf(cancel, serviceID, instanceID)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(<-chan time.Time)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(<-chan interface{}, string, int) error); ok {
		r2 = rf(cancel, serviceID, instanceID)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}
func (_m *HostStateHandler) ResumeContainer(serviceID string, instanceID int) error {
	ret := _m.Called(serviceID, instanceID)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, int) error); ok {
		r0 = rf(serviceID, instanceID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *HostStateHandler) PauseContainer(serviceID string, instanceID int) error {
	ret := _m.Called(serviceID, instanceID)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, int) error); ok {
		r0 = rf(serviceID, instanceID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
