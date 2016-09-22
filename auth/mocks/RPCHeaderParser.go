package mocks

import "github.com/control-center/serviced/auth"
import "github.com/stretchr/testify/mock"

type RPCHeaderParser struct {
	mock.Mock
}

func (_m *RPCHeaderParser) ParseHeader(_a0 []byte) (auth.Identity, error) {
	ret := _m.Called(_a0)

	var r0 auth.Identity
	if rf, ok := ret.Get(0).(func([]byte) auth.Identity); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Get(0).(auth.Identity)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func([]byte) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
