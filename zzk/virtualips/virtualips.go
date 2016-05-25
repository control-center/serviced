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

package virtualips

import (
	"errors"
	"fmt"
	"path"
	"time"

	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/domain/pool"
	"github.com/control-center/serviced/utils"
	"github.com/control-center/serviced/zzk"
	zkservice "github.com/control-center/serviced/zzk/service"
	"github.com/zenoss/glog"
)

const (
	zkVirtualIP            = "/virtualIPs"
	virtualInterfacePrefix = ":z"
	maxRetries             = 2
	waitTimeout            = 30 * time.Second
)

var (
	ErrInvalidVirtualIP = errors.New("invalid virtual ip")
)

func vippath(nodes ...string) string {
	p := append([]string{zkVirtualIP}, nodes...)
	return path.Join(p...)
}

type VirtualIPNode struct {
	*pool.VirtualIP
	version interface{}
}

// ID implements zzk.Node
func (node *VirtualIPNode) GetID() string {
	return node.IP
}

// Create implements zzk.Node
func (node *VirtualIPNode) Create(conn client.Connection) error {
	return AddVirtualIP(conn, node.VirtualIP)
}

// Update implements zzk.Node
func (node *VirtualIPNode) Update(conn client.Connection) error {
	return nil
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

	index chan uint
	ips   map[string]chan bool
	retry map[string]int
}

// NewVirtualIPListener instantiates a new VirtualIPListener object
func NewVirtualIPListener(handler VirtualIPHandler, hostID string) *VirtualIPListener {
	l := &VirtualIPListener{
		handler: handler,
		hostID:  hostID,
		index:   make(chan uint),
		ips:     make(map[string]chan bool),
	}

	// Index generator for bind interface
	// Clamp the index string length to 3 base 62 digits so that validation
	// methods can make sure the length of the VIP name doesn't exceed 15 chars.
	// Base 62 is used so that we can pack more indices into those 3 digits.
	go func(start uint) {
		for {
			l.index <- start
			start++
			if start > 238327 { // ZZZ in base 62
				start = 0
			}
		}
	}(0)

	return l
}

// GetConnection implements zzk.Listener
func (l *VirtualIPListener) SetConnection(conn client.Connection) { l.conn = conn }

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

// PostProcess implements zzk.Listener
func (l *VirtualIPListener) PostProcess(p map[string]struct{}) {}

// Spawn implements zzk.Listener
func (l *VirtualIPListener) Spawn(shutdown <-chan interface{}, ip string) {
	// ensure that the retry sentinel has good initial state
	if l.retry == nil {
		l.retry = make(map[string]int)
	}
	if _, ok := l.retry[ip]; !ok {
		l.retry[ip] = maxRetries
	}

	// Check if this ip has exceeded the number of retries for this host
	if l.retry[ip] > maxRetries {
		glog.Warningf("Throttling acquisition of %s for %s", ip, l.hostID)
		select {
		case <-time.After(waitTimeout):
		case <-shutdown:
			return
		}
	}

	glog.V(2).Infof("Host %s waiting to acquire virtual ip %s", l.hostID, ip)
	// Try to take lead on the path
	leader, err := l.conn.NewLeader(l.GetPath(ip))
	if err != nil {
		glog.Errorf("Could not initialize leader node for ip %s: %s", ip, err)
		return
	}
	hlnode := zzk.HostLeader{
		HostID: l.hostID,
	}
	leaderDone := make(chan struct{})
	defer close(leaderDone)
	_, err = leader.TakeLead(&hlnode, leaderDone)
	if err != nil {
		glog.Errorf("Error while trying to acquire a lock for %s: %s", ip, err)
		return
	}
	defer l.stopInstances(ip)
	defer leader.ReleaseLead()

	select {
	case <-shutdown:
		return
	default:
	}

	// Check if the path still exists
	if exists, err := zzk.PathExists(l.conn, l.GetPath(ip)); err != nil {
		glog.Errorf("Error while checking ip %s: %s", ip, err)
		return
	} else if !exists {
		return
	}

	index := l.getIndex()
	done := make(chan struct{})
	defer func(channel *chan struct{}) { close(*channel) }(&done)
	for {
		var vip pool.VirtualIP
		event, err := l.conn.GetW(l.GetPath(ip), &VirtualIPNode{VirtualIP: &vip}, done)
		if err == client.ErrEmptyNode {
			glog.Errorf("Deleting empty node for ip %s", ip)
			RemoveVirtualIP(l.conn, ip)
			return
		} else if err != nil {
			glog.Errorf("Could not load virtual ip %s: %s", ip, err)
			return
		}

		glog.V(2).Infof("Host %s binding to %s", l.hostID, ip)
		rebind, err := l.bind(&vip, index)
		if err != nil {
			glog.Errorf("Could not bind to virtual ip %s: %s", ip, err)
			l.retry[ip]++
			return
		}

		if l.retry[ip] > 0 {
			l.retry[ip]--
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
			glog.V(4).Infof("virtual ip listener for %s received event: %v", ip, e)
		case <-rebind:
			// If the primary virtual IP is removed, all other virtual IPs on
			// that subnet are removed.  This is in place to restore the
			// virtual IPs that were removed soley by the removal of the
			// primary virtual IP.
			glog.V(2).Infof("Host %s rebinding to %s", l.hostID, ip)
		case <-shutdown:
			if err := l.unbind(ip); err != nil {
				glog.Errorf("Could not unbind to virtual ip %s: %s", ip, err)
			}
			return
		}

		close(done)
		done = make(chan struct{})
	}
}

func (l *VirtualIPListener) getIndex() uint {
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

func (l *VirtualIPListener) bind(vip *pool.VirtualIP, index uint) (<-chan bool, error) {
	vmap, err := l.handler.VirtualInterfaceMap(virtualInterfacePrefix)
	if err != nil {
		return nil, err
	}

	if _, ok := vmap[vip.IP]; !ok {
		if vip.BindInterface == "" {
			return nil, ErrInvalidVirtualIP
		}
		postfix := fmt.Sprintf("%03s", utils.Base62(index))
		vname := fmt.Sprintf("%s%s%s", vip.BindInterface, virtualInterfacePrefix, postfix)
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

func (l *VirtualIPListener) stopInstances(ip string) {
	glog.Infof("Stopping service instances using ip %s on host %s", ip, l.hostID)
	rss, err := zkservice.LoadRunningServicesByHost(l.conn, l.hostID)
	if err != nil {
		glog.Errorf("Could not load running instances on host %s: %s", l.hostID, err)
		return
	}
	for _, rs := range rss {
		if rs.IPAddress == ip {
			if err := zkservice.StopServiceInstance(l.conn, l.hostID, rs.ID); err != nil {
				glog.Warningf("Could not stop service instance %s on host %s: %s", rs.ID, l.hostID, err)
			}
		}
	}
}

func SyncVirtualIPs(conn client.Connection, virtualIPs []pool.VirtualIP) error {
	nodes := make([]zzk.Node, len(virtualIPs))
	for i := range virtualIPs {
		nodes[i] = &VirtualIPNode{VirtualIP: &virtualIPs[i]}
	}
	return zzk.Sync(conn, nodes, vippath())
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
	err := conn.Delete(vippath(ip))
	if err == nil || err == client.ErrNoNode {
		return nil
	}
	return err
}

func GetHostID(conn client.Connection, ip string) (string, error) {
	leader, err := conn.NewLeader(vippath(ip))
	if err != nil {
		return "", err
	}
	return zzk.GetHostID(leader)
}
