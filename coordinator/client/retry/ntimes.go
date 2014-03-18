package retry

import (
	"time"
)

type nTimes struct {
	n                   int // number of time to retry
	sleepBetweenRetries time.Duration
	done                chan struct{}
}

// NTimes returns a retry policy that retries up to n times.
func NTimes(n int, sleepBetweenRetries time.Duration) Policy {
	return nTimes{
		n:                   n,
		sleepBetweenRetries: sleepBetweenRetries,
		done:                make(chan struct{}),
	}
}

func (n nTimes) Name() string {
	return "NTimes"
}

func (n nTimes) Close() {
	select {
	case n.done <- struct{}{}:
	default:
	}
}

func (n nTimes) AllowRetry(retryCount int, elapsed time.Duration) bool {
	if retryCount < n.n {
		select {
		case <-time.After(n.sleepBetweenRetries):
			return true
		case <-n.done:
			return false
		}
	}
	return false
}
