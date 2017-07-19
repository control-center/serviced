// Copyright 2017 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package mocks

import (
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain/logfilter"
	"github.com/stretchr/testify/mock"
)

type Store struct {
	mock.Mock
}

func (_m *Store) Get(ctx datastore.Context, name, version string) (*logfilter.LogFilter, error) {
	ret := _m.Called(ctx, name, version)

	var r0 *logfilter.LogFilter
	if rf, ok := ret.Get(0).(func(datastore.Context, string, string) *logfilter.LogFilter); ok {
		r0 = rf(ctx, name, version)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*logfilter.LogFilter)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(datastore.Context, string, string) error); ok {
		r1 = rf(ctx, name, version)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *Store) Put(ctx datastore.Context, val *logfilter.LogFilter) error {
	ret := _m.Called(ctx, val)

	var r0 error
	if rf, ok := ret.Get(0).(func(datastore.Context, *logfilter.LogFilter) error); ok {
		r0 = rf(ctx, val)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *Store) Delete(ctx datastore.Context, name, version string) error {
	ret := _m.Called(ctx, name, version)

	var r0 error
	if rf, ok := ret.Get(0).(func(datastore.Context, string, string) error); ok {
		r0 = rf(ctx, name, version)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
