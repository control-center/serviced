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

// HostRegistryListener monitors the availability of hosts within a pool
// by watching for children within the path
// /pools/POOLID/hosts/HOSTID/online
type HostRegistryListener struct {
	conn     client.Connection
	poolid   string
	isOnline chan struct{}
}

// NewHostRegistryListener instantiates a new host registry listener
func NewHostRegistryListener(poolid string) *HostRegistryListener {
	return &HostRegistryListener{
		poolid:   poolid,
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

	// set up the connection timeout timer and track outage times.
	isOnline := false
	outage := time.Now()

	firstTimeout := true
	offlineTimer := time.NewTimer(h.getTimeout())
	defer offlineTimer.Stop()
	onlineTimer := time.NewTimer(0)
	defer onlineTimer.Stop()

	// set up cancellable on coordinator events
	stop := make(chan struct{})
	defer func() { close(stop) }()

	for {
		// does the host exist?
		isAvailable, availEv, err := h.conn.ExistsW(h.GetPath(hostid), stop)
		if !isAvailable {
			if err != nil {
				glog.Errorf("Could not find host %s in pool %s: %s", hostid, h.poolid, err)
			} else {
				glog.V(2).Infof("Could not find host %s in pool %s, shutting down", hostid, h.poolid)
				if count := removeInstancesOnHost(h.conn, h.poolid, hostid); count > 0 {
					glog.Warningf("Reported shutdown of host %s in pool %s and cleaned up %d orphaned nodes", hostid, h.poolid, count)
				}
			}
			return
		}

		// check to see if the host is up
		var ch []string
		onlinepth := h.GetPath(hostid, "online")
		isAvailable, onlineEv, err := h.conn.ExistsW(onlinepth, stop)
		if err != nil {
			glog.Errorf("Could not check online status of host %s in pool %s: %s", hostid, h.poolid, err)
			return
		}
		if isAvailable {
			ch, onlineEv, err = h.conn.ChildrenW(onlinepth, stop)
			if err != nil {
				glog.Errorf("Could not check online status of host %s in pool %s: %s", hostid, h.poolid, err)
				return
			}
			isAvailable = len(ch) > 0
		} else {
			// host has performed a proper shutdown, ensure all nodes are
			// cleaned up
			if count := removeInstancesOnHost(h.conn, h.poolid, hostid); count > 0 {
				glog.Warningf("Reported shutdown of host %s in pool %s and cleaned up %d orphaned nodes", hostid, h.poolid, count)
			}
			glog.V(2).Infof("Host in pool %s has shut down", hostid, h.poolid)
		}

		// set the node's online status
		if !isAvailable && isOnline {

			// host is down, begin the countdown
			glog.V(2).Infof("Host %s in pool %s is not available", hostid, h.poolid)
			isOnline = false
			outage = time.Now()

			firstTimeout = true
			offlineTimer.Stop()
			offlineTimer = time.NewTimer(h.getTimeout())

		} else if isAvailable && !isOnline {

			// host is up, halt the countdown
			glog.V(0).Infof("Host %s in pool %s is online after %s", hostid, h.poolid, time.Since(outage))
			isOnline = true
		}

		// find out if the host can schedule services
		lockpth := h.GetPath(hostid, "locked")
		isLocked, lockev, err := h.conn.ExistsW(lockpth, stop)
		if err != nil {
			glog.Errorf("Could not check if host %s in pool %s can receive new services: %s", hostid, h.poolid, err)
			return
		}
		if isLocked {
			ch, lockev, err = h.conn.ChildrenW(lockpth, stop)
			if err != nil {
				glog.Errorf("Could not check if host %s in pool %s can receive new services: %s", hostid, h.poolid, err)
				return
			}
			isLocked = len(ch) > 0
		}

		// is the host running anything?
		ch, err = h.conn.Children(h.GetPath(hostid, "instances"))
		if err != nil && err != client.ErrNoNode {
			glog.Errorf("Could not track what instances are running on host %s in pool %s: %s", hostid, h.poolid, err)
			return
		}

		// clean up any incongruent states
		isRunning := false
		for _, stateid := range ch {
			hpth := h.GetPath(hostid, "instances", stateid)
			hdat := HostState{}
			if err := h.conn.Get(hpth, &hdat); err == client.ErrNoNode {
				continue
			} else if err != nil {
				glog.Errorf("Could not verify instance %s on host %s in pool %s: %s", stateid, hostid, h.poolid, err)
				return
			}

			spth := path.Join("/pools", h.poolid, "services", hdat.ServiceID, stateid)
			if ok, err := h.conn.Exists(spth); err != nil {
				glog.Errorf("Could not verify instance %s from service %s in pool %s: %s", stateid, hdat.ServiceID, h.poolid, err)
				return
			} else if !ok {
				if err := removeInstance(h.conn, h.poolid, hostid, hdat.ServiceID, stateid); err != nil {
					glog.Errorf("Could not remove incongruent instance %s on host %s in pool %s: %s", stateid, hostid, h.poolid, err)
					return
				}
				continue
			}
			isRunning = true
		}

		if isOnline {

			if !isLocked {

				// If the host is online, try to tell someone who cares.
				// Expectedly, this is not something that should be in high
				// demand.
				select {
				case h.isOnline <- struct{}{}:
				case <-lockev:
				case <-availEv:
				case <-onlineEv:
				case <-cancel:
					return
				}

			} else {

				// If the host is locked, then we cannot advertise scheduling
				// on this host, so rather we should wait until the lock is
				// freed.
				select {
				case <-lockev:
				case <-availEv:
				case <-onlineEv:
				case <-cancel:
					return
				}
			}

		} else if !isRunning {

			// I only care about an outage if I am running instances.  If I am
			// offline and not running instances, nothing will get scheduled to
			// me anyway.
			select {
			case <-availEv:
			case <-onlineEv:
			case <-cancel:
				return
			}

		} else {

			// If this is a network outage, not all hosts may appear offline at
			// the same time, so lets allow it to quiesce before trying to
			// reschedule.
			select {
			case <-offlineTimer.C:
				offlineTimer.Reset(0)

				// This may be a genuine host outage.  Alert when a host is
				// available.
				select {
				case <-h.isOnline:

					// Reset the online timer in case this is an outage and
					// we need to allow the system quiesce as it is coming back
					// online.
					if firstTimeout {
						onlineTimer.Stop()
						onlineTimer = time.NewTimer(h.getTimeout())
						firstTimeout = false
					}

					select {
					case <-onlineTimer.C:
						onlineTimer.Reset(0)

						// We have exceeded the wait timeout, so reschedule as
						// soon as possible.
						select {
						case <-h.isOnline:
							count := removeInstancesOnHost(h.conn, h.poolid, hostid)
							glog.Warningf("Unexpected outage of host %s in pool %s.  Cleaned up %d orphaned instances", hostid, h.poolid, count)
						case <-availEv:
						case <-onlineEv:
						case <-cancel:
							return
						}
					case <-availEv:
					case <-onlineEv:
					case <-cancel:
						return
					}
				case <-availEv:
				case <-onlineEv:
				case <-cancel:
					return
				}
			case <-availEv:
			case <-onlineEv:
			case <-cancel:
				return
			}
		}

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
	return p.GetConnectionTimeout()
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

			// only return hosts that are not locked
			ch, err := h.conn.Children(h.GetPath(hostid, "locked"))
			if err != nil && err != client.ErrNoNode {
				return nil, err
			}
			if len(ch) == 0 {
				hdat := host.Host{}
				if err := h.conn.Get(h.GetPath(hostid), &HostNode{Host: &hdat}); err == client.ErrNoNode {
					continue
				} else if err != nil {
					return nil, err
				}

				hosts = append(hosts, hdat)
			}
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
	basepth := "/"
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
func RegisterHost(cancel <-chan interface{}, conn client.Connection, hostid string) error {

	pth := path.Join("/hosts", hostid, "online")

	// clean up ephemeral nodes on exit
	defer func() {
		ch, _ := conn.Children(pth)
		for _, n := range ch {
			conn.Delete(path.Join(pth, n))
		}
	}()

	// set up cancellable on event watcher
	stop := make(chan struct{})
	defer func() { close(stop) }()
	for {

		// monitor the parent node
		regok, regev, err := conn.ExistsW(path.Dir(pth), stop)
		if err != nil {
			glog.Errorf("Could not verify whether host %s is registered: %s", hostid, err)
			return err
		} else if !regok {
			glog.Warningf("Host %s is not registered; system is idle", hostid)
			select {
			case <-regev:
			case <-cancel:
				return nil
			}
			close(stop)
			stop = make(chan struct{})
			continue
		}

		// the host goes online
		if err := conn.CreateIfExists(pth, &client.Dir{}); err == client.ErrNoNode {
			glog.Warningf("Host %s is not registered; system is idle", hostid)
			select {
			case <-regev:
			case <-cancel:
				return nil
			}
			close(stop)
			stop = make(chan struct{})
			continue
		} else if err != nil && err != client.ErrNodeExists {
			glog.Errorf("Could not verify if host %s is set online: %s", hostid, err)
			return err
		}

		// the host becomes active
		ch, ev, err := conn.ChildrenW(pth, stop)
		if err == client.ErrNoNode {
			glog.Warningf("Host %s is not active; system is idle", hostid)
			select {
			case <-regev:
			case <-cancel:
				return nil
			}
			close(stop)
			stop = make(chan struct{})
			continue
		} else if err != nil {
			glog.Errorf("Could not verify if host %s is set as active: %s", hostid, err)
			return err
		}

		// register the host if it isn't showing up as active
		if len(ch) == 0 {
			// Need to give the ephemeral a node name, despite the name
			// changing when it is written to the coordinator.
			_, err = conn.CreateEphemeralIfExists(path.Join(pth, hostid), &client.Dir{})
			if err != nil {
				glog.Errorf("Could not register host %s as active: %s", hostid, err)
				return err
			}
		}

		select {
		case <-regev:
		case <-ev:
		case <-cancel:
			return nil
		}
		close(stop)
		stop = make(chan struct{})
	}
}
