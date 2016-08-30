package mocks

import "github.com/stretchr/testify/mock"

type StringMatcher struct {
	mock.Mock
}

func (_m *StringMatcher) MatchString(_a0 string) bool {
	ret := _m.Called(_a0)

	var r0 bool
	if rf, ok := ret.Get(0).(func(string) bool); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}
