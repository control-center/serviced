package mocks

import "github.com/control-center/serviced/zzk/registry"
import "github.com/stretchr/testify/mock"

type VHostHandler struct {
	mock.Mock
}

func (_m *VHostHandler) Enable(name string) {
	_m.Called(name)
}
func (_m *VHostHandler) Disable(name string) {
	_m.Called(name)
}
func (_m *VHostHandler) Set(name string, exports []registry.ExportDetails) {
	_m.Called(name, exports)
}
