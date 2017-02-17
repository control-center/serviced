package mocks

import "github.com/control-center/serviced/node"
import "github.com/stretchr/testify/mock"

type VIP struct {
	mock.Mock
}

func (_m *VIP) GetAll() []node.IP {
	ret := _m.Called()

	var r0 []node.IP
	if rf, ok := ret.Get(0).(func() []node.IP); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]node.IP)
		}
	}

	return r0
}
func (_m *VIP) Find(ipprefix string) *node.IP {
	ret := _m.Called(ipprefix)

	var r0 *node.IP
	if rf, ok := ret.Get(0).(func(string) *node.IP); ok {
		r0 = rf(ipprefix)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*node.IP)
		}
	}

	return r0
}
func (_m *VIP) Release(ipaddr string, device string) error {
	ret := _m.Called(ipaddr, device)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string) error); ok {
		r0 = rf(ipaddr, device)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *VIP) Bind(ipaddr string, device string) error {
	ret := _m.Called(ipaddr, device)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string) error); ok {
		r0 = rf(ipaddr, device)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
