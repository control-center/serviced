package mocks

import zks "github.com/control-center/serviced/zzk/service2"
import "github.com/stretchr/testify/mock"

import "time"

import "github.com/control-center/serviced/domain/service"

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
func (_m *HostStateHandler) AttachContainer(state *zks.ServiceState, serviceID string, instanceID int) (<-chan time.Time, error) {
	ret := _m.Called(state, serviceID, instanceID)

	var r0 <-chan time.Time
	if rf, ok := ret.Get(0).(func(*zks.ServiceState, string, int) <-chan time.Time); ok {
		r0 = rf(state, serviceID, instanceID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(<-chan time.Time)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*zks.ServiceState, string, int) error); ok {
		r1 = rf(state, serviceID, instanceID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *HostStateHandler) StartContainer(cancel <-chan interface{}, svc *service.Service, instanceID int) (*zks.ServiceState, <-chan time.Time, error) {
	ret := _m.Called(cancel, svc, instanceID)

	var r0 *zks.ServiceState
	if rf, ok := ret.Get(0).(func(<-chan interface{}, *service.Service, int) *zks.ServiceState); ok {
		r0 = rf(cancel, svc, instanceID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*zks.ServiceState)
		}
	}

	var r1 <-chan time.Time
	if rf, ok := ret.Get(1).(func(<-chan interface{}, *service.Service, int) <-chan time.Time); ok {
		r1 = rf(cancel, svc, instanceID)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(<-chan time.Time)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(<-chan interface{}, *service.Service, int) error); ok {
		r2 = rf(cancel, svc, instanceID)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}
func (_m *HostStateHandler) ResumeContainer(svc *service.Service, instanceID int) error {
	ret := _m.Called(svc, instanceID)

	var r0 error
	if rf, ok := ret.Get(0).(func(*service.Service, int) error); ok {
		r0 = rf(svc, instanceID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *HostStateHandler) PauseContainer(svc *service.Service, instanceID int) error {
	ret := _m.Called(svc, instanceID)

	var r0 error
	if rf, ok := ret.Get(0).(func(*service.Service, int) error); ok {
		r0 = rf(svc, instanceID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
