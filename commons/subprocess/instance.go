package proxy

import (
	"errors"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

// Instance manages a
type Instance struct {
	command        string
	args           []string
	commandExit    chan error
	closing        chan chan error
	closeLock      sync.Mutex    // mutex to synchronize Close() calls
	restart        time.Duration // amount of time between restarts of subprocess. 0 indicates do not restart.
	sigtermTimeout time.Duration // sigterm timeout
	restarts       int           //number of restarts
}

func New(restart, sigtermTimeout time.Duration, command string, args ...string) (*Instance, error) {
	s := &Instance{
		command:        command,
		args:           args,
		commandExit:    make(chan error),
		restart:        restart,
		sigtermTimeout: sigtermTimeout,
	}
	go s.loop()
	return s, nil
}

// Close() signals the subprocess to shutdown via sigterm. If sigterm fails to shutdown
// withing the s.timeout, a sigkill is issued.
func (s *Instance) Close() error {
	s.closeLock.Lock()
	defer s.closeLock.Unlock()
	if s.closing == nil {
		return errors.New("subprocess instance already closed")
	}
	errc := make(chan error)
	s.closing <- errc
	err := <-errc
	s.closing = nil
	return err
}

func (s *Instance) loop() {

	var restart <-chan time.Time

	setUpCmd := func(exitChan chan error) *exec.Cmd {
		cmd := exec.Command(s.command, s.args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		go func() {
			exitChan <- cmd.Run()
		}()
		return cmd
	}
	cmd := setUpCmd(s.commandExit)
	var returnChan chan error
	sigterm := make(chan error)
	sigkill := make(<-chan time.Time)

	closing := s.closing
	for {

		select {
		case <-restart:
			cmd = setUpCmd(s.commandExit)
			s.restarts = s.restarts + 1

		case <-s.commandExit:
			if s.restart > 0 {
				restart = time.After(s.restart)
			}

		case returnChan = <-closing:
			cmd.Process.Signal(syscall.SIGQUIT)
			// setup hard killing the task
			sigkill = time.After(s.sigtermTimeout)
			closing = nil // avoid this case again
			go func(rchan chan error) {
				rchan <- cmd.Wait()
			}(sigterm)

		case err := <-sigterm:
			returnChan <- err
			return

		case <-sigkill:
			cmd.Process.Signal(syscall.SIGKILL)
			// our previous goroutine should take care of doing the right thing
			returnChan <- errors.New("subprocess instance sigkilled")
			return
		}

	}
}
