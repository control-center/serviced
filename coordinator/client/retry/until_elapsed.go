package retry

import (
	"time"
)

type untilElapsed struct {
	maxElapsed          time.Duration // amount of time
	sleepBetweenRetries time.Duration // time between retries
	done                chan struct{}
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

func (u untilElapsed) Close() {
	select {
	case u.done <- struct{}{}:
	default:
	}
}

func (u untilElapsed) AllowRetry(retryCount int, elapsed time.Duration) bool {
	if elapsed < u.maxElapsed {
		select {
		case <-time.After(u.sleepBetweenRetries):
			return true
		case <-u.done:
			return false
		}
	}
	return false
}
