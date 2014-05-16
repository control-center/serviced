package virtualips

import (
	"errors"
	"fmt"
	"os/exec"
	"path"
	"strconv"
	"strings"

	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/coordinator/client"
	"github.com/zenoss/serviced/datastore"
	"github.com/zenoss/serviced/domain/pool"
	"github.com/zenoss/serviced/facade"
	"github.com/zenoss/serviced/utils"
)

const (
	zkVirtualIPs             = "/VIPs"
	VIRTUAL_INTERFACE_PREFIX = ":cpvip"
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

func GenerateInterfaceName(virtualIP pool.VirtualIP) (string, error) {
	if virtualIP.BindInterface == "" {
		msg := fmt.Sprintf("GenerateInterfaceName failed as virtual IP: %v has no Bind Interface.", virtualIP.IP)
		return "", errors.New(msg)
	}
	if virtualIP.InterfaceIndex == "" {
		msg := fmt.Sprintf("GenerateInterfaceName failed as Virtual IP: %v has no Index.", virtualIP.IP)
		return "", errors.New(msg)
	}
	return virtualIP.BindInterface + VIRTUAL_INTERFACE_PREFIX + virtualIP.InterfaceIndex, nil
}

func DetermineVirtualInterfaceIndex(virtualIPToAdd pool.VirtualIP, interfaceMap []pool.VirtualIP) (string, error) {
	MAX_INDEX := 100
	interfaceIndex := 0

	for interfaceIndex = 0; interfaceIndex < MAX_INDEX; interfaceIndex++ {
		virtualIPToAdd.InterfaceIndex = strconv.Itoa(interfaceIndex)
		proposedInterfaceName, err := GenerateInterfaceName(virtualIPToAdd)
		if err != nil {
			return "", err
		}

		proposedInterfaceNameIsAcceptable := true
		for _, currentVirtualInterface := range interfaceMap {
			currentInterfaceName, err := GenerateInterfaceName(currentVirtualInterface)
			if err != nil {
				return "", err
			}
			glog.Infof(" ########## checking %v === %v", proposedInterfaceName, currentInterfaceName)
			if proposedInterfaceName == currentInterfaceName {
				glog.Warningf("Proposed interface name: %v is already taken...", proposedInterfaceName)
				proposedInterfaceNameIsAcceptable = false
				break
			}
		}
		if proposedInterfaceNameIsAcceptable {
			// found an open interface index!
			return strconv.Itoa(interfaceIndex), nil
		}
	}

	// MAX_INDEX
	msg := fmt.Sprintf("There are over %v virtual IP interfaces. Could not generate index.", MAX_INDEX)
	return "", errors.New(msg)
}

func CreateVirtualInterfaceMap() (error, []pool.VirtualIP) {
	glog.Info("Creating Virtual Interface Map...")
	interfaceMap := []pool.VirtualIP{}

	//ifconfig | awk '/cpvip/{print $1}'
	virtualInterfaceNames, err := exec.Command("bash", "-c", "ifconfig | awk '/cpvip/{print $1}'").CombinedOutput()
	if err != nil {
		glog.Warningf("Determining virtual interfaces failed: %v --- %v", virtualInterfaceNames, err)
		return err, interfaceMap
	}
	glog.Infof("Control plane virtual interfaces: %v", string(virtualInterfaceNames))

	for _, virtualInterfaceName := range strings.Fields(string(virtualInterfaceNames)) {
		virtualInterfaceName = strings.TrimSpace(virtualInterfaceName)
		// ifconfig eth0 | awk '/inet addr:/{print $2}' | cut -d: -f2
		// 10.87.110.175
		virtualIP, err := exec.Command("bash", "-c", "ifconfig "+virtualInterfaceName+" | awk '/inet addr:/{print $2}' | cut -d: -f2").CombinedOutput()
		if err != nil {
			glog.Warningf("Determining IP address of interface %v failed: %v --- %v", virtualInterfaceName, virtualIP, err)
			return err, interfaceMap
		}
		bindInterfaceAndIndex := strings.Split(virtualInterfaceName, VIRTUAL_INTERFACE_PREFIX)
		if len(bindInterfaceAndIndex) != 2 {
			err := fmt.Errorf("Unexpected interface format: %v", bindInterfaceAndIndex)
			return err, interfaceMap
		}
		bindInterface := strings.TrimSpace(string(bindInterfaceAndIndex[0]))
		interfaceIndex := strings.TrimSpace(string(bindInterfaceAndIndex[1]))
		interfaceMap = append(interfaceMap, pool.VirtualIP{PoolID: "", IP: strings.TrimSpace(string(virtualIP)), Netmask: "", BindInterface: bindInterface, InterfaceIndex: interfaceIndex})
	}

	glog.Infof(" ********** Virtual Interface Map: %v", interfaceMap)

	return nil, interfaceMap
}

func (h *Handler) GetMyPool() (*pool.ResourcePool, error) {
	hostID, err := utils.HostID()
	if err != nil {
		glog.Errorf("Could not get host ID: %v", err)
		return nil, err
	}

	myHost, err := h.facade.GetHost(h.context, hostID)
	if err != nil {
		glog.Errorf("Cannot retrieve host information for pool host %v", hostID)
		return nil, err
	}
	if myHost == nil {
		msg := fmt.Sprintf("Host: %v does not exist.", hostID)
		return nil, errors.New(msg)
	}

	myPool, err := h.facade.GetResourcePool(h.context, myHost.PoolID)
	if err != nil {
		glog.Errorf("Unable to load resource pool from host: %v", myHost.PoolID)
		return nil, err
	}

	return myPool, nil
}

func addVirtualIP(virtualIP pool.VirtualIP) error {
	glog.Infof("Adding: %v", virtualIP)
	// ensure that the Bind Address is reported by ifconfig ... ?
	if err := exec.Command("ifconfig", virtualIP.BindInterface).Run(); err != nil {
		msg := fmt.Sprintf("Problem with BindInterface %s", virtualIP.BindInterface)
		return errors.New(msg)
	}

	virtualInterfaceName, err := GenerateInterfaceName(virtualIP)
	if err != nil {
		return err
	}
	// ADD THE VIRTUAL INTERFACE
	// sudo ifconfig eth0:1 inet 192.168.1.136 netmask 255.255.255.0
	if err := exec.Command("ifconfig", virtualInterfaceName, "inet", virtualIP.IP, "netmask", virtualIP.Netmask).Run(); err != nil {
		msg := fmt.Sprintf("Problem with creating virtual interface %s", virtualInterfaceName)
		return errors.New(msg)
	}

	glog.Infof("Added: %v", virtualIP)
	return nil
}

func RemoveVirtualIP(virtualIP pool.VirtualIP) error {
	glog.Infof("Removing: %v", virtualIP)
	virtualInterfaceName, err := GenerateInterfaceName(virtualIP)
	if err != nil {
		return err
	}
	// ifconfig eth0:1 down
	if err := exec.Command("ifconfig", virtualInterfaceName, "down").Run(); err != nil {
		msg := fmt.Sprintf("Problem with removing virtual interface %v: %v", virtualInterfaceName, err)
		return errors.New(msg)
	}

	glog.Infof("Removed interface: %s %v", virtualInterfaceName, virtualIP)
	return nil
}

func (h *Handler) ConfigureVIP(vip string) error {
	myPool, err := h.GetMyPool()
	if err != nil {
		return err
	}

	// confirm that the VIP we are going to add is in this pool
	poolConfirmed := false
	myVirtualIP := pool.VirtualIP{}
	myVirtualIPPosition := 0
	for virtualIPPosition, virtualIP := range myPool.VirtualIPs {
		if vip == virtualIP.IP {
			myVirtualIP = virtualIP
			myVirtualIPPosition = virtualIPPosition
			poolConfirmed = true
			break
		}
	}
	if !poolConfirmed {
		msg := fmt.Sprintf("Requested vip: %v does not exist in pool: %v", vip, myPool.ID)
		return errors.New(msg)
	}

	// confirm the VIP is not already on this host
	vipPresent := false
	err, interfaceMap := CreateVirtualInterfaceMap()
	if err != nil {
		glog.Warningf("Creating virtual interface map failed")
		return err
	}
	for _, virtualInterface := range interfaceMap {
		if vip == virtualInterface.IP {
			vipPresent = true
		}
	}
	if vipPresent {
		msg := fmt.Sprintf("Requested vip: %v is already on this host.", vip)
		return errors.New(msg)
	}

	interfaceIndex, err := DetermineVirtualInterfaceIndex(myVirtualIP, interfaceMap)
	if err != nil {
		return err
	}
	myPool.VirtualIPs[myVirtualIPPosition].InterfaceIndex = interfaceIndex
	if err := h.facade.UpdateResourcePool(h.context, myPool); err != nil {
		return err
	}

	if err := addVirtualIP(myPool.VirtualIPs[myVirtualIPPosition]); err != nil {
		return err
	}

	glog.Infof("Virtual IP Address: %v has been configured on this host.", vip)
	return nil
}

func (h *Handler) RemoveVIP(virtualIPAddress string) error {
	myPool, err := h.GetMyPool()
	if err != nil {
		return err
	}

	// confirm that the VIP we are going to remove is no longer in this pool
	for _, virtualIP := range myPool.VirtualIPs {
		if virtualIPAddress == virtualIP.IP {
			msg := fmt.Sprintf("Requested virtual IP address: %v still exists in pool: %v", virtualIPAddress, myPool.ID)
			return errors.New(msg)
		}
	}

	// confirm the VIP is on this host and remove it
	err, interfaceMap := CreateVirtualInterfaceMap()
	if err != nil {
		glog.Warningf("Creating virtual interface map failed")
		return err
	}
	for _, virtualIP := range interfaceMap {
		if virtualIPAddress == virtualIP.IP {
			if err := RemoveVirtualIP(virtualIP); err != nil {
				return err
			}
			glog.Infof("Virtual IP address: %v has been removed from this host.", virtualIPAddress)
			return nil
		}
	}

	glog.Warningf("Requested virtual IP address: %v is not on this host.", virtualIPAddress)
	return nil
}

func SetSubtract(a []string, b []string) []string {
	difference := []string{}
	for _, aElement := range a {
		aElementFound := false
		for _, bElement := range b {
			if aElement == bElement {
				aElementFound = true
				break
			}
		}
		if !aElementFound {
			difference = append(difference, aElement)
		}
	}
	return difference
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

	var oldVirtualIPAddresses []string
	var currentVirtualIPAddresses []string
	var zkEvent <-chan client.Event
	var err error

	for {
		glog.Infof("Agent watching for changes to node: %v", virtualIPsPath())

		// deep copy currentVirtualIPAddresses into oldVirtualIPAddresses
		oldVirtualIPAddresses = nil
		for _, virtualIPAddress := range currentVirtualIPAddresses {
			oldVirtualIPAddresses = append(oldVirtualIPAddresses, virtualIPAddress)
		}

		currentVirtualIPAddresses, zkEvent, err = h.conn.ChildrenW(virtualIPsPath())
		if err != nil {
			glog.Errorf("Agent unable to find any virtual IPs: %s", err)
			return
		}

		removedVirtualIPAddresses := SetSubtract(oldVirtualIPAddresses, currentVirtualIPAddresses)
		if len(removedVirtualIPAddresses) > 0 {
			for _, virtualIPAddress := range removedVirtualIPAddresses {
				if processing[virtualIPAddress] != nil {
					glog.Infof("A goroutine for %v is still running...", virtualIPAddress)
					exists, err := h.conn.Exists(virtualIPsPath(virtualIPAddress))
					if err != nil {
						glog.Errorf("conn.Exists failed: %v (attempting to check %v)", err, virtualIPsPath())
						return
					}
					if !exists {
						glog.Infof("node %v no longer exists, stopping corresponding goroutine...", virtualIPAddress)
						// this VIP node has been deleted from zookeeper
						// Remove the VIP from the host
						if err := h.RemoveVIP(virtualIPAddress); err != nil {
							glog.Errorf("Failed to remove virtual IP %v: %v", virtualIPAddress, err)
						}
						// therefore, stop the go routine responsible for watching this particular VIP
						processing[virtualIPAddress] <- 1
					} else {
						glog.Warningf("node %v does not exists, although its goroutine does not", virtualIPAddress)
					}
				} else {
					glog.Warningf("Newly removed virtual IP address: %v does not have a goroutine running to monitor it?", virtualIPAddress)
				}
			}
		}

		addedVirtualIPAddresses := SetSubtract(currentVirtualIPAddresses, oldVirtualIPAddresses)
		if len(addedVirtualIPAddresses) > 0 {
			for _, virtualIPAddress := range addedVirtualIPAddresses {
				if processing[virtualIPAddress] == nil {
					glog.Infof("Agent starting goroutine to watch VIP: %v", virtualIPAddress)
					virtualIPChannel := make(chan int)
					processing[virtualIPAddress] = virtualIPChannel
					go h.WatchVirtualIP(virtualIPChannel, sDone, virtualIPAddress)
				} else {
					glog.Warningf("Newly added virtual IP address: %v already has a goroutine running to monitor it?", virtualIPAddress)
				}
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

type VIP struct {
	HostID  string
	VIP     string
	version int32
}

func (n *VIP) SetVersion(version int32) { n.version = version }
func (n *VIP) Version() int32           { return n.version }

func (h *Handler) WatchVirtualIP(shutdown <-chan int, done chan<- string, virtualIPAddress string) {
	glog.Infof(" ### Started WatchVirtualIP: %v", virtualIPAddress)

	hostID, err := utils.HostID()
	if err != nil {
		glog.Errorf("Could not get host ID: %v", err)
		return
	}
	vipOwnerNode := &VIP{
		HostID: hostID,
		VIP:    virtualIPAddress,
	}
	vipOwner := h.conn.NewLeader(virtualIPsPath(virtualIPAddress), vipOwnerNode)
	vipOwnerResponse := make(chan error)

	defer func() {
		glog.Infof(" ### Exiting WatchVirtualIP: %v", virtualIPAddress)
		done <- virtualIPAddress
	}()

	go func() {
		_, err := vipOwner.TakeLead()
		vipOwnerResponse <- err
	}()

	for {
		select {
		// the lock has been released?
		case err = <-vipOwnerResponse:
			if err != nil {
				glog.Errorf("Error in attempting to secure a lock on %v: %v", virtualIPsPath(virtualIPAddress), err)
			} else {
				glog.Infof(" +++++ I OWN THE LOCK FOR VIP: %v", virtualIPsPath(virtualIPAddress))
				if err := h.ConfigureVIP(virtualIPAddress); err != nil {
					glog.Errorf("Failed to configure VIP: %v", virtualIPAddress)
				}
			}

		// agent stopping
		case <-shutdown:
			glog.Infof("Agent stopped VIP %v", virtualIPsPath(virtualIPAddress))
			return
		}
	}
}

func (h *Handler) SyncVirtualIPs() error {
	glog.Info("started syncVirtualIPs")
	defer glog.Info("finished syncVirtualIPs")

	myPool, err := h.GetMyPool()
	if err != nil {
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

	for _, virtualIP := range myPool.VirtualIPs {
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
		for _, virtualIP := range myPool.VirtualIPs {
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
