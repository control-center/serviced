package mocks

import "github.com/stretchr/testify/mock"

import "time"

import "github.com/control-center/serviced/metrics"

type MetricsClient struct {
	mock.Mock
}

func (_m *MetricsClient) GetInstanceMemoryStats(_a0 time.Time, _a1 ...metrics.ServiceInstance) ([]metrics.MemoryUsageStats, error) {
	ret := _m.Called(_a0, _a1)

	var r0 []metrics.MemoryUsageStats
	if rf, ok := ret.Get(0).(func(time.Time, ...metrics.ServiceInstance) []metrics.MemoryUsageStats); ok {
		r0 = rf(_a0, _a1...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]metrics.MemoryUsageStats)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(time.Time, ...metrics.ServiceInstance) error); ok {
		r1 = rf(_a0, _a1...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
