package mocks

import "github.com/control-center/serviced/dfs"
import "github.com/stretchr/testify/mock"

import "io"
import "time"

import "github.com/control-center/serviced/domain/service"

type DFS struct {
	mock.Mock
}

// Lock provides a mock function with given fields: opName
func (_m *DFS) Lock(opName string) {
	return
}

// LockWithTimeout provides a mock function with given fields: opName, timeout
func (_m *DFS) LockWithTimeout(opName string, timeout time.Duration) error {
	return nil
}

// Unlock provides a mock function with given fields:
func (_m *DFS) Unlock() {
	return
}

// Timeout provides a mock function with given fields:
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

// Create provides a mock function with given fields: tenantID
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

// Destroy provides a mock function with given fields: tenantID
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

// Download provides a mock function with given fields: image, tenantID, upgrade
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

// Commit provides a mock function with given fields: ctrID
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

// Snapshot provides a mock function with given fields: info
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

// Rollback provides a mock function with given fields: snapshotID
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

// Delete provides a mock function with given fields: snapshotID
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

// List provides a mock function with given fields: tenantID
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

// Info provides a mock function with given fields: snapshotID
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

// Backup provides a mock function with given fields: info, w
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

// Restore provides a mock function with given fields: r, backupInfo
func (_m *DFS) Restore(r io.Reader, backupInfo *dfs.BackupInfo) error {
	ret := _m.Called(r, backupInfo)

	var r0 error
	if rf, ok := ret.Get(0).(func(io.Reader, *dfs.BackupInfo) error); ok {
		r0 = rf(r, backupInfo)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// BackupInfo provides a mock function with given fields: r
func (_m *DFS) BackupInfo(r io.Reader) (*dfs.BackupInfo, error) {
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

// Tag provides a mock function with given fields: snapshotID, tagName
func (_m *DFS) Tag(snapshotID string, tagName string) error {
	ret := _m.Called(snapshotID, tagName)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string) error); ok {
		r0 = rf(snapshotID, tagName)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Untag provides a mock function with given fields: tenantID, tagName
func (_m *DFS) Untag(tenantID string, tagName string) (string, error) {
	ret := _m.Called(tenantID, tagName)

	var r0 string
	if rf, ok := ret.Get(0).(func(string, string) string); ok {
		r0 = rf(tenantID, tagName)
	} else {
		r0 = ret.Get(0).(string)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, string) error); ok {
		r1 = rf(tenantID, tagName)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// TagInfo provides a mock function with given fields: tenantID, tagName
func (_m *DFS) TagInfo(tenantID string, tagName string) (*dfs.SnapshotInfo, error) {
	ret := _m.Called(tenantID, tagName)

	var r0 *dfs.SnapshotInfo
	if rf, ok := ret.Get(0).(func(string, string) *dfs.SnapshotInfo); ok {
		r0 = rf(tenantID, tagName)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*dfs.SnapshotInfo)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, string) error); ok {
		r1 = rf(tenantID, tagName)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// UpgradeRegistry provides a mock function with given fields: svcs, tenantID, registryHost, override
func (_m *DFS) UpgradeRegistry(svcs []service.Service, tenantID string, registryHost string, override bool) error {
	ret := _m.Called(svcs, tenantID, registryHost, override)

	var r0 error
	if rf, ok := ret.Get(0).(func([]service.Service, string, string, bool) error); ok {
		r0 = rf(svcs, tenantID, registryHost, override)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Override provides a mock function with given fields: newImage, oldImage
func (_m *DFS) Override(newImage string, oldImage string) error {
	ret := _m.Called(newImage, oldImage)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string) error); ok {
		r0 = rf(newImage, oldImage)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
