// Copyright 2017 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package dfs

import (
	"time"

	"github.com/control-center/serviced/logging"
)

var (
	plog = logging.PackageLogger()
)

type Clock interface {
	Now() time.Time
	Since(t time.Time) time.Duration
}

type DefaultClock struct{}

func (c *DefaultClock) Now() time.Time { return time.Now() }

func (c *DefaultClock) Since(t time.Time) time.Duration { return time.Since(t) }

type ProgressCounter struct {
	Total                 uint64
	Log                   func()
	clock                 Clock
	lastLogged            time.Time
	updateIntervalSeconds int
}

func (pc *ProgressCounter) Write(data []byte) (int, error) {
	pc.Total += uint64(len(data))

	if pc.Log != nil {
		timeSinceUpdate := int(pc.clock.Since(pc.lastLogged).Seconds())
		if timeSinceUpdate > pc.updateIntervalSeconds {
			pc.Log()
			pc.lastLogged = pc.clock.Now()
		}
	}

	return len(data), nil
}

func NewProgressCounter(interval int) *ProgressCounter {
	return NewProgressCounterWithClock(interval, &DefaultClock{})
}

func NewProgressCounterWithClock(interval int, clock Clock) *ProgressCounter {
	return &ProgressCounter{
		clock:                 clock,
		lastLogged:            clock.Now(),
		updateIntervalSeconds: interval,
	}
}
