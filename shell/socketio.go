package shell

import (
	"github.com/googollee/go-socket.io"
)

type SocketIOProcessStream struct {
	*baseProcessStream
	ns *socketio.NameSpace
}

func NewSocketIOProcessStream(addr string) *SocketIOProcessStream {
	return &SocketIOProcessStream{
		baseProcessStream: &baseProcessStream{addr: addr},
	}
}

func (s *SocketIOProcessStream) StreamClient() {
	for {
		select {
		case m := <-s.process.Stdout:
			s.ns.Emit("data", m)
		case m := <-s.process.Stderr:
			s.ns.Emit("data", m)
		case <-s.process.Exited:
			return
		}
	}
}

func onConnect(addr string) func(ns *socketio.NameSpace) {

	return func(ns *socketio.NameSpace) {
		servicename := "Zope"
		cmd := "/bin/bash"

		stream := NewSocketIOProcessStream(addr)
		stream.ns = ns
		stream.process = NewProcess(servicename, cmd, []string{}, true)

		ns.On("data", func(ns *socketio.NameSpace, data string) {
			stream.process.Stdin <- data
		})

		go stream.StreamClient()
		go stream.StreamAgent()
		stream.Wait()
	}

}

func NewSocketIOServer(addr string) *socketio.SocketIOServer {
	sio := socketio.NewSocketIOServer(&socketio.Config{})
	sio.On("connect", onConnect(addr))
	return sio
}
