package mocks

import "github.com/control-center/serviced/utils"
import "github.com/stretchr/testify/mock"

type ConfigReader struct {
	mock.Mock
}

func (_m *ConfigReader) StringVal(key string, dflt string) string {
	ret := _m.Called(key, dflt)

	var r0 string
	if rf, ok := ret.Get(0).(func(string, string) string); ok {
		r0 = rf(key, dflt)
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}
func (_m *ConfigReader) StringSlice(key string, dflt []string) []string {
	ret := _m.Called(key, dflt)

	var r0 []string
	if rf, ok := ret.Get(0).(func(string, []string) []string); ok {
		r0 = rf(key, dflt)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]string)
		}
	}

	return r0
}
func (_m *ConfigReader) IntVal(key string, dflt int) int {
	ret := _m.Called(key, dflt)

	var r0 int
	if rf, ok := ret.Get(0).(func(string, int) int); ok {
		r0 = rf(key, dflt)
	} else {
		r0 = ret.Get(0).(int)
	}

	return r0
}
func (_m *ConfigReader) BoolVal(key string, dflt bool) bool {
	ret := _m.Called(key, dflt)

	var r0 bool
	if rf, ok := ret.Get(0).(func(string, bool) bool); ok {
		r0 = rf(key, dflt)
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}
func (_m *ConfigReader) GetConfigValues() map[string]utils.ConfigValue {
	ret := _m.Called()

	var r0 map[string]utils.ConfigValue
	if rf, ok := ret.Get(0).(func() map[string]utils.ConfigValue); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(map[string]utils.ConfigValue)
		}
	}

	return r0
}
