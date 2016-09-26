package mocks

import "github.com/control-center/serviced/zzk/service"
import "github.com/stretchr/testify/mock"

type ServiceHandler struct {
	mock.Mock
}

func (_m *ServiceHandler) SelectHost(_a0 *service.ServiceNode) (string, error) {
	ret := _m.Called(_a0)

	var r0 string
	if rf, ok := ret.Get(0).(func(*service.ServiceNode) string); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Get(0).(string)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*service.ServiceNode) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
