package mocks

import "github.com/control-center/serviced/auth"
import "github.com/stretchr/testify/mock"

import "io"

type RPCHeaderParser struct {
	mock.Mock
}

// ReadHeader provides a mock function with given fields: _a0
func (_m *RPCHeaderParser) ReadHeader(_a0 io.Reader) (auth.Identity, []byte, error) {
	ret := _m.Called(_a0)

	var r0 auth.Identity
	if rf, ok := ret.Get(0).(func(io.Reader) auth.Identity); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Get(0).(auth.Identity)
	}

	var r1 []byte
	if rf, ok := ret.Get(1).(func(io.Reader) []byte); ok {
		r1 = rf(_a0)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).([]byte)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(io.Reader) error); ok {
		r2 = rf(_a0)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}
