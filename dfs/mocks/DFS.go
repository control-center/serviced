// Copyright 2014 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package mocks

import "github.com/control-center/serviced/dfs"
import "github.com/stretchr/testify/mock"

import "io"
import "time"

type DFS struct {
	mock.Mock
}

func (_m *DFS) Timeout() time.Duration {
	ret := _m.Called()

	var r0 time.Duration
	if rf, ok := ret.Get(0).(func() time.Duration); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(time.Duration)
	}

	return r0
}
func (_m *DFS) Create(tenantID string) error {
	ret := _m.Called(tenantID)

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(tenantID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *DFS) Destroy(tenantID string) error {
	ret := _m.Called(tenantID)

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(tenantID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *DFS) Download(image string, tenantID string, upgrade bool) (string, error) {
	ret := _m.Called(image, tenantID, upgrade)

	var r0 string
	if rf, ok := ret.Get(0).(func(string, string, bool) string); ok {
		r0 = rf(image, tenantID, upgrade)
	} else {
		r0 = ret.Get(0).(string)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, string, bool) error); ok {
		r1 = rf(image, tenantID, upgrade)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *DFS) Commit(ctrID string) (string, error) {
	ret := _m.Called(ctrID)

	var r0 string
	if rf, ok := ret.Get(0).(func(string) string); ok {
		r0 = rf(ctrID)
	} else {
		r0 = ret.Get(0).(string)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(ctrID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *DFS) Snapshot(info dfs.SnapshotInfo) (string, error) {
	ret := _m.Called(info)

	var r0 string
	if rf, ok := ret.Get(0).(func(dfs.SnapshotInfo) string); ok {
		r0 = rf(info)
	} else {
		r0 = ret.Get(0).(string)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(dfs.SnapshotInfo) error); ok {
		r1 = rf(info)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *DFS) Rollback(snapshotID string) error {
	ret := _m.Called(snapshotID)

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(snapshotID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *DFS) Delete(snapshotID string) error {
	ret := _m.Called(snapshotID)

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(snapshotID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *DFS) List(tenantID string) ([]string, error) {
	ret := _m.Called(tenantID)

	var r0 []string
	if rf, ok := ret.Get(0).(func(string) []string); ok {
		r0 = rf(tenantID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]string)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(tenantID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *DFS) Info(snapshotID string) (*dfs.SnapshotInfo, error) {
	ret := _m.Called(snapshotID)

	var r0 *dfs.SnapshotInfo
	if rf, ok := ret.Get(0).(func(string) *dfs.SnapshotInfo); ok {
		r0 = rf(snapshotID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*dfs.SnapshotInfo)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(snapshotID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *DFS) Backup(info dfs.BackupInfo, w io.Writer) error {
	ret := _m.Called(info, w)

	var r0 error
	if rf, ok := ret.Get(0).(func(dfs.BackupInfo, io.Writer) error); ok {
		r0 = rf(info, w)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *DFS) Restore(r io.Reader) (*dfs.BackupInfo, error) {
	ret := _m.Called(r)

	var r0 *dfs.BackupInfo
	if rf, ok := ret.Get(0).(func(io.Reader) *dfs.BackupInfo); ok {
		r0 = rf(r)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*dfs.BackupInfo)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(io.Reader) error); ok {
		r1 = rf(r)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
