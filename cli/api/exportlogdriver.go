package api

import (
	"github.com/stretchr/testify/mock"
)

type ExportLogDriverMock struct {
	mock.Mock
}

func (_m *ExportLogDriverMock) SetLogstashInfo(logstashES string) error {
	ret := _m.Called(logstashES)

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(logstashES)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *ExportLogDriverMock) LogstashDays() ([]string, error) {
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
func (_m *ExportLogDriverMock) StartSearch(logstashIndex string, query string) (ElasticSearchResults, error) {
	ret := _m.Called(logstashIndex, query)

	var r0 ElasticSearchResults
	if rf, ok := ret.Get(0).(func(string, string) ElasticSearchResults); ok {
		r0 = rf(logstashIndex, query)
	} else {
		r0 = ret.Get(0).(ElasticSearchResults)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, string) error); ok {
		r1 = rf(logstashIndex, query)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *ExportLogDriverMock) ScrollSearch(scrollID string) (ElasticSearchResults, error) {
	ret := _m.Called(scrollID)

	var r0 ElasticSearchResults
	if rf, ok := ret.Get(0).(func(string) ElasticSearchResults); ok {
		r0 = rf(scrollID)
	} else {
		r0 = ret.Get(0).(ElasticSearchResults)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(scrollID)
	} else {
		r1 = ret.Error(1)
	}
	return r0, r1
}
