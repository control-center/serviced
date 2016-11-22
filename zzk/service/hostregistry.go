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

	log "github.com/Sirupsen/logrus"
	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/domain/service"
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
	logger := plog.WithFields(log.Fields{
		"poolid": h.poolid,
		"hostid": hostid,
	})

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
		// path: /pools/<poolid>/hosts/<hostid>
		isAvailable, availEv, err := h.conn.ExistsW(h.GetPath(hostid), stop)
		if err != nil {

			logger.WithError(err).Error("Could not look up host")
			return
		}
		if !isAvailable {

			logger.Debug("Host does not exist; stopping listener")
			return
		}

		// check to see if the host is up
		// path: /pools/<poolid>/hosts/<hostid>/online
		var ch []string
		onlinepth := h.GetPath(hostid, "online")
		isAvailable, onlineEv, err := h.conn.ExistsW(onlinepth, stop)
		if err != nil {

			logger.WithError(err).Error("Could not check online status of host")
			return
		}
		if isAvailable {
			// host is online, check the network availability
			ch, onlineEv, err = h.conn.ChildrenW(onlinepth, stop)
			if err != nil {

				logger.WithError(err).Error("Could not verify online status of host")
				return
			}
			isAvailable = len(ch) > 0
		} else {
			// host has shut down cleanly, ensure all nodes are cleaned up
			count := DeleteHostStates(h.conn, h.poolid, hostid)
			if count > 0 {
				logger.WithField("unscheduled", count).Warn("Host reported shutdown; cleaned up orphaned nodes")
			} else {
				logger.Debug("Host reported shutdown")
			}
		}

		// update the node's online status
		if !isAvailable && isOnline {

			// host is down, begin the countdown
			logger.Debug("Host is not available, starting network timeout")

			isOnline = false
			outage = time.Now()

			firstTimeout = true
			offlineTimer.Stop()
			offlineTimer = time.NewTimer(h.getTimeout())

		} else if isAvailable && !isOnline {

			// host is up, halt the countdown
			logger.WithField("outage", time.Since(outage)).Info("Host is online")
			isOnline = true
		}

		// find out if the host can receive new services
		// path: /pools/<poolid>/hosts/<hostid>/locked
		lockpth := h.GetPath(hostid, "locked")
		isLocked, lockev, err := h.conn.ExistsW(lockpth, stop)
		if err != nil {

			logger.WithError(err).Error("Could not check locked status of host")
			return
		}
		if isLocked {
			ch, lockev, err = h.conn.ChildrenW(lockpth, stop)
			if err != nil {

				logger.WithError(err).Error("Could not verify locked status of host")
				return
			}
			isLocked = len(ch) > 0
		}

		// clean up invalid states and find out if the host is running anything
		// path: /pools/<poolid>/hosts/<hostid>/instances
		if err := CleanHostStates(h.conn, h.poolid, hostid); err != nil {

			logger.WithError(err).Error("Could not clean states on host")
			return
		}
		ch, err = h.conn.Children(h.GetPath(hostid, "instances"))
		if err != nil && err != client.ErrNoNode {

			logger.WithError(err).Error("Could not look up instances on host")
			return
		}
		isRunning := len(ch) > 0

		eventLogger := plog.WithFields(log.Fields{
			"poolid":    h.poolid,
			"hostid":    hostid,
			"isonline":  isOnline,
			"islocked":  isLocked,
			"isrunning": isRunning,
		})
		eventLogger.Debug("Waiting for host event")

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
							// Only reschedule services without address
							// assignments.
							count := DeleteHostStatesWhen(h.conn, h.poolid, hostid, func(s *State) bool {
								return s.DesiredState == service.SVCStop || !s.Static
							})
							logger.WithField("unscheduled", count).Warn("Host is experiencing an outage.  Cleaned up orphaned nodes")

							// To prevent a tight loop, wait for something to
							// happen.
							select {
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
	logger := plog.WithField("poolid", h.poolid)

	var p PoolNode
	if err := h.conn.Get("/pools/"+h.poolid, &p); err != nil {
		logger.WithError(err).Warn("Could not look up resource pool for connection timeout")
		return 0
	}

	return p.GetConnectionTimeout()
}

// GetRegisteredHosts returns a list of hosts that are active.  If there are
// zero active hosts, then it will wait until at least one host is available.
func (h *HostRegistryListener) GetRegisteredHosts(cancel <-chan interface{}) ([]host.Host, error) {
	logger := plog.WithField("poolid", h.poolid)

	hosts := []host.Host{}
	for {
		hostids, err := GetCurrentHosts(h.conn, h.poolid)
		if err != nil {
			return nil, err
		}

		for _, hostid := range hostids {
			hstlog := logger.WithField("hostid", hostid)

			// only return hosts that are not locked
			ch, err := h.conn.Children(h.GetPath(hostid, "locked"))
			if err != nil && err != client.ErrNoNode {

				hstlog.WithError(err).Debug("Could not check if host is locked")

				// TODO: wrap error?
				return nil, err
			}

			isLocked := len(ch) > 0
			if !isLocked {
				hdat := host.Host{}
				err := h.conn.Get(h.GetPath(hostid), &HostNode{Host: &hdat})
				if err == client.ErrNoNode {
					continue
				} else if err != nil {

					hstlog.WithError(err).Debug("Could not load host")

					// TODO: wrap error?
					return nil, err
				}
				hosts = append(hosts, hdat)
			}
		}

		if count := len(hosts); count > 0 {

			logger.WithField("hostcount", count).Debug("Loaded active hosts")
			return hosts, nil
		}

		logger.Warn("No active hosts registered, waiting")
		select {
		case <-h.isOnline:
			logger.Info("At least one active host detected, checking")
		case <-cancel:
			return []host.Host{}, nil
		}
	}
}

// RegisterHost persists a registered host to the coordinator.  This is managed
// by the worker node, so it is expected that the connection will be pre-loaded
// with the path to the resource pool.
func RegisterHost(cancel <-chan interface{}, conn client.Connection, hostid string) error {
	logger := plog.WithField("hostid", hostid)

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

			logger.WithError(err).Debug("Could not check if host is registered")

			// TODO: wrap error?
			return err

		} else if !regok {

			logger.Warn("Host not found; system is idle")
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

			logger.Warn("Host is not registered; system is idle")
			select {
			case <-regev:
			case <-cancel:
				return nil
			}
			close(stop)
			stop = make(chan struct{})
			continue
		} else if err != nil && err != client.ErrNodeExists {

			logger.WithError(err).Debug("Could not check if host is already registered")

			// TODO: wrap error?
			return err
		}

		// the host becomes active
		ch, ev, err := conn.ChildrenW(pth, stop)
		if err == client.ErrNoNode {

			logger.Warn("Host is not active; system is idle")
			select {
			case <-regev:
			case <-cancel:
				return nil
			}
			close(stop)
			stop = make(chan struct{})
			continue
		} else if err != nil {

			logger.WithError(err).Debug("Could not check if host is active")

			// TODO: wrap error?
			return err
		}

		// register the host if it isn't showing up as active
		if len(ch) == 0 {
			// Need to give the ephemeral a node name, despite the name
			// changing when it is written to the coordinator.
			_, err = conn.CreateEphemeralIfExists(path.Join(pth, hostid), &client.Dir{})
			if err != nil {

				logger.WithError(err).Debug("Could not register host")

				// TODO: wrap error?
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
