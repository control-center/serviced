package mocks

import "github.com/stretchr/testify/mock"

import "github.com/control-center/serviced/coordinator/client"

type Listener2 struct {
	mock.Mock
}

func (_m *Listener2) Listen(cancel <-chan interface{}, conn client.Connection) {
	_m.Called(cancel, conn)
}
func (_m *Listener2) Exited() {
	_m.Called()
}
