// Copyright 2014 The Serviced Authors.
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
	"errors"
	"path"

	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/zzk"
	"github.com/zenoss/glog"
)

const (
	zkRegistry = "/registry"
)

var (
	ErrHostInvalid = errors.New("invalid host")
	ErrShutdown    = errors.New("listener shut down")
)

func hostregpath(nodes ...string) string {
	p := append([]string{zkRegistry, zkHost}, nodes...)
	return path.Clean(path.Join(p...))
}

// HostNode is the zk node for Host
type HostNode struct {
	*host.Host
	version interface{}
}

// ID implements zzk.Node
func (node *HostNode) GetID() string {
	return node.ID
}

// Create implements zzk.Node
func (node *HostNode) Create(conn client.Connection) error {
	return AddHost(conn, node.Host)
}

// Update implements zzk.Node
func (node *HostNode) Update(conn client.Connection) error {
	return UpdateHost(conn, node.Host)
}

// Version implements client.Node
func (node *HostNode) Version() interface{} {
	return node.version
}

// SetVersion implements client.Node
func (node *HostNode) SetVersion(version interface{}) {
	node.version = version
}

// HostRegistryListener watches ephemeral nodes on /registry/hosts and provides
// information about available hosts
type HostRegistryListener struct {
	conn client.Connection
}

// NewHostRegistryListener instantiates a new HostRegistryListener
func NewHostRegistryListener() *HostRegistryListener {
	return &HostRegistryListener{}
}

// SetConnection implements zzk.Listener
func (l *HostRegistryListener) SetConnection(conn client.Connection) { l.conn = conn }

// GetPath implements zzk.Listener
func (l *HostRegistryListener) GetPath(nodes ...string) string { return hostregpath(nodes...) }

// Ready implements zzk.Listener
func (l *HostRegistryListener) Ready() (err error) { return }

// Done shuts down any running processes outside of the main listener, like l.GetHosts()
func (l *HostRegistryListener) Done() {}

// PostProcess implments zzk.Listener
func (l *HostRegistryListener) PostProcess(p map[string]struct{}) {}

// Spawn listens on the host registry and waits til the node is deleted to unregister
func (l *HostRegistryListener) Spawn(shutdown <-chan interface{}, eHostID string) {
	done := make(chan struct{})
	defer func(channel *chan struct{}) { close(*channel) }(&done)
	for {
		var host host.Host
		ev, err := l.conn.GetW(hostregpath(eHostID), &HostNode{Host: &host}, done)
		if err != nil {
			glog.Errorf("Could not load ephemeral node %s: %s", eHostID, err)
			return
		}
		select {
		case e := <-ev:
			if e.Type == client.EventNodeDeleted {
				removeInstancesOnHost(l.conn, host.ID)
				return
			}
		case <-shutdown:
			glog.V(2).Infof("Recieved signal to stop listening to %s for host %s (%s)", eHostID, host.ID, host.IPAddr)
			return
		}

		close(done)
		done = make(chan struct{})
	}
}

// registerHost alerts the resource manager that a host is available for
// scheduling.  Returns the path to the node.
func registerHost(conn client.Connection, host *host.Host) (string, error) {
	node := &HostNode{}

	// CreateEphemeral returns the full path from root.
	// e.g. /pools/default/registry/hosts/[node]
	hostpath, err := conn.CreateEphemeral(hostregpath(host.ID), node)
	if err != nil {
		glog.Errorf("Could not register host %s (%s): %s", host.ID, host.IPAddr, err)
		return "", err
	}
	relpath := hostregpath(path.Base(hostpath))
	node.Host = host
	if err := conn.Set(relpath, node); err != nil {
		defer conn.Delete(relpath)
		glog.Errorf("Could not register host %s (%s): %s", host.ID, host.IPAddr, err)
		return "", err
	}
	return relpath, nil
}

