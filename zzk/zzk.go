// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package zzk

import (
	"sync"

	"github.com/zenoss/glog"
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
	glog.Infof("shutdown finished for %#v", master)
	close(_shutdown)
	wg.Wait()
	glog.Info("all listeners stopped")
}
