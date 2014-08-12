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

type untilElapsed struct {
	maxElapsed          time.Duration // amount of time
	sleepBetweenRetries time.Duration // time between retries
}

// UntilElapsed returns a policy that retries until a given amount of time elapses
func UntilElapsed(maxElapsed, sleepBetweenRetries time.Duration) Policy {
	return untilElapsed{
		maxElapsed:          maxElapsed,
		sleepBetweenRetries: sleepBetweenRetries,
	}
}

func (u untilElapsed) Name() string {
	return "UntilElapsed"
}

func (u untilElapsed) AllowRetry(retryCount int, elapsed time.Duration) (bool, time.Duration) {
	if elapsed < u.maxElapsed {
		return true, u.sleepBetweenRetries
	}
	return false, time.Duration(0)
}