// GetHosts returns active hosts or waits if none are available.
func GetRegisteredHosts(conn client.Connection, cancel <-chan interface{}) ([]*host.Host, error) {
	// wait for the parent node to be available
	if err := zzk.Ready(cancel, conn, hostregpath()); err != nil {
		return nil, err
	}
	done := make(chan struct{})
	defer func(channel *chan struct{}) { close(*channel) }(&done)
	for {
		// get all of the ready nodes
		children, ev, err := conn.ChildrenW(hostregpath(), done)
		if err != nil {
			return nil, err
		}
		var hosts []*host.Host
		for _, child := range children {
			host := &host.Host{}
			if err := conn.Get(hostregpath(child), &HostNode{Host: host}); err != nil {
				return nil, err
			}
			// has this host been added to the database?
			if exists, err := conn.Exists(hostpath(host.ID)); err != nil && err != client.ErrNoNode {
				return nil, err
			} else if exists {
				hosts = append(hosts, host)
			}
		}
		if len(hosts) == 0 {
			// wait for hosts if none are registered
			glog.Warningf("No hosts are registered in pool; did you add hosts or bounce running agents?")
			select {
			case <-ev:
			case <-cancel:
				return nil, ErrShutdown
			}
		} else {
			return hosts, nil
		}

		close(done)
		done = make(chan struct{})
	}
}

func GetActiveHosts(conn client.Connection) ([]string, error) {
	ehosts, err := conn.Children(hostregpath())
	if err == client.ErrNoNode {
		return []string{}, nil
	}
	if err != nil {
		return nil, err
	}
	hostIDs := make([]string, len(ehosts))
	for _, ehostID := range ehosts {
		var ehost host.Host
		if err := conn.Get(hostregpath(ehostID), &HostNode{Host: &ehost}); err != nil {
			glog.Warningf("Could not look up registration for host at %s: %s", ehostID, err)
			continue
		}
		if exists, err := conn.Exists(hostpath(ehost.ID)); !exists {
			glog.Warningf("Host %s (%s) is registered, but not available (%s).  Please restart this host.", ehost.ID, ehost.Name, err)
		} else {
			hostIDs = append(hostIDs, ehost.ID)
		}
	}
	return hostIDs, nil
}

func SyncHosts(conn client.Connection, hosts []host.Host) error {
	nodes := make([]zzk.Node, len(hosts))
	for i := range hosts {
		nodes[i] = &HostNode{Host: &hosts[i]}
	}
	return zzk.Sync(conn, nodes, hostpath())
}

func AddHost(conn client.Connection, host *host.Host) error {
	var node HostNode
	if err := conn.Create(hostpath(host.ID), &node); err != nil {
		return err
	}
	node.Host = host
	return conn.Set(hostpath(host.ID), &node)
}

func UpdateHost(conn client.Connection, host *host.Host) error {
	var node HostNode
	if err := conn.Get(hostpath(host.ID), &node); err != nil {
		return err
	}
	node.Host = host
	return conn.Set(hostpath(host.ID), &node)
}

func RemoveHost(cancel <-chan interface{}, conn client.Connection, hostID string) error {
	if exists, err := zzk.PathExists(conn, hostpath(hostID)); err != nil {
		return err
	} else if !exists {
		return nil
	}

	// stop all the instances running on that host
	nodes, err := conn.Children(hostpath(hostID))
	if err != nil {
		return err
	}
	for _, stateID := range nodes {
		if err := StopServiceInstance(conn, hostID, stateID); err != nil {
			glog.Errorf("Could not stop service instance %s: %s", stateID, err)
			return err
		}
	}

	// wait until all the service instances have stopped
	done := make(chan struct{})
	defer func(channel *chan struct{}) { close(*channel) }(&done)
loop:
	for {
		nodes, event, err := conn.ChildrenW(hostpath(hostID), done)
		if err != nil {
			return err
		} else if len(nodes) == 0 {
			break
		}
		glog.V(2).Infof("%d services still running on host %s", len(nodes), hostID)

		select {
		case <-event:
			// pass
		case <-cancel:
			glog.Warningf("Giving up on waiting for services on host %s to stop", hostID)
			break loop
		}

		close(done)
		done = make(chan struct{})
	}

	// remove the parent node
	return conn.Delete(hostpath(hostID))
}
