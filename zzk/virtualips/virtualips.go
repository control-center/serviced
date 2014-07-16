// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package virtualips

import (
	"errors"
	"fmt"
	"path"

	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/coordinator/client"
	"github.com/zenoss/serviced/domain/pool"
	"github.com/zenoss/serviced/zzk"
)

const (
	zkVirtualIP            = "/virtualIPs"
	virtualInterfacePrefix = ":zvip"
)

var (
	ErrInvalidVirtualIP = errors.New("invalid virtual ip")
)

func vippath(nodes ...string) string {
	p := append([]string{zkVirtualIP}, nodes...)
	return path.Join(p...)
}

type VirtualIPNode struct {
	VirtualIP *pool.VirtualIP
	version   interface{}
}

func (node *VirtualIPNode) Version() interface{}           { return node.version }
func (node *VirtualIPNode) SetVersion(version interface{}) { node.version = version }

// VirtualIPHandler is the handler interface for virtual ip bindings on the host
type VirtualIPHandler interface {
	BindVirtualIP(*pool.VirtualIP, string) error
	UnbindVirtualIP(*pool.VirtualIP) error
	VirtualInterfaceMap(string) (map[string]*pool.VirtualIP, error)
}

// VirtualIPListener is the listener object for watching the zk object for
// virtual IP nodes
type VirtualIPListener struct {
	conn    client.Connection
	handler VirtualIPHandler
	hostID  string

	index chan int
	ips   map[string]chan bool
}

// NewVirtualIPListener instantiates a new VirtualIPListener object
func NewVirtualIPListener(conn client.Connection, handler VirtualIPHandler, hostID string) *VirtualIPListener {
	l := &VirtualIPListener{
		conn:    conn,
		handler: handler,
		hostID:  hostID,
		index:   make(chan int),
		ips:     make(map[string]chan bool),
	}

	go func(start int) {
		for {
			l.index <- start
			start++
		}
	}(0)

	return l
}

// GetConnection implements zzk.Listener
func (l *VirtualIPListener) GetConnection() client.Connection { return l.conn }

// GetPath implements zzk.Listener
func (l *VirtualIPListener) GetPath(nodes ...string) string {
	return vippath(nodes...)
}

// Ready removes all virtual IPs that may be present
func (l *VirtualIPListener) Ready() error {
	vmap, err := l.handler.VirtualInterfaceMap(virtualInterfacePrefix)
	if err != nil {
		return err
	}

	for _, vip := range vmap {
		if err := l.handler.UnbindVirtualIP(vip); err != nil {
			return err
		}
	}
	return nil
}

// Done implements zzk.Listener
func (l *VirtualIPListener) Done() {}

// Spawn implements zzk.Listener
func (l *VirtualIPListener) Spawn(shutdown <-chan interface{}, ip string) {

	// Try to take lead on the path
	leader := zzk.NewHostLeader(l.conn, l.hostID, l.GetPath(ip))
	_, err := leader.TakeLead()
	if err != nil {
		glog.Errorf("Error while trying to acquire a lock for %s: %s", ip, err)
		return
	}
	defer leader.ReleaseLead()

	// Check if the path still exists
	if exists, err := zzk.PathExists(l.conn, l.GetPath(ip)); err != nil {
		glog.Errorf("Error while checking ip %s: %s", ip, err)
		return
	} else if !exists {
		return
	}

	index := l.getIndex()
	for {
		var vip pool.VirtualIP
		event, err := l.conn.GetW(l.GetPath(ip), &VirtualIPNode{VirtualIP: &vip})
		if err != nil {
			glog.Errorf("Could not load virtual ip %s: %s", ip, err)
			return
		}

		if err := l.bind(&vip, index); err != nil {
			glog.Errorf("Could not bind to virtual ip %s: %s", ip, err)
			return
		}

		select {
		case e := <-event:
			// If the virtual ip is changed, you need to update the bindings
			if err := l.unbind(ip); err != nil {
				glog.Errorf("Could not unbind to virtual ip %s: %s", ip, err)
				return
			}
			if e.Type == client.EventNodeDeleted {
				return
			}
			glog.V(4).Infof("virtual ip listener for %s receieved event: %v", ip, e)
		case <-l.get(ip):
			// if the primary virtual IP is removed, all other virtual IPs on that subnet are removed
			// this is in place to restore the virtual IPs that were removed soley by the removal of the primary virtual IP
		case <-shutdown:
			if err := l.unbind(ip); err != nil {
				glog.Errorf("Could not unbind to virtual ip %s: %s", ip, err)
				return
			}
		}
	}
}

func (l *VirtualIPListener) getIndex() int {
	return <-l.index
}

func (l *VirtualIPListener) reset(ip string) {
	if _, ok := l.ips[ip]; ok {
		delete(l.ips, ip)
	}

	for _, ipChan := range l.ips {
		ipChan <- true
	}
}

func (l *VirtualIPListener) get(ip string) <-chan bool {
	if _, ok := l.ips[ip]; !ok {
		l.ips[ip] = make(chan bool)
	}

	return l.ips[ip]
}

func (l *VirtualIPListener) bind(vip *pool.VirtualIP, index int) error {
	vmap, err := l.handler.VirtualInterfaceMap(virtualInterfacePrefix)
	if err != nil {
		return err
	}

	if _, ok := vmap[vip.IP]; ok {
		glog.V(2).Infof("Virtual IP %s already exists", vip.IP)
		return nil
	}

	if vip.BindInterface == "" {
		return ErrInvalidVirtualIP
	}

	vname := fmt.Sprintf("%s%s%d", vip.BindInterface, virtualInterfacePrefix, index)
	return l.handler.BindVirtualIP(vip, vname)
}

func (l *VirtualIPListener) unbind(ip string) error {
	defer l.reset(ip)
	vmap, err := l.handler.VirtualInterfaceMap(virtualInterfacePrefix)
	if err != nil {
		return err
	}

	if vip, ok := vmap[ip]; ok {
		return l.handler.UnbindVirtualIP(vip)
	}

	return nil
}

func AddVirtualIP(conn client.Connection, virtualIP *pool.VirtualIP) error {
	return conn.Create(vippath(virtualIP.IP), &VirtualIPNode{VirtualIP: virtualIP})
}

func RemoveVirtualIP(conn client.Connection, ip string) error {
	return conn.Delete(vippath(ip))
}

func GetHostID(conn client.Connection, ip string) (string, error) {
	leader := zzk.NewHostLeader(conn, "", vippath(ip))
	return zzk.GetHostID(leader)
}
