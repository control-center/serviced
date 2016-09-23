package mocks

import "github.com/control-center/serviced/auth"
import "github.com/stretchr/testify/mock"

import "net/rpc"

type RPCHeaderParser struct {
	mock.Mock
}

func (_m *RPCHeaderParser) ParseHeader(_a0 []byte, _a1 *rpc.Request) (auth.Identity, error) {
	ret := _m.Called(_a0, _a1)

	var r0 auth.Identity
	if rf, ok := ret.Get(0).(func([]byte, *rpc.Request) auth.Identity); ok {
		r0 = rf(_a0, _a1)
	} else {
		r0 = ret.Get(0).(auth.Identity)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func([]byte, *rpc.Request) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
