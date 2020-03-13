package mocks

import "github.com/control-center/serviced/domain/hostkey"
import "github.com/stretchr/testify/mock"

import "github.com/control-center/serviced/datastore"

type Store struct {
	mock.Mock
}

func (_m *Store) Get(ctx datastore.Context, id string) (*hostkey.RSAKey, error) {
	ret := _m.Called(ctx, id)

	var r0 *hostkey.RSAKey
	if rf, ok := ret.Get(0).(func(datastore.Context, string) *hostkey.RSAKey); ok {
		r0 = rf(ctx, id)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*hostkey.RSAKey)
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
func (_m *Store) Put(ctx datastore.Context, id string, val *hostkey.RSAKey) error {
	ret := _m.Called(ctx, id, val)

	var r0 error
	if rf, ok := ret.Get(0).(func(datastore.Context, string, *hostkey.RSAKey) error); ok {
		r0 = rf(ctx, id, val)
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
