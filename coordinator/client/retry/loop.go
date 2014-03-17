
package retry

import (
	"time"
)

type Loop struct {
	isDone bool
	startTime time.Duration
	retryCount int
	retryPolicy Policy
	callable func() error
	done chan struct{}
}

func (loop *Loop) Close() {
	select {
	case loop.done <- struct{}{}:
	default:
	}
}

func (loop *Loop) ShouldContinue() bool {
	return !loop.isDone
}


func (loop *Loop) MarkComplete() {
	loop.isDone = true
}



