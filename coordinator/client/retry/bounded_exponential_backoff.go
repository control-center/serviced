package retry

import (
	"math/rand"
	"time"
)

type boundedExponentialBackoff struct {
	baseSleepTime time.Duration // amount of time
	maxSleepTime  time.Duration
	maxRetries    int // time between retries
	done          chan struct{}
}

// ExponentialBackoff returns a policy that will retry up to maxRetries with an exponentially increasing
// sleep time up to maxSleepTime.
func BoundedExponentialBackoff(baseSleepTime time.Duration, maxSleepTime time.Duration, maxRetries int) Policy {
	return boundedExponentialBackoff{
		baseSleepTime: baseSleepTime,
		maxSleepTime:  maxSleepTime,
		maxRetries:    maxRetries,
		done:          make(chan struct{}),
	}
}

func (u boundedExponentialBackoff) Name() string {
	return "BoundedExponentialBackoff"
}

func (u boundedExponentialBackoff) Close() {
	// attempt to signal AllowRetry sleep to shutdown.
	select {
	case u.done <- struct{}{}:
	default:
	}
}

func (u boundedExponentialBackoff) getSleepTime(retryCount int) time.Duration {
	sleep := int(rand.Int31n(1<<uint(retryCount) + 1))
	if sleep < 1 {
		sleep = 1
	}
	sleepTime := u.baseSleepTime * time.Duration(sleep)
	if sleepTime > u.maxSleepTime {
		return u.maxSleepTime
	}
	return sleepTime
}

func (u boundedExponentialBackoff) AllowRetry(retryCount int, elapsed time.Duration) bool {

	if retryCount < u.maxRetries {
		select {
		case <-time.After(u.getSleepTime(retryCount)):
			return true
		case <-u.done:
		}
	}
	return false
}
