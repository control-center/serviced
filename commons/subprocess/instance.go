// Copyright 2014 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package subprocess

import (
	"github.com/zenoss/glog"

	"errors"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

// Instance manages a subprocess instance.
type Instance struct {
	command        string
	args           []string
	env            []string
	commandExit    chan error // used to send command exit values to the parent controller
	closing        chan chan error
	closeLock      sync.Mutex     // mutex to synchronize Close() calls
	sigtermTimeout time.Duration  // sigterm timeout
	signalChan     chan os.Signal // used to send signals to the command process
}

// New creates a subprocess.Instance
func New(sigtermTimeout time.Duration, env []string, command string, args ...string) (*Instance, chan error, error) {
	s := &Instance{
		command:        command,
		args:           args,
		env:            env,
		commandExit:    make(chan error, 1),
		sigtermTimeout: sigtermTimeout,
		signalChan:     make(chan os.Signal, 1),
	}
	go s.loop()
	return s, s.commandExit, nil
}

// Notify sends the sig to the subprocess instance.
func (s *Instance) Notify(sig os.Signal) {
	glog.V(1).Infof("Notify: sending signal %v", sig)
	select {
	case s.signalChan <- sig:
		glog.V(1).Infof("Notify: sent signal %v", sig)
	default:
		// This may happen if we're trying to kill a process that has already died. The controller will first send
		// SIGTERM, then it will try SIGKILL
		glog.Warningf("Notify: unable to send signal %v because channel is full", sig)
	}
}

// Close signals the subprocess to shutdown via sigterm. If sigterm fails to shutdown
// withing the s.timeout, a sigkill is issued.
func (s *Instance) Close() error {
	if s == nil {
		return nil
	}
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

	setUpCmd := func(exitChan chan error) *exec.Cmd {
		glog.Infof("about to execute: %s , %v[%d]", s.command, s.args, len(s.args))
		cmd := exec.Command(s.command, s.args...)
		cmd.Env = s.env
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		go func() {
			exitChan <- cmd.Run()
		}()
		return cmd
	}

	processExit := make(chan error, 1) //lets us know when the process exits
	cmd := setUpCmd(processExit)
	var returnChan chan error
	sigterm := make(chan error)
	sigkill := make(<-chan time.Time)

	closing := s.closing
	for {

		select {
		case s := <-s.signalChan:
			glog.V(1).Infof("loop: sending signal %v", s)
			cmd.Process.Signal(s)
			glog.V(1).Infof("loop: sent signal %v", s)

		case exitError := <-processExit:
			glog.V(1).Infof("loop: process exited with error %v", exitError)
			select {
			case s.commandExit <- exitError: // tell our the parent controller that the command has exited
			}
			return

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
