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

	log "github.com/Sirupsen/logrus"
	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/domain/service"
)

// HostNode is the storage object for host data
type HostNode struct {
	*host.Host
	version interface{}
}

// Version implements client.Node
func (h *HostNode) Version() interface{} {
	return h.version
}

// SetVersion implements client.Node
func (h *HostNode) SetVersion(version interface{}) {
	h.version = version
}

// AddHost creates the host if it doesn't already exist. (uses a pool-based
// connection)
func AddHost(conn client.Connection, h host.Host) error {
	pth := path.Join("/hosts", h.ID)

	logger := plog.WithFields(log.Fields{
		"poolid": h.PoolID,
		"hostid": h.ID,
		"zkpath": pth,
	})

	// create the /hosts path if it doesn't exist
	if err := conn.CreateIfExists("/hosts", &client.Dir{}); err != nil && err != client.ErrNodeExists {
		logger.WithError(err).Debug("Could not initialize hosts path in zookeeper")
		return err
	}

	if err := conn.CreateIfExists(pth, &HostNode{Host: &h}); err != nil {
		logger.WithError(err).Debug("Could not create host entry in zookeeper")
		return err
	}

	logger.Debug("Created entry for host in zookeeper")
	return nil
}

// UpdateHost updates an existing host. (uses a pool-based connection)
func UpdateHost(conn client.Connection, h host.Host) error {
	pth := path.Join("/hosts", h.ID)

	logger := plog.WithFields(log.Fields{
		"poolid": h.PoolID,
		"hostid": h.ID,
		"zkpath": pth,
	})

	if err := conn.Set(pth, &HostNode{Host: &h}); err != nil {
		logger.WithError(err).Debug("Could not update host entry in zookeeper")
		return err
	}

	logger.Debug("Updated entry to host in zookeeper")
	return nil
}

// RemoveHost removes an existing host, after waiting for existing states to
// shutdown.
func RemoveHost(cancel <-chan struct{}, conn client.Connection, poolID, hostID string) error {
	basepth := ""
	if poolID != "" {
		basepth = path.Join("/pools", poolID)
	}
	pth := path.Join(basepth, "/hosts", hostID)

	logger := plog.WithFields(log.Fields{
		"hostid": hostID,
		"zkpath": pth,
	})

	// lock the host from scheduling
	mu, err := conn.NewLock(path.Join(pth, "locked"))
	if err != nil {
		logger.WithError(err).Debug("Could not instantiate scheduling lock")
		return err
	}
	if err := mu.Lock(); err != nil {
		logger.WithError(err).Debug("Could not lock host from scheduling")
		return err
	}
	defer mu.Unlock()

	// schedule all running states to stop
	done := make(chan struct{})
	defer func() { close(done) }()
	for {

		// clean any bad host states
		if err := CleanHostStates(conn, poolID, hostID); err != nil {
			return err
		}

		// get the list of states
		ch, ev, err := conn.ChildrenW(path.Join(pth, "instances"), done)
		if err != nil && err != client.ErrNoNode {
			logger.WithError(err).Debug("Could not watch instances for host")
			return err
		}

		for _, stateID := range ch {
			st8log := logger.WithField("stateid", stateID)

			_, serviceID, instanceID, err := ParseStateID(stateID)
			if err != nil {

				// This should never happen, but handle it
				st8log.WithError(err).Error("Invalid state id while monitoring host")
				return err
			}

			req := StateRequest{
				PoolID:     poolID,
				HostID:     hostID,
				ServiceID:  serviceID,
				InstanceID: instanceID,
			}

			// set the state to stopped if not already stopped
			if err := UpdateState(conn, req, func(s *State) bool {
				if s.DesiredState != service.SVCStop {
					s.DesiredState = service.SVCStop
					return true
				}
				return false
			}); err != nil {
				return err
			}
		}

		// if all the states have died, exit loop
		if len(ch) == 0 {
			break
		}

		// otherwise, wait for the number of states to change
		select {
		case <-ev:
		case <-cancel:
			logger.Debug("Delete was cancelled")
			return nil
		}
		close(done)
		done = make(chan struct{})
	}

	if err := removeHost(conn, poolID, hostID); err != nil {
		logger.WithError(err).Debug("Could not delete host entry from zookeeper")
		return err
	}

	logger.Debug("Deleted host entry from zookeeper")
	return nil
}

// removeHost deletes a host from zookeeper
func removeHost(conn client.Connection, poolID, hostID string) error {
	basepth := ""
	if poolID != "" {
		basepth = path.Join("/pools", poolID)
	}
	pth := path.Join(basepth, "/hosts", hostID)

	t := conn.NewTransaction()
	if err := rmr(conn, t, pth); err != nil {
		return err
	}
	if err := t.Commit(); err != nil {
		return err
	}
	return nil
}

// rmr is a recursive delete via transaction
func rmr(conn client.Connection, t client.Transaction, pth string) error {
	ch, err := conn.Children(pth)
	if err == client.ErrNoNode {
		return nil
	} else if err != nil {
		return err
	}

	for _, n := range ch {
		if err := rmr(conn, t, path.Join(pth, n)); err != nil {
			return err
		}
	}
	t.Delete(pth)
	return nil
}

// SyncHosts synchronizes the hosts to the provided list (uses a pool-based
// connection)
func SyncHosts(conn client.Connection, hosts []host.Host) error {
	pth := path.Join("/hosts")

	logger := plog.WithField("zkpath", pth)

	// look up children host ids
	ch, err := conn.Children(pth)
	if err != nil && err != client.ErrNoNode {
		logger.WithError(err).Debug("Could not look up hosts")
		return err
	}

	// store the host ids in a hash map
	chmap := make(map[string]struct{})
	for _, hostID := range ch {
		chmap[hostID] = struct{}{}
	}

	// set the hosts
	for _, h := range hosts {
		if _, ok := chmap[h.ID]; ok {
			if err := UpdateHost(conn, h); err != nil {
				return err
			}
			delete(chmap, h.ID)
		} else {
			if err := AddHost(conn, h); err != nil {
				return err
			}
		}
	}

	// remove any leftovers
	for hostID := range chmap {
		if err := removeHost(conn, "", hostID); err != nil {
			logger.WithField("hostid", hostID).WithError(err).Debug("Could not delete host entry from zookeeper")
			return err
		}
	}
	return nil
}
