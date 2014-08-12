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

type boundedExponentialBackoff struct {
	exponentialBackoff
	maxSleepTime time.Duration
}

// BoundedExponentialBackoff returns a policy that will retry up to maxRetries with an exponentially increasing
// sleep time up to maxSleepTime.
func BoundedExponentialBackoff(baseSleepTime time.Duration, maxSleepTime time.Duration, maxRetries int) Policy {
	return boundedExponentialBackoff{
		exponentialBackoff: exponentialBackoff{
			baseSleepTime: baseSleepTime,
			maxRetries:    maxRetries,
		},
		maxSleepTime: maxSleepTime,
	}
}

func (u boundedExponentialBackoff) Name() string {
	return "BoundedExponentialBackoff"
}

func (u boundedExponentialBackoff) AllowRetry(retryCount int, elapsed time.Duration) (bool, time.Duration) {

	retry, sleep := u.exponentialBackoff.AllowRetry(retryCount, elapsed)
	if sleep > u.maxSleepTime {
		sleep = u.maxSleepTime
	}
	return retry, sleep
}

func (u boundedExponentialBackoff) getSleepTime(retryCount int) time.Duration {
	sleepTime := u.baseSleepTime * u.exponentialBackoff.getSleepTime(retryCount)
	if sleepTime > u.maxSleepTime {
		return u.maxSleepTime
	}
	return sleepTime
}
