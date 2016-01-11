package mocks

import "github.com/control-center/serviced/volume"
import "github.com/stretchr/testify/mock"

import "io"

type Volume struct {
	mock.Mock
}

func (_m *Volume) Name() string {
	ret := _m.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}
func (_m *Volume) Path() string {
	ret := _m.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}
func (_m *Volume) Driver() volume.Driver {
	ret := _m.Called()

	var r0 volume.Driver
	if rf, ok := ret.Get(0).(func() volume.Driver); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(volume.Driver)
	}

	return r0
}
func (_m *Volume) Snapshot(label string, message string, tags []string) error {
	ret := _m.Called(label, message, tags)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string, []string) error); ok {
		r0 = rf(label, message, tags)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *Volume) SnapshotInfo(label string) (*volume.SnapshotInfo, error) {
	ret := _m.Called(label)

	var r0 *volume.SnapshotInfo
	if rf, ok := ret.Get(0).(func(string) *volume.SnapshotInfo); ok {
		r0 = rf(label)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*volume.SnapshotInfo)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(label)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *Volume) WriteMetadata(label string, name string) (io.WriteCloser, error) {
	ret := _m.Called(label, name)

	var r0 io.WriteCloser
	if rf, ok := ret.Get(0).(func(string, string) io.WriteCloser); ok {
		r0 = rf(label, name)
	} else {
		r0 = ret.Get(0).(io.WriteCloser)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, string) error); ok {
		r1 = rf(label, name)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *Volume) ReadMetadata(label string, name string) (io.ReadCloser, error) {
	ret := _m.Called(label, name)

	var r0 io.ReadCloser
	if rf, ok := ret.Get(0).(func(string, string) io.ReadCloser); ok {
		r0 = rf(label, name)
	} else {
		r0 = ret.Get(0).(io.ReadCloser)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, string) error); ok {
		r1 = rf(label, name)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *Volume) Snapshots() ([]string, error) {
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
func (_m *Volume) RemoveSnapshot(label string) error {
	ret := _m.Called(label)

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(label)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *Volume) Rollback(label string) error {
	ret := _m.Called(label)

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(label)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *Volume) TagSnapshot(label string, tagName string) error {
	ret := _m.Called(label, tagName)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string) error); ok {
		r0 = rf(label, tagName)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *Volume) UntagSnapshot(tagName string) (string, error) {
	ret := _m.Called(tagName)

	var r0 string
	if rf, ok := ret.Get(0).(func(string) string); ok {
		r0 = rf(tagName)
	} else {
		r0 = ret.Get(0).(string)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(tagName)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *Volume) GetSnapshotWithTag(tagName string) (*volume.SnapshotInfo, error) {
	ret := _m.Called(tagName)

	var r0 *volume.SnapshotInfo
	if rf, ok := ret.Get(0).(func(string) *volume.SnapshotInfo); ok {
		r0 = rf(tagName)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*volume.SnapshotInfo)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(tagName)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *Volume) Export(label string, parent string, writer io.Writer) error {
	ret := _m.Called(label, parent, writer)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string, io.Writer) error); ok {
		r0 = rf(label, parent, writer)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *Volume) Import(label string, reader io.Reader) error {
	ret := _m.Called(label, reader)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, io.Reader) error); ok {
		r0 = rf(label, reader)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *Volume) Tenant() string {
	ret := _m.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}
