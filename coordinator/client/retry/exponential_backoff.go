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
	"math/rand"
	"time"
)

type exponentialBackoff struct {
	baseSleepTime time.Duration // amount of time
	maxRetries    int           // time between retries
}

// ExponentialBackoff returns a policy that will retry up to maxRetries with an exponentially increasing
// sleep time.
func ExponentialBackoff(baseSleepTime time.Duration, maxRetries int) Policy {
	return exponentialBackoff{
		baseSleepTime: baseSleepTime,
		maxRetries:    maxRetries,
	}
}

func (u exponentialBackoff) Name() string {
	return "ExponentialBackoff"
}

func (u exponentialBackoff) getSleepTime(retryCount int) time.Duration {
	sleep := int(rand.Int31n(1<<uint(retryCount) + 1))
	if sleep < 1 {
		sleep = 1
	}
	sleepTime := u.baseSleepTime * time.Duration(sleep)
	return sleepTime
}

func (u exponentialBackoff) AllowRetry(retryCount int, elapsed time.Duration) (bool, time.Duration) {

	if retryCount < u.maxRetries {
		return true, u.getSleepTime(retryCount)
	}
	return false, time.Duration(0)
}
