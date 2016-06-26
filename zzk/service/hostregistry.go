// Copyright 2016 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package service

import (
	"path"
	"time"

	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/domain/host"
	"github.com/zenoss/glog"
)

// HostRegistryListener is a monitors the availability of hosts within a pool
// by watching for children within the path
// /pools/POOLID/hosts/HOSTID/online
type HostRegistryListener struct {
	conn     client.Connection
	poolid   string
	qtime    time.Duration
	isOnline chan struct{}
}

// NewHostRegistryListener instantiates a new host registry listener
func NewHostRegistryListener(poolid string, qtime time.Duration) *HostRegistryListener {
	return &HostRegistryListener{
		poolid:   poolid,
		qtime:    qtime,
		isOnline: make(chan struct{}),
	}
}

func (h *HostRegistryListener) SetConnection(conn client.Connection) {
	h.conn = conn
}

func (h *HostRegistryListener) GetPath(nodes ...string) string {
	base := append([]string{"/pools", h.poolid, "hosts"}, nodes...)
	return path.Join(base...)
}

func (h *HostRegistryListener) Ready() error {
	return nil
}

func (h *HostRegistryListener) Done() {
}

func (h *HostRegistryListener) PostProcess(p map[string]struct{}) {
}

func (h *HostRegistryListener) Spawn(cancel <-chan interface{}, hostid string) {

	// get the path of the node that tracks the connectivity of the host
	pth, err := h.waitOnline(cancel, hostid)
	if err != nil {
		glog.Errorf("Could not wait for host %s in pool %s to be online: %s", hostid, h.poolid, err)
		return
	} else if pth == "" {

		// cancel has been triggered, listener is shutting down
		return
	}

	// set up the connection timeout timer and track outage times.
	t := time.NewTimer(h.getTimeout())
	defer t.Stop()
	outage := time.Now()
	isOnline := false

	// set up quiesce timeout timer
	qt := time.NewTimer(h.qtime)
	defer qt.Stop()
	timedOut := false

	// set up cancellable on coordinator events
	stop := make(chan struct{})
	defer func() { close(stop) }()

	for {

		// check to see if the host is up.
		ch, ev, err := h.conn.ChildrenW(pth, stop)
		if err == client.ErrNoNode {
			if count := removeInstancesOnHost(h.conn, h.poolid, hostid); count > 0 {
				glog.Warningf("Reported shutdown of host %s in pool %s and cleaned up %d orphaned nodes", hostid, h.poolid, count)
			}
			glog.V(2).Infof("Host %s in pool %s has shut down", hostid, h.poolid)
			return
		} else if err != nil {
			glog.Errorf("Could not check the online status of host %s in pool %s: %s", hostid, h.poolid, err)
			return
		}

		// change the node's online status
		if len(ch) == 0 && isOnline {

			// host is dead, begin the countdown
			glog.V(2).Infof("Host %s in pool %s is not available", hostid, h.poolid)
			t.Reset(h.getTimeout())
			outage = time.Now()
			isOnline = false
		} else if len(ch) > 0 && !isOnline {

			// host is up, halt the countdown
			glog.V(0).Infof("Host %s in pool %s is back online after %s", hostid, h.poolid, time.Since(outage))
			t.Stop()
			isOnline = true
			timedOut = false
		}

		// is the host running anything?
		ch, err = h.conn.Children(h.GetPath(hostid, "instances"))
		if err != nil && err != client.ErrNoNode {
			glog.Errorf("Could not track what instances are running on host %s in pool %s: %s", hostid, h.poolid, err)
			return
		}

		if isOnline {

			// If the host is online, try to tell someone who cares. Expectedly,
			// this is not something that should be in high demand.
			select {
			case h.isOnline <- struct{}{}:
			case <-ev:
			case <-cancel:
				return
			}

		} else if len(ch) == 0 {

			// I only care about an outage if I am running instances.  If I am
			// offline and not running instances, nothing will get scheduled to
			// me anyway.
			select {
			case <-ev:
			case <-cancel:
				return
			}

		} else if timedOut {

			// My initial reconnect timed out, so I will reschedule as soon as
			// anything is available.
			select {
			case <-h.isOnline:

				// Okay, I am not that harsh.  I will wait just a little longer
				// in case there was a network outage.
				qt.Reset(h.qtime)
				select {
				case <-qt.C:

					// Let's make sure we didn't get another outage while we
					// were waiting for the pool to quiesce.
					qt.Reset(h.qtime)
					select {
					case <-h.isOnline:

						// Well, I tried...
						count := removeInstancesOnHost(h.conn, h.poolid, hostid)
						glog.Warningf("Unexpected outage of host %s in pool %s. Cleaned up %d orphaned nodes", hostid, h.poolid, count)

					case <-qt.C:

						// The pool is still unstable so lets not be hasty
						// about rescheduling.

					case <-ev:

						// Looks like we came back after all.

					case <-cancel:
						return
					}

				case <-ev:
				case <-cancel:
					return
				}
			case <-ev:
			case <-cancel:
				return
			}
		} else {

			// This is an initial outage of a host running service instances.
			// Let's wait and see if the host comes back.
			select {
			case <-t.C:
				timedOut = true
			case <-ev:
			case <-cancel:
				return
			}
		}

		close(stop)
		stop = make(chan struct{})
	}
}

