package mocks

import "github.com/stretchr/testify/mock"

import "net/rpc"

type RPCHeaderBuilder struct {
	mock.Mock
}

func (_m *RPCHeaderBuilder) BuildHeader(_a0 *rpc.Request) ([]byte, error) {
	ret := _m.Called(_a0)

	var r0 []byte
	if rf, ok := ret.Get(0).(func(*rpc.Request) []byte); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]byte)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*rpc.Request) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
