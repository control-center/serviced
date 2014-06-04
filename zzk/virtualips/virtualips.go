package virtualips

import (
	"fmt"
	"os/exec"
	"path"
	"strconv"
	"strings"

	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/coordinator/client"
	"github.com/zenoss/serviced/domain/pool"
	"github.com/zenoss/serviced/utils"
)

const (
	zkVirtualIPs           = "/VirtualIPs"
	virtualInterfacePrefix = ":zvip"
)

func virtualIPsPath(nodes ...string) string {
	p := []string{zkVirtualIPs}
	p = append(p, nodes...)
	return path.Join(p...)
}

/*
RemoveAllVirtualIPs removes (unbinds) all virtual IPs on the agent
*/
func RemoveAllVirtualIPs() error {
	// confirm the virtual IP is on this host and remove it
	err, interfaceMap := createVirtualInterfaceMap()
	if err != nil {
		glog.Warningf("Creating virtual interface map failed")
		return err
	}
	glog.V(2).Infof("Removing all virtual IPs...")
	for virtualInterface, _ := range interfaceMap {
		if err := unbindVirtualIP(virtualInterface); err != nil {
			return err
		}
	}
	glog.V(2).Infof("All virtual IPs have been removed.")
	return nil
}

/*
SyncVirtualIPs is responsible for monitoring the virtual IPs in the model
 if a new virtual IP is added, create a zookeeper node corresponding to the new virtual IP
 if a virtual IP is removed, remove the zookeeper node corresponding to that virtual IP
*/
func SyncVirtualIPs(conn client.Connection, virtualIPs []pool.VirtualIP) error {
	glog.V(10).Infof("    start SyncVirtualIPs: VirtualIPs: %v", virtualIPs)
	defer glog.V(10).Info("    end SyncVirtualIPs")
	// create root VirtualIPs node if it does not exists
	exists, err := conn.Exists(virtualIPsPath())
	if err != nil {
		glog.Errorf("conn.Exists failed: %v (attempting to check %v)", err, virtualIPsPath())
		return err
	}
	if !exists {
		conn.CreateDir(virtualIPsPath())
		glog.Infof("Syncing virtual IPs... Created %v dir in zookeeper", virtualIPsPath())
	}

	// add nodes into zookeeper if the corresponding virtual IP is new to the model
	for _, virtualIP := range virtualIPs {
		currentVirtualIPNodePath := virtualIPsPath(virtualIP.IP)
		exists, err := conn.Exists(currentVirtualIPNodePath)
		if err != nil {
			glog.Errorf("conn.Exists failed: %v (attempting to check %v)", err, currentVirtualIPNodePath)
			return err
		}
		if !exists {
			// creating node in zookeeper for this virtual IP
			// the HostID is not yet known as the virtual IP is not on a host yet
			vipNode := virtualIPNode{HostID: "", VirtualIP: virtualIP}
			conn.Create(currentVirtualIPNodePath, &vipNode)
			glog.Infof("Syncing virtual IPs... Created %v in zookeeper", currentVirtualIPNodePath)
		}
	}

	// remove nodes from zookeeper if the corresponding virtual IP has been removed from the model
	children, err := conn.Children(virtualIPsPath())
	if err != nil {
		return err
	}
	for _, child := range children {
		removedVirtualIP := true
		for _, virtualIP := range virtualIPs {
			if child == virtualIP.IP {
				removedVirtualIP = false
				break
			}
		}
		if removedVirtualIP {
			// remove virtual IP from zookeeper
			nodeToDeletePath := virtualIPsPath(child)
			if err := conn.Delete(nodeToDeletePath); err != nil {
				glog.Errorf("conn.Delete failed:%v (attempting to delete %v))", err, nodeToDeletePath)
				return err
			}
			glog.Infof("Syncing virtual IPs... Removed %v from zookeeper", nodeToDeletePath)
		}
	}
	return nil
}

