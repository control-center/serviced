package mocks

import "github.com/stretchr/testify/mock"

import "github.com/control-center/serviced/coordinator/client"

type Spawner struct {
	mock.Mock
}

func (_m *Spawner) SetConn(conn client.Connection) {
	_m.Called(conn)
}
func (_m *Spawner) Path() string {
	ret := _m.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}
func (_m *Spawner) Pre() {
	_m.Called()
}
func (_m *Spawner) Spawn(cancel <-chan struct{}, n string) {
	_m.Called(cancel, n)
}
func (_m *Spawner) Post(p map[string]struct{}) {
	_m.Called(p)
}