// waitOnline waits for the host to formally announce when it has gone online.
// This node will not exist if the host is not running and can exist if the
// host loses connectivity or fails to remove its online node. Returns the path
// to watch for connection losses.
func (h *HostRegistryListener) waitOnline(cancel <-chan interface{}, hostid string) (string, error) {
	pth := h.GetPath(hostid, "online")

	// set up a cancellable on the event watcher
	stop := make(chan struct{})
	defer func() { close(stop) }()

	// wait for the node to exist
	for {

		// set up the listener
		ok, ev, err := h.conn.ExistsW(pth, stop)
		if err != nil {
			glog.Errorf("Could not monitor host %s in pool %s: %s", hostid, h.poolid, err)
			return "", err
		}

		// the node is ready, so let's move on
		if ok {
			return pth, nil
		}

		// the node is not ready, so wait
		select {
		case <-ev:
		case <-cancel:

			// listener receieved signal to shutdown
			return "", nil
		}

		// cancel the listener and try again.
		close(stop)
		stop = make(chan struct{})
	}
}

// getTimeout returns the pool connection timeout.  Returns 0 if data cannot be
// acquired.
func (h *HostRegistryListener) getTimeout() time.Duration {
	var p PoolNode
	if err := h.conn.Get("/pools/"+h.poolid, &p); err != nil {
		glog.Warningf("Could not get pool connection timeout for %s: %s", h.poolid, err)
		return 0
	}
	return p.ConnectionTimeout
}

// GetRegisteredHosts returns a list of hosts that are active.  If there are
// zero active hosts, then it will wait until at least one host is available.
func (h *HostRegistryListener) GetRegisteredHosts(cancel <-chan interface{}) ([]host.Host, error) {
	hosts := []host.Host{}
	for {
		hostids, err := GetCurrentHosts(h.conn, h.poolid)
		if err != nil {
			return nil, err
		}

		for _, hostid := range hostids {
			hpth := path.Join("pools", h.poolid, "/hosts", hostid)
			hdat := host.Host{}
			if err := h.conn.Get(hpth, &HostNode{Host: &hdat}); err == client.ErrNoNode {
				continue
			}
			hosts = append(hosts, hdat)
		}
		if len(hosts) > 0 {
			return hosts, err
		}
		glog.Infof("No hosts reported as active in pool %s, waiting", h.poolid)
		select {
		case <-h.isOnline:
			glog.Infof("At least one host reported as active in pool %s, checking", h.poolid)
		case <-cancel:
			return nil, ErrShutdown
		}
	}
}

// GetCurrentHosts returns the list of hosts that are currently active.
func GetCurrentHosts(conn client.Connection, poolid string) ([]string, error) {
	onlineHosts := make([]string, 0)
	pth := path.Join("/pools", poolid, "hosts")
	ch, err := conn.Children(pth)
	if err != nil {
		return nil, err
	}
	for _, hostid := range ch {
		isOnline, err := IsHostOnline(conn, poolid, hostid)
		if err != nil {
			return nil, err
		}

		if isOnline {
			onlineHosts = append(onlineHosts, hostid)
		}
	}
	return onlineHosts, nil
}

// IsHostOnline returns true if a provided host is currently active.
func IsHostOnline(conn client.Connection, poolid, hostid string) (bool, error) {
	basepth := ""
	if poolid != "" {
		basepth = path.Join("/pools", poolid)
	}

	pth := path.Join(basepth, "/hosts", hostid, "online")
	ch, err := conn.Children(pth)
	if err != nil && err != client.ErrNoNode {
		return false, err
	}
	return len(ch) > 0, nil
}

// RegisterHost persists a registered host to the coordinator.  This is managed
// by the worker node, so it is expected that the connection will be pre-loaded
// with the path to the resource pool.
func RegisterHost(cancel <-chan struct{}, conn client.Connection, hostid string) error {

	// the host is online
	pth := path.Join("/hosts", hostid, "online")
	err := conn.CreateIfExists(pth, &client.Dir{})
	if err != nil && err != client.ErrNodeExists {
		glog.Errorf("Could not register host %s as active: %s", hostid, err)
		return err
	}

	// set up cancellable on the event watcher
	stop := make(chan struct{})
	defer func() { close(stop) }()
	for {

		// monitor the state of the host
		ch, ev, err := conn.ChildrenW(pth, stop)
		if err != nil {
			glog.Errorf("Could not verify the online status of host %s: %s", hostid, err)
			return err
		}

		// register the host if it isn't showing up as online
		if len(ch) == 0 {
			_, err = conn.CreateEphemeralIfExists(pth, &client.Dir{})
			if err != nil {
				glog.Errorf("Could not register host %s as active: %s", hostid, err)
				return err
			}
		}

		select {
		case <-ev:
		case <-cancel:
			glog.V(2).Infof("Host %s is shutting down", hostid)
			conn.Delete(pth)
			return nil
		}
		close(stop)
		stop = make(chan struct{})
	}
}
