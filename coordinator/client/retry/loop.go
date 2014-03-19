package retry

import (
	"log"
	"time"
)

// Loop is an object that manages running the callable function and retrying it
// based on a give policy.
type Loop struct {
	isDone      bool
	startTime   time.Time
	retryCount  int
	retryPolicy Policy
	cancelable  func(chan chan error) chan error
	waiting     chan error
	closing     chan chan error
	done        bool
}

func NewLoop(policy Policy, cancelable func(chan chan error) chan error) Loop {
	loop := Loop{
		startTime:   time.Now(),
		retryPolicy: policy,
		cancelable:  cancelable,
		waiting:     make(chan error),
		closing:     make(chan chan error),
	}
	go loop.loop()
	return loop
}

func (loop *Loop) loop() {
	var err error
	cancelRequest := make(chan chan error)

	var loopSleep <-chan time.Time
	quit := false
	loopRun := loop.cancelable(cancelRequest)
	for {
		select {
		case err = <-loopRun:
			log.Printf("runnloop: %s", err)
			if err == nil {
				loopRun = nil
				go func() {
					loop.waiting <- nil
				}()
				continue
			}
			if quit {
				go func() {
					loop.waiting <- err
				}()
				return
			}
			loopRun = nil
			tryAgain, timeToSleep := loop.retryPolicy.AllowRetry(loop.retryCount, time.Since(loop.startTime))
			if !tryAgain {
				go func() {
					loop.waiting <- err
				}()
				return
			}
			loop.retryCount++
			loopSleep = time.After(timeToSleep)
		case <-loopSleep:
			loopRun = loop.cancelable(cancelRequest)
			loopSleep = nil
		case errc := <-loop.closing:
			quit = true
			if loopSleep != nil {
				go func() {
					loop.waiting <- err
				}()
				errc <- err
				return
			}
			cancelRequest <- errc
		}
	}
}

func (loop *Loop) Wait() error {
	return <-loop.waiting
}

func (loop *Loop) Close() error {
	errc := make(chan error)
	loop.closing <- errc
	return <-errc
}
