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
	ErrHostNotInitialized   = errors.New("host not initialized")
	ErrHostInvalid          = errors.New("invalid host")
	ErrHostRegistryShutdown = errors.New("host registry shut down")
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
	hostmap  map[string]string
	shutdown <-chan interface{}
	alertC   chan<- bool // for testing only
}

// NewHostRegistryListener instantiates a new HostRegistryListener
func NewHostRegistryListener(conn client.Connection) *HostRegistryListener {
	return &HostRegistryListener{
		conn:    conn,
		hostmap: make(map[string]string),
	}
}

// Listen listens for changes to /registry/hosts and updates the host list
// accordingly
func (l *HostRegistryListener) Listen(shutdown <-chan interface{}) {
	_shutdown := make(chan interface{})
	l.shutdown = _shutdown
	defer close(_shutdown)

	// create the path
	regpath := hostregpath()
	if exists, err := zkutils.PathExists(l.conn, regpath); err != nil {
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
	unsynced := make(map[string]string)
	for _, id := range ehosts {
		var node HostNode
		if err := l.conn.Get(hostregpath(id), &node); err != nil {
			glog.Error("Error trying to get host registry information for node ", id)
			return
		}
		unsynced[id] = node.Host.ID
	}

	for id, _ := range l.hostmap {
		if _, ok := unsynced[id]; ok {
			delete(unsynced, id)
		} else {
			if err := l.unregister(id); err != nil {
				glog.Warningf("Could not unregister %s: %s", id, err)
			}
		}
	}

	for id, hostID := range unsynced {
		if err := l.register(id, hostID); err != nil {
			glog.Warningf("Could not register host %s (%s): %s", hostID, id, err)
		}
	}
}

func (l *HostRegistryListener) register(id string, hostID string) error {
	// verify that there is a running listener for that host
	if exists, err := zkutils.PathExists(l.conn, hostpath(hostID)); err != nil {
		return err
	} else if !exists {
		return ErrHostNotInitialized
	}

	l.hostmap[id] = hostID
	l.alert()
	return nil
}

func (l *HostRegistryListener) unregister(id string) error {
	defer func() {
		delete(l.hostmap, id)
		l.alert()
	}()

	// remove all the instances running on that host
	hostID := l.hostmap[id]
	if exists, err := zkutils.PathExists(l.conn, hostpath(hostID)); err != nil {
		return err
	} else if !exists {
		return nil
	}

	ssids, err := l.conn.Children(hostpath(hostID))
	if err != nil {
		return err
	}
	for _, ssid := range ssids {
		if err := removeInstance(l.conn, hostID, ssid); err != nil {
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

		if len(ehosts) == 0 {
			select {
			case <-eventW:
				// pass
			case <-l.shutdown:
				return nil, ErrHostRegistryShutdown
			}
		} else {
			break
		}
	}

	hosts = make([]*host.Host, len(ehosts))
	for i, ehostID := range ehosts {
		var host host.Host
		if err := l.conn.Get(hostregpath(ehostID), &HostNode{Host: &host}); err != nil {
			return nil, err
		}
		hosts[i] = &host
	}

	return hosts, nil
}

func registerHost(conn client.Connection, host *host.Host) (string, error) {
	if host == nil || host.ID == "" {
		return "", ErrHostInvalid
	}

	// verify that a listener has been initialized
	if exists, err := conn.Exists(hostpath(host.ID)); err != nil {
		return "", err
	} else if !exists {
		return "", ErrHostNotInitialized
	}

	// create the ephemeral host
	return conn.CreateEphemeral(hostregpath(host.ID), &HostNode{Host: host})
}