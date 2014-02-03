package shell

import (
	"syscall"
)

type Runner interface {
	Reader(size_t int)
	Write(data []byte) (int, error)
	StdoutPipe() chan string
	StderrPipe() chan string
	ExitedPipe() chan bool
	Error() error
	Signal(signal syscall.Signal) error
	Kill() error
	Close()
}
