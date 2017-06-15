// Copyright 2017 The Serviced Authors.
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

package client

import (
	"math/rand"
	"time"

	zklib "github.com/control-center/go-zookeeper/zk"
)

// Verify the interface
var _ zklib.Backoff = &Backoff{}

// Backoff controls the exponential backoff used when connection attempts to all zookeepers fail
type Backoff struct {
	InitialDelay time.Duration	// the initial delay
	MaxDelay     time.Duration	// The maximum delay
	delay        time.Duration	// the current delay
	random       *rand.Rand

}

// GetDelay returns the amount of delay that should be used for the current connection attempt.
// It will return a randomized value of initialDelay on the first call, and will increase the delay
// randomly on each subsequent call up to maxDelay. The initial delay and each subsequent delay
// are randomized to avoid a scenario where multiple instances on the same host all start trying
// to reconnection. In scenarios like those, we don't want all instances reconnecting in lock-step
// with each other.
func (backoff *Backoff) GetDelay() time.Duration {
	defer func() {
		factor := 2.0
		jitter := 6.0

		backoff.delay = time.Duration(float64(backoff.delay) * factor)
		backoff.delay += time.Duration(backoff.random.Float64() * jitter * float64(time.Second))
		if backoff.delay > backoff.MaxDelay {
			backoff.delay = backoff.MaxDelay
		}
	}()

	if backoff.random == nil {
		backoff.initialize()
	}
	if backoff.delay == 0 {
		backoff.Reset()
	}
	plog.WithField("delay", backoff.delay).Debug("Returned zk backoff interval")
	return backoff.delay
}

// Reset resets the backoff delay to some random value that is btwn 80-120% of the initialDelay.
//     We want to randomize the initial delay so in cases where many instances simultaneously
//     lose all ZK connections, they will not all start trying to reconnect at the same time.
func (backoff *Backoff) Reset() {
	start := backoff.InitialDelay.Seconds()
	minStart := 0.8 * start
	maxStart := 1.2 * start

	if backoff.random == nil {
		backoff.initialize()
	}

	// compute a random value between min and max start
	rando := backoff.random.Float64()
	start = minStart + (rando * (maxStart - minStart))
	backoff.delay = time.Duration(start * float64(time.Second))

	// never exceed maxDelay
	if backoff.delay > backoff.MaxDelay {
		backoff.delay = backoff.MaxDelay
	}
}

func (backoff *Backoff) initialize() {
	backoff.random = rand.New(rand.NewSource(time.Now().UnixNano()))
}
