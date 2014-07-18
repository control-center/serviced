// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

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
