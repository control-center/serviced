// Copyright 2017 The Serviced Authors.
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
	"sync"

	"github.com/Sirupsen/logrus"
	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/zzk"
)

// HostIPHandler is the handler for running the HostIPListener.
type HostIPHandler interface {
	// Bind adds a virtual ip to an interface. If the interface is already
	// bound then this command returns nil.
	BindIP(ip, netmask, iface string) error

	// Release removes a virtual ip from an interface.  If the interface doesn't
	// exist, then this command returns nil.
	ReleaseIP(ip string) error
}

// HostIPListener is the listener for monitoring virtual ips.
type HostIPListener struct {
	hostid  string
	handler HostIPHandler
	conn    client.Connection

	active  *sync.WaitGroup
	passive map[string]struct{}
	mu      *sync.Mutex
}

// NewHostIPListener instantiates a new host ip listener.
func NewHostIPListener(hostid string, handler HostIPHandler, binds []string) *HostIPListener {
	passive := make(map[string]struct{})
	for _, ip := range binds {
		req := IPRequest{HostID: hostid, IPAddress: ip}
		passive[req.IPID()] = struct{}{}
	}

	return &HostIPListener{
		hostid:  hostid,
		handler: handler,
		active:  &sync.WaitGroup{},
		passive: passive,
		mu:      &sync.Mutex{},
	}
}

// Listen implements zzk.Listener2. It starts the listener for the host ip
// nodes.
func (l *HostIPListener) Listen(cancel <-chan interface{}, conn client.Connection) {
	zzk.Listen2(cancel, conn, l)
}

// Exited implements zzk.Listener2.  It manages cleanup for the host ip
func (l *HostIPListener) Exited() {
	// wait for all the active threads to exit
	l.active.Wait()

	// now unbind all of the orphaned ips
	l.Post(map[string]struct{}{})
}

// SetConn implements zzk.Spawner.  It sets the zookeeper connection for the
// listener.
func (l *HostIPListener) SetConn(conn client.Connection) {
	l.conn = conn
}

// Path implements zzk.Spawner.  It returns the path to the parent.
func (l *HostIPListener) Path() string {
	return path.Join("/hosts", l.hostid, "ips")
}

// Pre implements zzk.Spawner.  It is the synchronous action that gets called
// before Spawn.
func (l *HostIPListener) Pre() {
	l.active.Add(1)
}

// Post synchronizes the passive thread list by unbinding all inactive and
// orphaned ips.
func (l *HostIPListener) Post(p map[string]struct{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for id := range l.passive {
		if _, ok := p[id]; !ok {
			delete(l.passive, id)
			_, ipaddr, _ := ParseIPID(id)
			req := IPRequest{HostID: l.hostid, IPAddress: ipaddr}
			l.release(req)
		}
	}
}

// Spawn implements zzk.Spawner.  It starts a new watcher for the given child
// node.
func (l *HostIPListener) Spawn(cancel <-chan struct{}, ipid string) {
	defer l.active.Done()
	logger := plog.WithField("ipid", ipid)

	// check valid ip id
	_, ipaddr, err := ParseIPID(ipid)
	if err != nil {
		logger.WithError(err).Warn("Deleting invalid ip id")
		if err := l.conn.Delete(path.Join(l.Path(), ipid)); err != nil && err != client.ErrNoNode {
			logger.WithError(err).Error("Could not delete invalid ip id")
		}
		return
	}

	logger = logger.WithField("ipaddress", ipaddr)

	// set up the request object for updates
	var (
		hpth = path.Join(l.Path(), ipid)
		ppth = path.Join("/ips", ipid)
		req  = IPRequest{
			HostID:    l.hostid,
			IPAddress: ipaddr,
		}
	)

	if !l.loadThread(req) {
		return
	}

	done := make(chan struct{})
	defer func() { close(done) }()
	for {
		// set up a listener on host ip node
		hdat := &HostIP{}
		hevt, err := l.conn.GetW(hpth, hdat, done)
		if err == client.ErrNoNode {
			logger.Debug("Host IP was removed, exiting")
			l.release(req)
			return
		} else if err != nil {
			logger.WithError(err).Warn("Could not watch host ip, detaching")
			l.saveThread(ipid)
			return
		}

		// set up a listener on the pool ip node to ensure the node's
		// existance.
		ok, pevt, err := l.conn.ExistsW(ppth, done)
		if err != nil {
			logger.WithError(err).Error("Could not watch pool ip, detaching")
			l.saveThread(ipid)
			return
		} else if !ok {
			logger.Debug("Pool ip was removed, exiting")
			l.release(req)
			return
		}

		// update the binding
		if err := l.handler.BindIP(ipaddr, hdat.Netmask, hdat.BindInterface); err != nil {
			logger.WithError(err).Error("Could not bind virtual ip, exiting")
			l.release(req)
			return
		}

		// set the status of the ip
		if err := UpdateIP(l.conn, req, func(ip *IP) bool {
			if !ip.OK {
				ip.OK = true
				return true
			}
			return false
		}); err != nil {
			logger.WithError(err).Error("Could not update ip state, detaching")
			l.saveThread(ipid)
			return
		}

		select {
		case <-hevt:
		case <-pevt:
		case <-cancel:
		}

		// cancel takes precedence
		select {
		case <-cancel:
			logger.Debug("Listener shut down, detaching")
			l.saveThread(ipid)
			return
		default:
		}

		close(done)
		done = make(chan struct{})
	}
}

// loadThread loads the thread from the passive map
func (l *HostIPListener) loadThread(req IPRequest) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	var (
		id  = req.IPID()
		pth = path.Join("/ips", id)
	)

	logger := plog.WithFields(logrus.Fields{
		"ipid":       id,
		"poolippath": pth,
	})

	// load from cache
	_, ipok := l.passive[req.IPAddress]

	// make sure the node exists
	ok, err := l.conn.Exists(pth)
	if err != nil {
		logger.WithError(err).Error("Could not check the existance of pool ip, exiting")
		return false
	}

	if ipok {
		// remove from the thread cache
		delete(l.passive, req.IPAddress)
	}

	if !ok {
		l.release(req)
	}

	return ok
}

// saveThread saves the thread to the passive map
func (l *HostIPListener) saveThread(id string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.passive[id] = struct{}{}
}

// release drops the virtual ip and cleans up the zookeeper node
func (l *HostIPListener) release(req IPRequest) {
	logger := plog.WithField("ipaddress", req.IPAddress)
	logger.Info("Releasing ip binding")
	if err := l.handler.ReleaseIP(req.IPAddress); err != nil {
		logger.WithError(err).Error("Could not release ip binding")
	}

	if err := DeleteIP(l.conn, req); err != nil {
		logger.WithError(err).Error("Could not delete ip node from zookeeper")
	}
}
