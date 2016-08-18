package mocks

import "github.com/control-center/serviced/zzk/registry2"
import "github.com/stretchr/testify/mock"

type PublicPortHandler struct {
	mock.Mock
}

func (_m *PublicPortHandler) Enable(port string, protocol string, useTLS bool) {
	_m.Called(port, protocol, useTLS)
}
func (_m *PublicPortHandler) Disable(port string) {
	_m.Called(port)
}
func (_m *PublicPortHandler) Set(port string, exports []registry.ExportDetails) {
	_m.Called(port, exports)
}
