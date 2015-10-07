package mocks

import "github.com/stretchr/testify/mock"

import "github.com/control-center/serviced/domain/registry"

type RegistryIndex struct {
	mock.Mock
}

func (_m *RegistryIndex) FindImage(image string) (*registry.Image, error) {
	ret := _m.Called(image)

	var r0 *registry.Image
	if rf, ok := ret.Get(0).(func(string) *registry.Image); ok {
		r0 = rf(image)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*registry.Image)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(image)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *RegistryIndex) PushImage(image string, uuid string) error {
	ret := _m.Called(image, uuid)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string) error); ok {
		r0 = rf(image, uuid)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *RegistryIndex) RemoveImage(image string) error {
	ret := _m.Called(image)

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(image)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *RegistryIndex) SearchLibraryByTag(library string, tag string) ([]registry.Image, error) {
	ret := _m.Called(library, tag)

	var r0 []registry.Image
	if rf, ok := ret.Get(0).(func(string, string) []registry.Image); ok {
		r0 = rf(library, tag)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]registry.Image)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, string) error); ok {
		r1 = rf(library, tag)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
