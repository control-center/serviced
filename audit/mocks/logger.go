package mocks

import "github.com/control-center/serviced/audit"
import "github.com/control-center/serviced/datastore"
import "github.com/Sirupsen/logrus"
import "github.com/stretchr/testify/mock"

type Logger struct {
	mock.Mock
}

func (_m *Logger) Context(context datastore.Context) audit.Logger {
	ret := _m.Called(context)

	var r0 audit.Logger
	if rf, ok := ret.Get(0).(func() audit.Logger); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(audit.Logger)
	}

	return r0
}
func (_m *Logger) Action(action string) audit.Logger {
	ret := _m.Called(action)

	var r0 audit.Logger
	if rf, ok := ret.Get(0).(func() audit.Logger); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(audit.Logger)
	}

	return r0
}
func (_m *Logger) Message(ctx datastore.Context, message string) audit.Logger {
	ret := _m.Called(ctx, message)

	var r0 audit.Logger
	if rf, ok := ret.Get(0).(func() audit.Logger); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(audit.Logger)
	}

	return r0
}
func (_m *Logger) Type(theType string) audit.Logger {
	ret := _m.Called(theType)

	var r0 audit.Logger
	if rf, ok := ret.Get(0).(func() audit.Logger); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(audit.Logger)
	}

	return r0
}
func (_m *Logger) ID(id string) audit.Logger {
	ret := _m.Called(id)

	var r0 audit.Logger
	if rf, ok := ret.Get(0).(func() audit.Logger); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(audit.Logger)
	}

	return r0
}
func (_m *Logger) Entity(entity datastore.Entity) audit.Logger {
	ret := _m.Called(entity)

	var r0 audit.Logger
	if rf, ok := ret.Get(0).(func() audit.Logger); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(audit.Logger)
	}

	return r0
}
func (_m *Logger) WithField(name string, value string) audit.Logger {
	ret := _m.Called(name, value)

	var r0 audit.Logger
	if rf, ok := ret.Get(0).(func() audit.Logger); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(audit.Logger)
	}

	return r0
}
func (_m *Logger) WithFields(fields logrus.Fields) audit.Logger {
	ret := _m.Called(fields)

	var r0 audit.Logger
	if rf, ok := ret.Get(0).(func() audit.Logger); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(audit.Logger)
	}

	return r0
}
func (_m *Logger) Succeeded() {
	_m.Called()
}
func (_m *Logger) Failed() {
	_m.Called()
}
func (_m *Logger) Error(err error) error {
	_m.Called(err)
	return err
}
func (_m *Logger) SucceededIf(value bool) {
	_m.Called(value)
}
