// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.
package retry

import (
	"time"
)

type once struct {
	sleepBetweenRetry time.Duration
	done              chan struct{}
}

// OneTime returns a policy that retries only once.
func Once(sleepBetweenRetry time.Duration) Policy {
	return once{
		sleepBetweenRetry: sleepBetweenRetry,
		done:              make(chan struct{}),
	}
}

func (u once) Name() string {
	return "OneTime"
}

// AllowRetry returns true if the retry count is 0.
func (u once) AllowRetry(retryCount int, elapsed time.Duration) (bool, time.Duration) {
	if retryCount == 0 {
		return true, u.sleepBetweenRetry
	}
	return false, time.Duration(0)
}
