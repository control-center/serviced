package mocks

import "github.com/stretchr/testify/mock"

import "github.com/Sirupsen/logrus"

type LogControl struct {
	mock.Mock
}

func (_m *LogControl) SetLevel(level logrus.Level) {
	_m.Called(level)
}
func (_m *LogControl) ApplyConfigFromFile(file string) error {
	ret := _m.Called(file)

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(file)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *LogControl) WatchConfigFile(file string) {
	_m.Called(file)
}
func (_m *LogControl) SetVerbosity(value int) {
	_m.Called(value)
}
func (_m *LogControl) GetVerbosity() int {
	ret := _m.Called()

	var r0 int
	if rf, ok := ret.Get(0).(func() int); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(int)
	}

	return r0
}
func (_m *LogControl) SetToStderr(value bool) {
	_m.Called(value)
}
func (_m *LogControl) SetAlsoToStderr(value bool) {
	_m.Called(value)
}
func (_m *LogControl) SetStderrThreshold(value string) error {
	ret := _m.Called(value)

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(value)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *LogControl) SetVModule(value string) error {
	ret := _m.Called(value)

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(value)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *LogControl) SetTraceLocation(value string) error {
	ret := _m.Called(value)

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(value)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
