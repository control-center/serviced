package mocks

import "github.com/stretchr/testify/mock"

import "time"

type TTL struct {
	mock.Mock
}

func (_m *TTL) Purge(_a0 time.Duration) (time.Duration, error) {
	ret := _m.Called(_a0)

	var r0 time.Duration
	if rf, ok := ret.Get(0).(func(time.Duration) time.Duration); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Get(0).(time.Duration)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(time.Duration) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
