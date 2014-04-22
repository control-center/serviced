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
	SaveAs    string
	Envv      []string
	Command   string
}

type Result struct {
	ExitCode    int
	Error       string
	Termination Termination
}

type ProcessInstance struct {
	disconnected bool
	closed       bool

	Stdin  chan byte
	Stdout chan byte
	Stderr chan byte
	Signal chan int
	Result chan Result
}

type ProcessActor interface {
	Exec(*ProcessConfig) *ProcessInstance
	onDisconnect(*socketio.NameSpace)
}

type Forwarder struct {
	addr string
}

type Executor struct {
	port string
}
