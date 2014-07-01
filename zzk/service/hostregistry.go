package service

import (
	"path"

	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/coordinator/client"
	"github.com/zenoss/serviced/domain/host"
)

const (
	zkRegistry = "/registry"
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
	conn    client.Connection
	hostmap map[string]*host.Host
}

// Listen listens for changes to /registry/hosts and updates the host list
// accordingly
func (l *HostRegistryListener) Listen(shutdown <-chan interface{}) {
	// create the path
	regpath := hostregpath()
	if exists, err := l.conn.Exists(regpath); err != nil {
		glog.Errorf("Error checking path %s: %s", regpath, err)
		return
	} else if exists {
		//pass
	} else if l.conn.CreateDir(regpath); err != nil {
		glog.Errorf("Error creating path %s: %s", regpath, err)
		return
	}

	for {
		ehosts, event, err := l.conn.ChildrenW(regpath)
		if err != nil {
			glog.Errorf("Could not watch host registry: %s", err)
			return
		}

		l.sync(ehosts)
		select {
		case <-event:
			glog.V(2).Info("Received host registry event: ", event)
		case <-shutdown:
			glog.V(2).Info("Host registry received signal to shutdown")
			return
		}
	}
}

func (l *HostRegistryListener) sync(ehosts []string) {
	unsynced := make(map[string]*host.Host)
	for _, id := range ehosts {
		var node HostNode
		if err := l.conn.Get(hostregpath(id), &node); err != nil {
			glog.Error("Error trying to get host registry information for node ", id)
			return
		}
		unsynced[id] = node.Host
	}

	for id, _ := range l.hostmap {
		if host, ok := unsynced[id]; ok {
			delete(unsynced, id)
			l.hostmap[id] = host
		} else {
			l.unregister(id, host)
		}
	}

	for id, host := range unsynced {
		l.register(id, host)
	}
}

func (l *HostRegistryListener) register(id string, host *host.Host) {
	if exists, err := l.conn.Exists(hostpath(host.ID)); err != nil {
		glog.Errorf("Could not look up host %s: %s", host.ID, err)
	} else if !exists {
		glog.Errorf("Host %s not initialized", host.ID)
	} else {
		l.hostmap[id] = host
	}
}

func (l *HostRegistryListener) unregister(id string, host *host.Host) {
	ssids, err := l.conn.Children(hostpath(host.ID))
	if err != nil {
		glog.Errorf("Could not get states on host %s: %s", host.ID, err)
		return
	}

	for _, ssid := range ssids {
		if err := removeInstance(l.conn, host.ID, ssid); err != nil {
			glog.Errorf("Could not remove instance %s: %s", ssid, err)
			return
		}
	}

	delete(l.hostmap, id)
}

func (l *HostRegistryListener) GetHosts() (hosts []*host.Host) {
	for _, host := range l.hostmap {
		hosts = append(hosts, host)
	}
	return hosts
}