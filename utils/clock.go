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

package utils

import "time"

var (
	RealClock Clock
)

func init() {
	RealClock = realClock{}
}

// Clock is an abstraction of a clock, for testing purposes
type Clock interface {
	After(d time.Duration) <-chan time.Time
}

type realClock struct{}

func (realClock) After(d time.Duration) <-chan time.Time {
	return time.After(d)
}

type TestClock struct {
	AfterChan chan time.Time
}

func (c TestClock) After(d time.Duration) <-chan time.Time {
	return c.AfterChan
}

func (c TestClock) Fire() {
	c.AfterChan <- time.Now()
}

func NewTestClock() TestClock {
	return TestClock{
		AfterChan: make(chan time.Time),
	}
}
