package zzk

import (
	"sync"
)

type Listener interface {
	Listen(shutdown <-chan interface{})
}

// Start starts a group of listeners that are governed by a master listener.
// When the master exits, it shuts down all of the child listeners and waits
// for all of the subprocesses to exit
func Start(shutdown <-chan interface{}, master Listener, listeners ...Listener) {
	var wg sync.WaitGroup
	_shutdown := make(chan interface{})
	for _, listener := range listeners {
		wg.Add(1)
		go func() {
			defer wg.Done()
			listener.Listen(_shutdown)
		}()
	}
	master.Listen(shutdown)
	close(_shutdown)
	wg.Wait()
}