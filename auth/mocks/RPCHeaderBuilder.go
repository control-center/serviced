package mocks

import "github.com/stretchr/testify/mock"

import "io"

type RPCHeaderBuilder struct {
	mock.Mock
}

// WriteHeader provides a mock function with given fields: _a0, _a1, _a2
func (_m *RPCHeaderBuilder) WriteHeader(_a0 io.Writer, _a1 []byte, _a2 bool) error {
	ret := _m.Called(_a0, _a1, _a2)

	var r0 error
	if rf, ok := ret.Get(0).(func(io.Writer, []byte, bool) error); ok {
		r0 = rf(_a0, _a1, _a2)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
