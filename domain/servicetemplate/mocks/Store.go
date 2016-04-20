package mocks

import "github.com/control-center/serviced/domain/servicetemplate"
import "github.com/stretchr/testify/mock"

import "github.com/control-center/serviced/datastore"

type Store struct {
	mock.Mock
}

func (_m *Store) Get(ctx datastore.Context, id string) (*servicetemplate.ServiceTemplate, error) {
	ret := _m.Called(ctx, id)

	var r0 *servicetemplate.ServiceTemplate
	if rf, ok := ret.Get(0).(func(datastore.Context, string) *servicetemplate.ServiceTemplate); ok {
		r0 = rf(ctx, id)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*servicetemplate.ServiceTemplate)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(datastore.Context, string) error); ok {
		r1 = rf(ctx, id)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *Store) Put(ctx datastore.Context, st servicetemplate.ServiceTemplate) error {
	ret := _m.Called(ctx, st)

	var r0 error
	if rf, ok := ret.Get(0).(func(datastore.Context, servicetemplate.ServiceTemplate) error); ok {
		r0 = rf(ctx, st)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *Store) Delete(ctx datastore.Context, id string) error {
	ret := _m.Called(ctx, id)

	var r0 error
	if rf, ok := ret.Get(0).(func(datastore.Context, string) error); ok {
		r0 = rf(ctx, id)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *Store) GetServiceTemplates(ctx datastore.Context) ([]*servicetemplate.ServiceTemplate, error) {
	ret := _m.Called(ctx)

	var r0 []*servicetemplate.ServiceTemplate
	if rf, ok := ret.Get(0).(func(datastore.Context) []*servicetemplate.ServiceTemplate); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*servicetemplate.ServiceTemplate)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(datastore.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
