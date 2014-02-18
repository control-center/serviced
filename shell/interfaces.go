// Shell package.
package shell

import (
	"io"
	"syscall"

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
	Error       error
	Termination Termination
}

type ProcessInstance struct {
	Stdin  chan string
	Stdout chan string
	Stderr chan string
	Signal chan int
	Result chan Result
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

type Runner interface {
	Reader(size_t int) error
	Write(data []byte) (int, error)
	StdoutPipe() chan string
	StderrPipe() chan string
	Signal(signal syscall.Signal) error
	Kill() error
}

type OSRunner interface {
	StdinPipe() (io.WriteCloser, error)
	StdoutPipe() (io.ReadCloser, error)
	StderrPipe() (io.ReadCloser, error)
	Start() error
	Wait() error
	Signal(s syscall.Signal) error
	Kill() error
}

type Command struct {
	cmd    OSRunner
	stdin  io.WriteCloser
	stdout io.ReadCloser
	stderr io.ReadCloser

	stdoutChan chan string
	stderrChan chan string
}
