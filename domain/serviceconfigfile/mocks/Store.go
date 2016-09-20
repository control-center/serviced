package mocks

import "github.com/control-center/serviced/domain/serviceconfigfile"
import "github.com/stretchr/testify/mock"

import "github.com/control-center/serviced/datastore"

type Store struct {
	mock.Mock
}

func (_m *Store) GetConfigFiles(ctx datastore.Context, tenantID string, svcPath string) ([]*serviceconfigfile.SvcConfigFile, error) {
	ret := _m.Called(ctx, tenantID, svcPath)

	var r0 []*serviceconfigfile.SvcConfigFile
	if rf, ok := ret.Get(0).(func(datastore.Context, string, string) []*serviceconfigfile.SvcConfigFile); ok {
		r0 = rf(ctx, tenantID, svcPath)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*serviceconfigfile.SvcConfigFile)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(datastore.Context, string, string) error); ok {
		r1 = rf(ctx, tenantID, svcPath)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *Store) GetConfigFile(ctx datastore.Context, tenantID string, svcPath string, filename string) (*serviceconfigfile.SvcConfigFile, error) {
	ret := _m.Called(ctx, tenantID, svcPath, filename)

	var r0 *serviceconfigfile.SvcConfigFile
	if rf, ok := ret.Get(0).(func(datastore.Context, string, string, string) *serviceconfigfile.SvcConfigFile); ok {
		r0 = rf(ctx, tenantID, svcPath, filename)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*serviceconfigfile.SvcConfigFile)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(datastore.Context, string, string, string) error); ok {
		r1 = rf(ctx, tenantID, svcPath, filename)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
