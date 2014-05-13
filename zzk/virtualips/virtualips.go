package virtualips

import (
	"errors"
	"fmt"
	"path"

	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/coordinator/client"
	"github.com/zenoss/serviced/datastore"
	"github.com/zenoss/serviced/facade"
	"github.com/zenoss/serviced/utils"
)

const (
	zkVirtualIPs = "/VIPs"
)

func virtualIPsPath(nodes ...string) string {
	p := []string{zkVirtualIPs}
	p = append(p, nodes...)
	return path.Join(p...)
}

type Handler struct {
	facade  *facade.Facade
	conn    client.Connection
	context datastore.Context
}

// New starts a new event listener
func New(facade *facade.Facade, conn client.Connection, context datastore.Context) *Handler {
	return &Handler{facade: facade, conn: conn, context: context}
}

// Listen listens for changes on the event node and processes the snapshot
func (h *Handler) WatchVirtualIPs() {
	glog.Infof("started WatchVirtualIPs ... going to watch %v", virtualIPsPath())
	defer glog.Info("finished WatchVirtualIPs")

	processing := make(map[string]chan int)
	sDone := make(chan string)

	// When this function exits, ensure that any started goroutines get
	// a signal to shutdown
	defer func() {
		glog.Info("Shutting down child goroutines")
		for key, shutdown := range processing {
			glog.Info("Sending shutdown signal for ", key)
			shutdown <- 1
		}
	}()

	// Make the path if it doesn't exist
	if exists, err := h.conn.Exists(virtualIPsPath()); err != nil && err != client.ErrNoNode {
		glog.Errorf("Error checking path %s: %s", virtualIPsPath(), err)
		return
	} else if !exists {
		if err := h.conn.CreateDir(virtualIPsPath()); err != nil {
			glog.Errorf("Could not create path %s: %s", virtualIPsPath(), err)
			return
		}
	}

	for {
		glog.Info(" ----- Agent watching for changes to ", virtualIPsPath())
		virtualIPAddresses, zkEvent, err := h.conn.ChildrenW(virtualIPsPath())
		if err != nil {
			glog.Errorf("Agent unable to find any virtual IPs: %s", err)
			return
		}
		for _, virtualIPAddress := range virtualIPAddresses {
			if processing[virtualIPAddress] == nil {
				glog.Info("Agent starting goroutine to watch ", virtualIPAddress)
				virtualIPChannel := make(chan int)
				processing[virtualIPAddress] = virtualIPChannel
				go h.WatchVirtualIP(virtualIPChannel, sDone, virtualIPAddress)
			}
		}
		select {
		case evt := <-zkEvent:
			glog.Infof("%v event: %v", virtualIPsPath(), evt)
		case virtualIPAddress := <-sDone:
			glog.Info("Cleaning up for virtual IP: ", virtualIPAddress)
			delete(processing, virtualIPAddress)
		}
	}
}

func (h *Handler) WatchVirtualIP(shutdown <-chan int, done chan<- string, virtualIPAddress string) {
	glog.Info("started WatchVirtualIP")
	//defer glog.Info("finished WatchVirtualIP")
	defer func() {
		glog.V(3).Info("Exiting function WatchVirtualIP ", virtualIPAddress)
		done <- virtualIPAddress
	}()

	for {
		_, zkEvent, err := h.conn.ChildrenW(virtualIPsPath(virtualIPAddress))
		if err != nil {
			glog.Errorf("Agent unable to find any virtual IP %v: %v", virtualIPAddress, err)
			return
		}
		/*evt := <-zkEvent
		if evt.Type == client.EventNodeDeleted {
			glog.Info("Shutting down due to node delete")
			return
		}*/
		select {
		case evt := <-zkEvent:
			glog.Info("Shutting down due to node delete (something changed on this node: %v): %v", virtualIPAddress, evt)
			return
		}
	}
}

func (h *Handler) SyncVirtualIPs() error {
	glog.Info("started syncVirtualIPs")
	defer glog.Info("finished syncVirtualIPs")

	hostId, err := utils.HostID()
	if err != nil {
		glog.Errorf("Could not get host ID: %v", err)
		return err
	}

	myHost, err := h.facade.GetHost(h.context, hostId)
	if err != nil {
		glog.Errorf("Cannot retrieve host information for pool host %v", hostId)
		return err
	}
	if myHost == nil {
		msg := fmt.Sprintf("Host: %v does not exist.", hostId)
		return errors.New(msg)
	}

	aPool, err := h.facade.GetResourcePool(h.context, myHost.PoolID)
	if err != nil {
		glog.Errorf("Unable to load resource pool: %v", myHost.PoolID)
		return err
	}

	exists, err := h.conn.Exists(virtualIPsPath())
	if err != nil {
		glog.Errorf("conn.Exists failed: %v (attempting to check %v)", err, virtualIPsPath())
		return err
	}
	if !exists {
		h.conn.CreateDir(virtualIPsPath())
		glog.Infof("Syncing virtual IPs... Created %v dir in zookeeper", virtualIPsPath())
	}

	for _, virtualIP := range aPool.VirtualIPs {
		currentVirtualIPDir := virtualIPsPath(virtualIP.IP)
		exists, err := h.conn.Exists(currentVirtualIPDir)
		if err != nil {
			glog.Errorf("conn.Exists failed: %v (attempting to check %v)", err, currentVirtualIPDir)
			return err
		}
		if !exists {
			h.conn.CreateDir(currentVirtualIPDir)
			glog.Infof("Syncing virtual IPs... Created %v dir in zookeeper", currentVirtualIPDir)
		}
	}

	children, err := h.conn.Children(virtualIPsPath())
	if err != nil {
		return err
	}
	for _, child := range children {
		removeVirtualIP := true
		for _, virtualIP := range aPool.VirtualIPs {
			if child == virtualIP.IP {
				removeVirtualIP = false
				break
			}
		}
		if removeVirtualIP {
			nodeToDelete := virtualIPsPath(child)
			if err := h.conn.Delete(nodeToDelete); err != nil {
				glog.Errorf("conn.Delete failed:%v (attempting to delete %v))", err, nodeToDelete)
				return err
			}
			glog.Infof("Syncing virtual IPs... Removed %v dir from zookeeper", nodeToDelete)
		}
	}
	return nil
}
