package mocks

import "github.com/stretchr/testify/mock"

import elastigocore "github.com/zenoss/elastigo/core"

type ExportLogDriver struct {
	mock.Mock
}

func (_m *ExportLogDriver) SetLogstashInfo(logstashES string) error {
	ret := _m.Called(logstashES)

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(logstashES)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *ExportLogDriver) LogstashDays() ([]string, error) {
	ret := _m.Called()

	var r0 []string
	if rf, ok := ret.Get(0).(func() []string); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]string)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *ExportLogDriver) StartSearch(logstashIndex string, query string) (elastigocore.SearchResult, error) {
	ret := _m.Called(logstashIndex, query)

	var r0 elastigocore.SearchResult
	if rf, ok := ret.Get(0).(func(string, string) elastigocore.SearchResult); ok {
		r0 = rf(logstashIndex, query)
	} else {
		r0 = ret.Get(0).(elastigocore.SearchResult)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, string) error); ok {
		r1 = rf(logstashIndex, query)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *ExportLogDriver) ScrollSearch(scrollID string) (elastigocore.SearchResult, error) {
	ret := _m.Called(scrollID)

	var r0 elastigocore.SearchResult
	if rf, ok := ret.Get(0).(func(string) elastigocore.SearchResult); ok {
		r0 = rf(scrollID)
	} else {
		r0 = ret.Get(0).(elastigocore.SearchResult)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(scrollID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
