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

type once struct {
	sleepBetweenRetry time.Duration
	done              chan struct{}
}

// Once returns a policy that retries only once.
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
