package mocks

import "github.com/stretchr/testify/mock"

import "github.com/control-center/serviced/domain/service"

type ServiceHandler struct {
	mock.Mock
}

func (_m *ServiceHandler) SelectHost(_a0 *service.Service) (string, string, error) {
	ret := _m.Called(_a0)

	var r0 string
	if rf, ok := ret.Get(0).(func(*service.Service) string); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Get(0).(string)
	}

	var r1 string
	if rf, ok := ret.Get(1).(func(*service.Service) string); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Get(1).(string)
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(*service.Service) error); ok {
		r2 = rf(_a0)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}
