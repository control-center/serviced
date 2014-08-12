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

package retry

import (
	"time"
)

// Loop is an object that manages running the callable function and retrying it
// based on a give policy.
type Loop struct {
	isDone      bool
	startTime   time.Time
	retryCount  int
	retryPolicy Policy
	cancelable  func(chan chan error) chan error
	waiting     chan error
	closing     chan chan error
	done        bool
}

// NewLoop creates a loop object that executes the cancelable function according to the
// given policy
func NewLoop(policy Policy, cancelable func(chan chan error) chan error) Loop {
	loop := Loop{
		startTime:   time.Now(),
		retryPolicy: policy,
		cancelable:  cancelable,
		waiting:     make(chan error, 1),
		closing:     make(chan chan error),
	}
	go loop.loop()
	return loop
}

func (loop *Loop) loop() {
	var err error
	cancelRequest := make(chan chan error)

	var loopSleep <-chan time.Time
	quit := false
	loopRun := loop.cancelable(cancelRequest)
	for {
		select {
		case err = <-loopRun:
			if err == nil {
				loopRun = nil
				loop.waiting <- nil
				continue
			}
			if quit {
				loop.waiting <- err
				return
			}
			loopRun = nil
			tryAgain, timeToSleep := loop.retryPolicy.AllowRetry(loop.retryCount, time.Since(loop.startTime))
			if !tryAgain {
				loop.waiting <- err
				return
			}
			loop.retryCount++
			loopSleep = time.After(timeToSleep)
		case <-loopSleep:
			loopRun = loop.cancelable(cancelRequest)
			loopSleep = nil
		case errc := <-loop.closing:
			quit = true
			if loopSleep != nil {
				loop.waiting <- err
				errc <- err
				return
			}
			cancelRequest <- errc
		}
	}
}

// Wait blocks until the loop exits
func (loop Loop) Wait() error {
	return <-loop.waiting
}

// Close stops the loop construct from attempting retries and notifies the running function to shutdown
func (loop Loop) Close() error {
	errc := make(chan error)
	loop.closing <- errc
	return <-errc
}
