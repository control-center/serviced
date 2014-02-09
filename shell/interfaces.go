// Shell package.
package shell

import (
	"github.com/googollee/go-socket.io"
)

// Describes whether a process terminated normally or abnormally
type Termination int

const (
	NORMAL   Termination = iota // Process terminated normally
	ABNORMAL                    // Process terminated abnormally
)

type ProcessServer struct {
	sio   *socketio.SocketIOServer
	actor ProcessActor
}

type ProcessConfig struct {
	ServiceId string
	IsTTY     bool
	Envv      []string
	Command   string
}

type Result struct {
	ExitCode    int
	Termination Termination
}

type ProcessInstance struct {
	stdin  chan string
	stdout chan string
	stderr chan string
	signal chan int
	result chan Result
}

type ProcessActor interface {
	Exec(*ProcessConfig) *ProcessInstance
}

type Forwarder struct {
	addr string
}

type Executor struct {
	port string
}
