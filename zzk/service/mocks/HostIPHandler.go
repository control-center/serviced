package mocks

import "github.com/stretchr/testify/mock"

type HostIPHandler struct {
	mock.Mock
}

func (_m *HostIPHandler) BindIP(ip string, netmask string, iface string) error {
	ret := _m.Called(ip, netmask, iface)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string, string) error); ok {
		r0 = rf(ip, netmask, iface)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *HostIPHandler) ReleaseIP(ip string) error {
	ret := _m.Called(ip)

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(ip)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
