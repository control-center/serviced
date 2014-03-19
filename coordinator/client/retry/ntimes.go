package retry

import (
	"time"
)

type nTimes struct {
	n                   int // number of time to retry
	sleepBetweenRetries time.Duration
}

// NTimes returns a retry policy that retries up to n times.
func NTimes(n int, sleepBetweenRetries time.Duration) Policy {
	return nTimes{
		n:                   n,
		sleepBetweenRetries: sleepBetweenRetries,
	}
}

func (n nTimes) Name() string {
	return "NTimes"
}

func (n nTimes) AllowRetry(retryCount int, elapsed time.Duration) (bool, time.Duration) {
	if retryCount < n.n {
		return true, n.sleepBetweenRetries
	}
	return false, time.Duration(0)
}