/*
WatchVirtualIPs monitors the virtual IP nodes in zookeeper, the "leader" agent (the agent that has a lock on the virtual IP),
   binds the virtual IP to the bind address specified by the virtual IP on itself
*/
func WatchVirtualIPs(conn client.Connection) {
	processing := make(map[string]chan int)
	sDone := make(chan string)

	// When this function exits, ensure that any started goroutines get
	// a signal to shutdown
	defer func() {
		glog.Info("Shutting down virtual IP child goroutines")
		for key, shutdown := range processing {
			glog.Info("Sending shutdown signal for ", key)
			shutdown <- 1
		}
	}()

	// Make the path if it doesn't exist
	if exists, err := conn.Exists(virtualIPsPath()); err != nil && err != client.ErrNoNode {
		glog.Errorf("Error checking path %s: %s", virtualIPsPath(), err)
		return
	} else if !exists {
		if err := conn.CreateDir(virtualIPsPath()); err != nil {
			glog.Errorf("Could not create path %s: %s", virtualIPsPath(), err)
			return
		}
	}

	// remove all virtual IPs that may be present before starting the loop
	if err := RemoveAllVirtualIPs(); err != nil {
		glog.Errorf("RemoveAllVirtualIPs failed: %v", err)
		return
	}

	var oldVirtualIPNodeIDs []string
	var currentVirtualIPNodeIDs []string // these are virtual IP addresses
	var virtualIPsNodeEvent <-chan client.Event
	var err error

	virtualInterfaceIndex := 0

	for {
		glog.Infof("Agent watching for changes to node: %v", virtualIPsPath())

		// deep copy currentVirtualIPNodeIDs into oldVirtualIPNodeIDs
		oldVirtualIPNodeIDs = nil
		for _, virtualIPAddress := range currentVirtualIPNodeIDs {
			oldVirtualIPNodeIDs = append(oldVirtualIPNodeIDs, virtualIPAddress)
		}

		currentVirtualIPNodeIDs, virtualIPsNodeEvent, err = conn.ChildrenW(virtualIPsPath())
		if err != nil {
			glog.Errorf("Agent unable to find any virtual IPs: %s", err)
			return
		}

		removedVirtualIPAddresses := setSubtract(oldVirtualIPNodeIDs, currentVirtualIPNodeIDs)
		for _, virtualIPAddress := range removedVirtualIPAddresses {
			if processing[virtualIPAddress] != nil {
				glog.Infof("A goroutine for %v is still running...", virtualIPAddress)
				exists, err := conn.Exists(virtualIPsPath(virtualIPAddress))
				if err != nil {
					glog.Errorf("conn.Exists failed: %v (attempting to check %v)", err, virtualIPsPath())
					return
				}
				if !exists {
					glog.Infof("node %v no longer exists, stopping corresponding goroutine...", virtualIPAddress)
					// this VIP node has been deleted from zookeeper
					// Remove the VIP from the host
					if err := removeVirtualIP(virtualIPAddress); err != nil {
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

		addedVirtualIPAddresses := setSubtract(currentVirtualIPNodeIDs, oldVirtualIPNodeIDs)
		for _, virtualIPAddress := range addedVirtualIPAddresses {
			if processing[virtualIPAddress] == nil {
				glog.V(2).Infof("Agent starting goroutine to watch VIP: %v", virtualIPAddress)
				virtualIPChannel := make(chan int)
				processing[virtualIPAddress] = virtualIPChannel
				myVirtualIP := pool.VirtualIP{PoolID: "", IP: virtualIPAddress, Netmask: "", BindInterface: ""}
				vipNode := virtualIPNode{HostID: "", VirtualIP: myVirtualIP}
				if err := conn.Get(virtualIPsPath(virtualIPAddress), &vipNode); err != nil {
					glog.Warningf("Unable to retrieve node: %v", virtualIPsPath(virtualIPAddress))
				} else {
					// kick off a watcher for this virtual IP
					go watchVirtualIP(virtualIPChannel, sDone, myVirtualIP, conn, virtualInterfaceIndex)
					virtualInterfaceIndex = virtualInterfaceIndex + 1
				}
			} else {
				glog.Warningf("Newly added virtual IP address: %v already has a goroutine running to monitor it?", virtualIPAddress)
			}
		}

		select {
		// something has changed on the head virtual IP node
		case evt := <-virtualIPsNodeEvent:
			glog.Infof("%v event: %v", virtualIPsPath(), evt)

		// a child goroutine has stopped
		case virtualIPAddress := <-sDone:
			glog.Info("Cleaning up for virtual IP: ", virtualIPAddress)
			delete(processing, virtualIPAddress)
		}
	}
}

type virtualIPNode struct {
	HostID    string
	VirtualIP pool.VirtualIP
	version   interface{}
}

func (v *virtualIPNode) Version() interface{}           { return v.version }
func (v *virtualIPNode) SetVersion(version interface{}) { v.version = version }

/*
GetVirtualIPHostID is used to figure out which host a virtual IP is configureds
*/
func GetVirtualIPHostID(conn client.Connection, virtualIPAddress string, hostID *string) error {
	myVirtualIP := pool.VirtualIP{PoolID: "", IP: "", Netmask: "", BindInterface: ""}
	vipNode := virtualIPNode{HostID: "", VirtualIP: myVirtualIP}
	if err := conn.Get(virtualIPsPath(virtualIPAddress), &vipNode); err != nil {
		glog.Warningf("Unable to retrieve node: %v (perhaps no agent has had time to acquire the virtual IP lock and realize the virtual interface)", virtualIPsPath(virtualIPAddress))
		return err
	}

	if vipNode.HostID == "" {
		return fmt.Errorf("Virtual IP: %v has been locked by an agent but has either not yet been configured or failed configuration. (the bind address may be invalid)", myVirtualIP.IP)
	}

	*hostID = vipNode.HostID

	return nil
}

/*
watchVirtualIP is invoked per virtual IP. It attempts to acquire a lock on the virtual IP.
If the lock is acquired, then virtual IP is realized on the agent.
*/
func watchVirtualIP(shutdown <-chan int, done chan<- string, watchingVirtualIP pool.VirtualIP, conn client.Connection, virtualInterfaceIndex int) {
	glog.V(2).Infof(" ### Started watchingVirtualIP: %v", watchingVirtualIP.IP)

	hostID, err := utils.HostID()
	if err != nil {
		glog.Errorf("Could not get host ID: %v", err)
		return
	}

	// try to lock
	vipOwnerNode := &virtualIPNode{HostID: "", VirtualIP: watchingVirtualIP}
	vipOwner := conn.NewLeader(virtualIPsPath(watchingVirtualIP.IP), vipOwnerNode)
	vipOwnerResponse := make(chan error)

	defer func() {
		glog.V(2).Infof(" ### Exiting watchingVirtualIP: %v", watchingVirtualIP.IP)
		done <- watchingVirtualIP.IP
	}()

	go func() {
		_, err := vipOwner.TakeLead()
		vipOwnerResponse <- err
	}()

	for {
		select {
		case err = <-vipOwnerResponse:
			if err != nil {
				glog.Errorf("Error in attempting to secure a lock on %v: %v", virtualIPsPath(watchingVirtualIP.IP), err)
			} else {
				// the lock has been secured
				glog.Infof("Configuring virtual IP address: %v on %v", virtualIPsPath(watchingVirtualIP.IP), hostID)
				if err := addVirtualIP(watchingVirtualIP, virtualInterfaceIndex); err != nil {
					glog.Errorf("Failed to configure virtual IP %v: %v", watchingVirtualIP.IP, err)
					break
				}

				// the virtual IP has successfully been configured on this agent!
				vipOwnerNode.HostID = hostID

				// set the HostID to the zookeeper node
				if err := conn.Set(virtualIPsPath(watchingVirtualIP.IP), vipOwnerNode); err != nil {
					glog.Errorf("Failed to set the HostID: %v to node: %v", vipOwnerNode.HostID, virtualIPsPath(watchingVirtualIP.IP))
				}
				glog.Infof("Virtual IP address: %v has been configured on %v", virtualIPsPath(watchingVirtualIP.IP), hostID)
			}

		// agent stopping
		case <-shutdown:
			glog.Infof("Agent stopped virtual IP: %v", virtualIPsPath(watchingVirtualIP.IP))
			return
		}
	}
}

// literally performs a set subtract
func setSubtract(a []string, b []string) []string {
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

// return the name of the interface for the virtual IP
// BINDADDRESS:zvipINDEX (zvip is defined by constant 'virtualInterfacePrefix')
func generateInterfaceName(virtualIP pool.VirtualIP, virtualInterfaceIndex int) (string, error) {
	if virtualIP.BindInterface == "" {
		return "", fmt.Errorf("generateInterfaceName failed as virtual IP: %v has no Bind Interface.", virtualIP.IP)
	}
	return virtualIP.BindInterface + virtualInterfacePrefix + strconv.Itoa(virtualInterfaceIndex), nil
}

// create an interface map of virtual interfaces configured on the agent
func createVirtualInterfaceMap() (error, map[string]pool.VirtualIP) {
	interfaceMap := make(map[string]pool.VirtualIP)

	//ifconfig | awk '/zvip/{print $1}'
	virtualInterfaceNames, err := exec.Command("bash", "-c", "ifconfig | awk '/"+virtualInterfacePrefix+"/{print $1}'").CombinedOutput()
	if err != nil {
		glog.Warningf("Determining virtual interfaces failed: %v --- %v", virtualInterfaceNames, err)
		return err, interfaceMap
	}
	glog.V(2).Infof("Control plane virtual interfaces: %v", string(virtualInterfaceNames))

	for _, virtualInterfaceName := range strings.Fields(string(virtualInterfaceNames)) {
		virtualInterfaceName = strings.TrimSpace(virtualInterfaceName)
		// ifconfig eth0 | awk '/inet addr:/{print $2}' | cut -d: -f2
		// 10.87.110.175
		virtualIP, err := exec.Command("bash", "-c", "ifconfig "+virtualInterfaceName+" | awk '/inet addr:/{print $2}' | cut -d: -f2").CombinedOutput()
		if err != nil {
			glog.Warningf("Determining IP address of interface %v failed: %v --- %v", virtualInterfaceName, virtualIP, err)
			return err, interfaceMap
		}
		bindInterfaceAndIndex := strings.Split(virtualInterfaceName, virtualInterfacePrefix)
		if len(bindInterfaceAndIndex) != 2 {
			err := fmt.Errorf("Unexpected interface format: %v", bindInterfaceAndIndex)
			return err, interfaceMap
		}
		bindInterface := strings.TrimSpace(string(bindInterfaceAndIndex[0]))
		interfaceMap[virtualInterfaceName] = pool.VirtualIP{PoolID: "", IP: strings.TrimSpace(string(virtualIP)), Netmask: "", BindInterface: bindInterface}
	}

	return nil, interfaceMap
}

// add (bind) a virtual IP on the agent
func addVirtualIP(virtualIPToAdd pool.VirtualIP, virtualInterfaceIndex int) error {
	// confirm the virtual IP is not already on this host
	virtualIPAlreadyHere := false
	err, interfaceMap := createVirtualInterfaceMap()
	if err != nil {
		glog.Warningf("Creating virtual interface map failed")
		return err
	}
	for _, virtualIP := range interfaceMap {
		if virtualIPToAdd.IP == virtualIP.IP {
			virtualIPAlreadyHere = true
		}
	}
	if virtualIPAlreadyHere {
		return fmt.Errorf("Requested virtual IP: %v is already on this host.", virtualIPToAdd.IP)
	}

	virtualInterfaceName, err := generateInterfaceName(virtualIPToAdd, virtualInterfaceIndex)
	if err != nil {
		return err
	}

	if err := bindVirtualIP(virtualIPToAdd, virtualInterfaceName); err != nil {
		return err
	}

	return nil
}

// remove (unbind) a virtual IP from the agent
func removeVirtualIP(virtualIPAddress string) error {
	// confirm the VIP is on this host and remove it
	err, interfaceMap := createVirtualInterfaceMap()
	if err != nil {
		glog.Warningf("Creating virtual interface map failed")
		return err
	}
	for virtualInterface, virtualIP := range interfaceMap {
		if virtualIPAddress == virtualIP.IP {
			if err := unbindVirtualIP(virtualInterface); err != nil {
				return err
			}
			return nil
		}
	}

	glog.Warningf("Requested virtual IP address: %v is not on this host.", virtualIPAddress)
	return nil
}

// bind the virtual IP to the agent
func bindVirtualIP(virtualIP pool.VirtualIP, virtualInterfaceName string) error {
	glog.Infof("Adding: %v", virtualIP)
	// ensure that the Bind Address is reported by ifconfig ... ?
	if err := exec.Command("ifconfig", virtualIP.BindInterface).Run(); err != nil {
		return fmt.Errorf("Problem with BindInterface %s", virtualIP.BindInterface)
	}

	// ADD THE VIRTUAL INTERFACE
	// sudo ifconfig eth0:1 inet 192.168.1.136 netmask 255.255.255.0
	if err := exec.Command("ifconfig", virtualInterfaceName, "inet", virtualIP.IP, "netmask", virtualIP.Netmask).Run(); err != nil {
		return fmt.Errorf("Problem with creating virtual interface %s", virtualInterfaceName)
	}

	glog.Infof("Added virtual interface/IP: %v (%v)", virtualInterfaceName, virtualIP)
	return nil
}

// unbind the virtual IP from the agent
func unbindVirtualIP(virtualInterface string) error {
	glog.Infof("Removing: %v", virtualInterface)
	// ifconfig eth0:1 down
	if err := exec.Command("ifconfig", virtualInterface, "down").Run(); err != nil {
		return fmt.Errorf("Problem with removing virtual interface %v: %v", virtualInterface, err)
	}

	glog.Infof("Removed virtual interface: %v", virtualInterface)
	return nil
}
