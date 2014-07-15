// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package service

import (
	"errors"
	"path"

	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/coordinator/client"
	"github.com/zenoss/serviced/domain/host"
	zkutils "github.com/zenoss/serviced/zzk/utils"
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
	Host    *host.Host
	version interface{}
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
	shutdown <-chan interface{}
}

// NewHostRegistryListener instantiates a new HostRegistryListener
func NewHostRegistryListener(conn client.Connection) (*HostRegistryListener, error) {
	return new(HostRegistryListener).init(conn)
}

func (l *HostRegistryListener) init(conn client.Connection) (*HostRegistryListener, error) {
	regpath := hostregpath()
	if exists, err := zkutils.PathExists(conn, regpath); err != nil {
		glog.Errorf("Error checking path %s: %s", regpath, err)
		return nil, err
	} else if exists {
		// pass
	} else if conn.CreateDir(regpath); err != nil {
		glog.Errorf("Error creating path %s: %s", regpath, err)
		return nil, err
	}

	return &HostRegistryListener{conn: conn}, nil
}

// Listen listens for changes to /registry/hosts and updates the host list
// accordingly
func (l *HostRegistryListener) Listen(shutdown <-chan interface{}) {
	var (
		_shutdown  = make(chan interface{})
		done       = make(chan string)
		processing = make(map[string]interface{})
	)
	l.shutdown = _shutdown

	defer func() {
		glog.Infof("Host registry receieved interrupt")
		close(_shutdown)
		for len(processing) > 0 {
			delete(processing, <-done)
		}
	}()

	for {
		ehostIDs, event, err := l.conn.ChildrenW(hostregpath())
		if err != nil {
			glog.Errorf("Could not watch host registry: %s", err)
			return
		}

		for _, ehostID := range ehostIDs {
			if _, ok := processing[ehostID]; !ok {
				glog.V(1).Info("Spawning a listener for ephemeral host ", ehostID)
				processing[ehostID] = nil
				go l.listenHost(done, ehostID)
			}
		}

		select {
		case e := <-event:
			if e.Type == client.EventNodeDeleted {
				glog.Infof("Host registry is no longer available, shutting down")
				return
			}
			glog.V(2).Infof("Receieved event: %v", e)
		case ehostID := <-done:
			glog.V(2).Infof("Cleaning up %s", ehostID)
			delete(processing, ehostID)
		case <-shutdown:
			return
		}
	}
}

func (l *HostRegistryListener) listenHost(done chan<- string, ehostID string) {
	defer func() {
		glog.V(2).Info("Shutting down listener for ephemeral host: ", ehostID)
		done <- ehostID
	}()

	hpath := hostregpath(ehostID)
	for {
		var host host.Host
		event, err := l.conn.GetW(hpath, &HostNode{Host: &host})
		if err != nil {
			glog.Errorf("Could not load ephemeral node %s: %s", ehostID, err)
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
		case <-l.shutdown:
			glog.V(2).Infof("Host listener for %s (%s) received signal to shutdown", host.ID, ehostID)
			return
		}
	}
}

func (l *HostRegistryListener) unregister(hostID string) {
	if exists, err := zkutils.PathExists(l.conn, hostpath(hostID)); err != nil {
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
	var (
		ehosts []string
		eventW <-chan client.Event
	)

	// wait if no hosts are registered
	for {
		ehosts, eventW, err = l.conn.ChildrenW(hostregpath())
		if err != nil {
			return nil, err
		}

		for _, ehostID := range ehosts {
			var host host.Host
			if err := l.conn.Get(hostregpath(ehostID), &HostNode{Host: &host}); err != nil {
				return nil, err
			}
			if exists, err := zkutils.PathExists(l.conn, hostpath(host.ID)); err != nil {
				return nil, err
			} else if exists {
				hosts = append(hosts, &host)
			}
		}

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

func RegisterHost(conn client.Connection, hostID string) error {
	if exists, err := zkutils.PathExists(conn, hostpath(hostID)); err != nil {
		return err
	} else if exists {
		return nil
	}

	return conn.CreateDir(hostpath(hostID))
}

func UnregisterHost(conn client.Connection, hostID string) error {
	if exists, err := zkutils.PathExists(conn, hostpath(hostID)); err != nil {
		return err
	} else if !exists {
		return nil
	}

	return conn.Delete(hostpath(hostID))
}
