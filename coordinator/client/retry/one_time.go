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
func (u oneTime) AllowRetry(retryCount int, elapsed time.Duration) bool {
	if retryCount == 0 {
		select {
		case <-time.After(u.sleepBetweenRetry):
			return true
		case <-u.done:
		}
	}
	return false
}

// Close() interrupts the OneTime retry policy.
func (u oneTime) Close() {
	select {
	case u.done <- struct{}{}:
	default:
	}
}
