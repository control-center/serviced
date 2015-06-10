// Copyright 2015 The Serviced Authors.
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

package scheduler

import (
	"sync"
	"time"

	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/domain/host"
	"github.com/zenoss/glog"
)

// Manager is the child process that is run on a particular master
type Manager interface {
	// Run starts the manager
	Run(cancel <-chan interface{}) error
	// Leader provides info about the master that is running the manager
	Leader(conn client.Connection, host *host.Host) client.Leader
}

// Scheduler delegates managers across masters
type Scheduler struct {
	shutdown chan struct{}
	threads  sync.WaitGroup
	host     *host.Host
	managers []Manager
}

// StartScheduler starts the scheduler
func StartScheduler(conn client.Connection, host *host.Host, managers ...Manager) *Scheduler {
	scheduler := &Scheduler{
		shutdown: make(chan struct{}),
		threads:  sync.WaitGroup{},
		host:     host,
		managers: managers,
	}

	scheduler.start(conn)
	return scheduler
}

// Wait waits for all the scheduler threads to exit
func (s *Scheduler) Wait() <-chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		s.threads.Wait()
	}()
	return done
}

// Stop stops the scheduler
func (s *Scheduler) Stop() {
	close(s.shutdown)
	s.threads.Wait()
}

// start initializes the manager
func (s *Scheduler) start(conn client.Connection) {
	for i := range s.managers {
		s.threads.Add(1)
		go func(index int) {
			defer s.threads.Done()
			s.manage(conn, s.managers[index])
		}(i)
	}
}

// manage runs the manager
func (s *Scheduler) manage(conn client.Connection, manager Manager) {
	leader := manager.Leader(conn, s.host)

	for {
		done := make(chan interface{}, 2)

		// become the leader
		ready := make(chan error)
		go func() {
			ev, err := leader.TakeLead()

			// send ready signal or quit
			select {
			case ready <- err:
				if err != nil {
					return
				}
			case <-s.shutdown:
				leader.ReleaseLead()
				return
			}

			// wait for exit
			done <- <-ev
		}()

		// waiting for the leader to be ready
		cancel := make(chan interface{})
		select {
		case err := <-ready:
			if err != nil {
				glog.Errorf("Host %s could not become the leader (%#v): %s", s.host.ID, leader, err)
				return
			}
		case <-s.shutdown:
			// Did I shutdown before I became the leader?
			glog.Infof("Stopping manager for %s (%#v)", s.host.ID, leader)
			return
		}

		// start the manager
		go func() {
			if err := manager.Run(cancel); err != nil {
				glog.Warningf("Manager for host %s exiting: %s", s.host.ID, err)
				time.Sleep(5 * time.Second)
			}
			done <- struct{}{}
		}()

		// wait for something to happen
		select {
		case <-done:
			glog.Warningf("Host %s exited unexpectedly, reconnecting", s.host.ID)
			// shutdown the manager and its leader and wait to exit
			close(cancel)
			leader.ReleaseLead()
			<-done
		case <-s.shutdown:
			glog.Infof("Host %s receieved signal to shutdown", s.host.ID)
			close(cancel)
			leader.ReleaseLead()
			return
		}
	}
}
