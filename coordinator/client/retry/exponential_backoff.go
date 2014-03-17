package retry

import (
	"math/rand"
	"time"
)

type exponentialBackoff struct {
	baseSleepTime time.Duration // amount of time
	maxRetries    int           // time between retries
	done          chan struct{}
}

// ExponentialBackoff returns a policy that will retry up to maxRetries with an exponentially increasing
// sleep time.
func ExponentialBackoff(baseSleepTime time.Duration, maxRetries int) Policy {
	return exponentialBackoff{
		baseSleepTime: baseSleepTime,
		maxRetries:    maxRetries,
		done:          make(chan struct{}),
	}
}

func (u exponentialBackoff) Name() string {
	return "ExponentialBackoff"
}

func (u exponentialBackoff) Close() {
	// attempt to signal AllowRetry sleep to shutdown.
	select {
	case u.done <- struct{}{}:
	default:
	}
}

func (u exponentialBackoff) getSleepTime(retryCount int) time.Duration {
	sleep := int(rand.Int31n(1<<uint(retryCount) + 1))
	if sleep < 1 {
		sleep = 1
	}
	sleepTime := u.baseSleepTime * time.Duration(sleep)
	return sleepTime
}

func (u exponentialBackoff) AllowRetry(retryCount int, elapsed time.Duration) bool {

	if retryCount < u.maxRetries {
		select {
		case <-time.After(u.getSleepTime(retryCount)):
			return true
		case <-u.done:
		}
	}
	return false
}
