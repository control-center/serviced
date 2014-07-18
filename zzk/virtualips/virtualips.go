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

	glog.V(2).Infof("Host %s waiting to acquire virtual ip %s", l.hostID, ip)
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

		glog.V(2).Infof("Host %s binding to %s", l.hostID, ip)
		rebind, err := l.bind(&vip, index)
		if err != nil {
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
		case <-rebind:
			// If the primary virtual IP is removed, all other virtual IPs on
			// that subnet are removed.  This is in place to restore the
			// virtual IPs that were removed soley by the removal of the
			// primary virtual IP.
			glog.V(2).Infof("Host %s rebinding to %s", l.hostID, ip)
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

func (l *VirtualIPListener) reset() {
	for _, ipChan := range l.ips {
		ipChan <- true
	}
}

func (l *VirtualIPListener) get(ip string) <-chan bool {
	l.ips[ip] = make(chan bool, 1)
	return l.ips[ip]
}

func (l *VirtualIPListener) bind(vip *pool.VirtualIP, index int) (<-chan bool, error) {
	vmap, err := l.handler.VirtualInterfaceMap(virtualInterfacePrefix)
	if err != nil {
		return nil, err
	}

	if _, ok := vmap[vip.IP]; !ok {
		if vip.BindInterface == "" {
			return nil, ErrInvalidVirtualIP
		}
		vname := fmt.Sprintf("%s%s%d", vip.BindInterface, virtualInterfacePrefix, index)
		if err := l.handler.BindVirtualIP(vip, vname); err != nil {
			return nil, err
		}
	}

	return l.get(vip.IP), nil
}

func (l *VirtualIPListener) unbind(ip string) error {
	defer l.reset()
	vmap, err := l.handler.VirtualInterfaceMap(virtualInterfacePrefix)
	if err != nil {
		return err
	}

	if vip, ok := vmap[ip]; ok {
		return l.handler.UnbindVirtualIP(vip)
	}

	return nil
}

func SyncVirtualIPs(conn client.Connection, virtualIPs []pool.VirtualIP) error {
	var current []string
	if exists, err := zzk.PathExists(conn, vippath()); err != nil {
		return err
	} else if !exists {
		//pass
	} else if current, err = conn.Children(vippath()); err != nil {
		return err
	}

	unsynced := make(map[string]*pool.VirtualIP)
	for i, virtualIP := range virtualIPs {
		unsynced[virtualIP.IP] = &virtualIPs[i]
	}

	for _, ip := range current {
		if _, ok := unsynced[ip]; ok {
			delete(unsynced, ip)
		} else if err := RemoveVirtualIP(conn, ip); err != nil {
			return err
		}
	}

	for _, virtualIP := range unsynced {
		if err := AddVirtualIP(conn, virtualIP); err != nil {
			return err
		}
	}

	return nil
}

func AddVirtualIP(conn client.Connection, virtualIP *pool.VirtualIP) error {
	var node VirtualIPNode
	path := vippath(virtualIP.IP)

	glog.V(1).Infof("Adding virtual ip to zookeeper: %s", path)
	if err := conn.Create(path, &node); err != nil {
		return err
	}
	node.VirtualIP = virtualIP
	return conn.Set(path, &node)
}

func RemoveVirtualIP(conn client.Connection, ip string) error {
	glog.V(1).Infof("Removing virtual ip from zookeeper: %s", vippath(ip))
	return conn.Delete(vippath(ip))
}

func GetHostID(conn client.Connection, ip string) (string, error) {
	leader := zzk.NewHostLeader(conn, "", vippath(ip))
	return zzk.GetHostID(leader)
}
