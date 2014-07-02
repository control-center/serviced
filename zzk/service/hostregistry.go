package service

import (
	"errors"
	"path"

	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/coordinator/client"
	"github.com/zenoss/serviced/domain/host"
)

const (
	zkRegistry = "/registry"
)

var (
	ErrHostNotInitialized = errors.New("host not initialized")
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
	alertC  chan<- bool // for testing only
}

// NewHostRegistryListener instantiates a new HostRegistryListener
func NewHostRegistryListener(conn client.Connection) *HostRegistryListener {
	return &HostRegistryListener{
		conn:    conn,
		hostmap: make(map[string]*host.Host),
	}
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
			if err := l.unregister(id); err != nil {
				glog.Warningf("Could not unregister %s: %s", id, err)
			}
		}
	}

	for id, host := range unsynced {
		if err := l.register(id, host); err != nil {
			glog.Warningf("Could not register host %s (%s): %s", host.ID, id, err)
		}
	}
}

func (l *HostRegistryListener) register(id string, host *host.Host) error {
	// verify that there is a running listener for that host
	if exists, err := l.conn.Exists(hostpath(host.ID)); err != nil {
		return err
	} else if !exists {
		return ErrHostNotInitialized
	}

	l.hostmap[id] = host
	l.alert()
	return nil
}

func (l *HostRegistryListener) unregister(id string) error {
	defer func() {
		delete(l.hostmap, id)
		l.alert()
	}()

	// remove all the instances running on that host
	host := l.hostmap[id]
	if exists, err := l.conn.Exists(hostpath(host.ID)); err != nil {
		return err
	} else if !exists {
		return nil
	}

	ssids, err := l.conn.Children(hostpath(host.ID))
	if err != nil {
		return err
	}
	for _, ssid := range ssids {
		if err := removeInstance(l.conn, host.ID, ssid); err != nil {
			return err
		}
	}
	return nil
}

func (l *HostRegistryListener) alert() {
	if l.alertC != nil {
		l.alertC <- true
	}
}

func (l *HostRegistryListener) GetHosts() (hosts []*host.Host) {
	for _, host := range l.hostmap {
		hosts = append(hosts, host)
	}
	return hosts
}