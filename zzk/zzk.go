package zzk

import (
	"errors"
	"path"
	"sync"

	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/coordinator/client"
)

var (
	ErrShutdown = errors.New("listener shutdown")
)

// Listener is zookeeper node listener type
type Listener interface {
	GetConnection() client.Connection
	GetPath(nodes ...string) string
	Ready() error
	Done()
	Spawn(<-chan interface{}, string)
}

// PathExists verifies if a path exists and does not raise an exception if the
// path does not exist
func PathExists(conn client.Connection, p string) (bool, error) {
	exists, err := conn.Exists(p)
	if err == client.ErrNoNode {
		return false, nil
	}
	return exists, err
}

// Ready waits for a node to be available for watching
func Ready(shutdown <-chan interface{}, conn client.Connection, p string) error {
	for {
		if exists, err := PathExists(conn, p); err != nil {
			return err
		} else if exists {
			return nil
		} else if err := Ready(shutdown, conn, path.Dir(p)); err != nil {
			return err
		}

		_, event, err := conn.ChildrenW(path.Dir(p))
		if err != nil {
			return err
		}
		select {
		case <-event:
			// pass
		case <-shutdown:
			return ErrShutdown
		}
	}
}

// Listen initializes a listener for a particular zookeeper node
func Listen(shutdown <-chan interface{}, l Listener) {
	var (
		_shutdown  = make(chan interface{})
		done       = make(chan string)
		processing = make(map[string]interface{})
		conn       = l.GetConnection()
	)

	if err := Ready(shutdown, conn, l.GetPath()); err != nil {
		glog.Errorf("Could not start listener at %s: %s", l.GetPath(), err)
		return
	} else if err := l.Ready(); err != nil {
		glog.Errorf("Could not start listener at %s: %s", l.GetPath(), err)
		return
	}

	defer func() {
		glog.Infof("Listener at %s receieved interrupt", l.GetPath())
		close(_shutdown)
		for len(processing) > 0 {
			delete(processing, <-done)
		}
		l.Done()
	}()

	for {
		nodes, event, err := conn.ChildrenW(l.GetPath())
		if err != nil {
			glog.Errorf("Could not watch for nodes at %s: %s", l.GetPath(), err)
			return
		}

		for _, node := range nodes {
			if _, ok := processing[node]; !ok {
				glog.V(1).Infof("Spawning a listener for %s", l.GetPath(node))
				processing[node] = nil
				go func(node string) {
					defer func() {
						glog.V(1).Infof("Listener at %s was shutdown", l.GetPath(node))
						done <- node
					}()
					l.Spawn(_shutdown, node)
				}(node)
			}
		}

		select {
		case e := <-event:
			if e.Type == client.EventNodeDeleted {
				glog.V(1).Infof("Node %s has been removed; shutting down listener", l.GetPath())
				return
			}
			glog.V(4).Infof("Node %s receieved event %v", l.GetPath(), e)
		case node := <-done:
			glog.V(3).Infof("Cleaning up %s", l.GetPath(node))
			delete(processing, node)
		case <-shutdown:
			return
		}
	}
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
			Listen(_shutdown, listener)
		}()
	}
	Listen(shutdown, master)
	close(_shutdown)
	wg.Wait()
}