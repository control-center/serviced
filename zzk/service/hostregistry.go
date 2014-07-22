// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

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
	return RegisterHost(conn, node.ID)
}

// Update implements zzk.Node
func (node *HostNode) Update(conn client.Connection) error {
	return nil
}

// Delete implements zzk.Node
func (node *HostNode) Delete(conn client.Connection) error {
	return UnregisterHost(conn, node.ID)
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
	conn     client.Connection
	shutdown chan interface{}
}

// NewHostRegistryListener instantiates a new HostRegistryListener
func NewHostRegistryListener(conn client.Connection) (*HostRegistryListener, error) {
	// initialize the hostregistry
	if exists, err := zzk.PathExists(conn, hostregpath()); err != nil {
		return nil, err
	} else if exists {
		// pass
	} else if err := conn.CreateDir(hostregpath()); err != nil {
		return nil, err
	}

	return &HostRegistryListener{conn, make(chan interface{})}, nil
}

// GetConnection implements zzk.Listener
func (l *HostRegistryListener) GetConnection() client.Connection { return l.conn }

// GetPath implements zzk.Listener
func (l *HostRegistryListener) GetPath(nodes ...string) string { return hostregpath(nodes...) }

// Ready implements zzk.Listener
func (l *HostRegistryListener) Ready() (err error) { return }

// Done shuts down any running processes outside of the main listener, like l.GetHosts()
func (l *HostRegistryListener) Done() { close(l.shutdown) }

// Spawn listens on the host registry and waits til the node is deleted to unregister
func (l *HostRegistryListener) Spawn(shutdown <-chan interface{}, eHostID string) {
	for {
		var host host.Host
		event, err := l.conn.GetW(l.GetPath(eHostID), &HostNode{Host: &host})
		if err != nil {
			glog.Errorf("Could not load ephemeral node %s: %s", eHostID, err)
			return
		}

		select {
		case e := <-event:
			if e.Type == client.EventNodeDeleted {
				glog.V(1).Info("Unregistering host: ", host.ID)
				l.unregister(host.ID)
				return
			}
			glog.V(2).Infof("Receieved event: ", e)
		case <-shutdown:
			glog.V(2).Infof("Host listener for %s (%s) received signal to shutdown", host.ID, eHostID)
			return
		}
	}
}

func (l *HostRegistryListener) unregister(hostID string) {
	if exists, err := zzk.PathExists(l.conn, hostpath(hostID)); err != nil {
		glog.Errorf("Unable to check path for host %s: %s", hostID, err)
		return
	} else if !exists {
		return
	}

	rss, err := LoadRunningServicesByHost(l.conn, hostID)
	if err != nil {
		glog.Errorf("Unable to get the running services for host %s: %s", hostID, err)
		return
	}

	for _, rs := range rss {
		if err := l.conn.Delete(hostpath(rs.HostID, rs.ID)); err != nil {
			glog.Warningf("Could not delete service instance %s on host %s", rs.ID, rs.HostID)
		}
		if err := l.conn.Delete(servicepath(rs.ServiceID, rs.ID)); err != nil {
			glog.Warningf("Could not delete service instance %s for service %s", rs.ID, rs.ServiceID)
		}
	}
	return
}

// GetHosts returns all of the registered hosts
func (l *HostRegistryListener) GetHosts() (hosts []*host.Host, err error) {
	if err := zzk.Ready(l.shutdown, l.conn, l.GetPath()); err != nil {
		return nil, err
	}

	for {
		ehosts, eventW, err := l.conn.ChildrenW(hostregpath())
		if err != nil {
			return nil, err
		}

		for _, ehostID := range ehosts {
			var host host.Host
			if err := l.conn.Get(hostregpath(ehostID), &HostNode{Host: &host}); err != nil {
				return nil, err
			}
			if exists, err := zzk.PathExists(l.conn, hostpath(host.ID)); err != nil {
				return nil, err
			} else if exists {
				hosts = append(hosts, &host)
			}
		}

		// wait if no hosts are registered
		if len(hosts) > 0 {
			return hosts, nil
		}

		select {
		case <-eventW:
			// pass
		case <-l.shutdown:
			return nil, ErrShutdown
		}
	}
}

func SyncHosts(conn client.Connection, hosts []*host.Host) error {
	nodes := make([]zzk.Node, len(hosts))
	for i := range hosts {
		nodes[i] = &HostNode{Host: hosts[i]}
	}
	return zzk.Sync(conn, nodes, hostpath())
}

func RegisterHost(conn client.Connection, hostID string) error {
	if exists, err := zzk.PathExists(conn, hostpath(hostID)); err != nil {
		return err
	} else if exists {
		return nil
	}

	return conn.CreateDir(hostpath(hostID))
}

func UnregisterHost(conn client.Connection, hostID string) error {
	if exists, err := zzk.PathExists(conn, hostpath(hostID)); err != nil {
		return err
	} else if !exists {
		return nil
	}

	return conn.Delete(hostpath(hostID))
}
