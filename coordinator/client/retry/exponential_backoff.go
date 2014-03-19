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
