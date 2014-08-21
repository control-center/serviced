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

type nTimes struct {
	n                   int // number of time to retry
	sleepBetweenRetries time.Duration
}

// NTimes returns a retry policy that retries up to n times.
func NTimes(n int, sleepBetweenRetries time.Duration) Policy {
	return nTimes{
		n:                   n,
		sleepBetweenRetries: sleepBetweenRetries,
	}
}

func (n nTimes) Name() string {
	return "NTimes"
}

func (n nTimes) AllowRetry(retryCount int, elapsed time.Duration) (bool, time.Duration) {
	if retryCount < n.n {
		return true, n.sleepBetweenRetries
	}
	return false, time.Duration(0)
}
