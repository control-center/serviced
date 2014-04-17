// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.
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
