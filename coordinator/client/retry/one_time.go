package retry

import (
	"time"
)

type oneTime struct {
	sleepBetweenRetry time.Duration
	done              chan struct{}
}

// OneTime returns a policy that retries only once.
func OneTime(sleepBetweenRetry time.Duration) Policy {
	return oneTime{
		sleepBetweenRetry: sleepBetweenRetry,
		done:              make(chan struct{}),
	}
}

func (u oneTime) Name() string {
	return "OneTime"
}

// AllowRetry returns true if the retry count is 0.
func (u oneTime) AllowRetry(retryCount int, elapsed time.Duration) (bool, time.Duration) {
	if retryCount == 0 {
		return true, u.sleepBetweenRetry
	}
	return false, time.Duration(0)
}
