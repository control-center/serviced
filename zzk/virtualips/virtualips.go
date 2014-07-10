package virtualips

import (
	"fmt"
	"net"
	"os/exec"
	"path"
	"strconv"
	"strings"

	"github.com/zenoss/glog"
	coordclient "github.com/zenoss/serviced/coordinator/client"
	"github.com/zenoss/serviced/domain/pool"
	"github.com/zenoss/serviced/utils"
	zkutils "github.com/zenoss/serviced/zzk/utils"
)

const (
	zkVirtualIPs           = "/virtualIPs"
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
	interfaceMap, err := createVirtualInterfaceMap()
	if err != nil {
		glog.Warningf("Creating virtual interface map failed")
		return err
	}
	glog.V(2).Infof("Removing all virtual IPs...")
	for _, virtualIP := range interfaceMap {
		if err := removeVirtualIP(virtualIP.IP); err != nil {
			return fmt.Errorf("removeVirtualIP failed: %v", err)
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
func SyncVirtualIPs(conn coordclient.Connection, virtualIPs []pool.VirtualIP) error {
	glog.V(10).Infof("    start SyncVirtualIPs: VirtualIPs: %v", virtualIPs)
	defer glog.V(10).Info("    end SyncVirtualIPs")

	if err := createNode(conn, virtualIPsPath()); err != nil {
		return err
	}

	// add nodes into zookeeper if the corresponding virtual IP is new to the model
	for _, virtualIP := range virtualIPs {
		currentVirtualIPNodePath := virtualIPsPath(virtualIP.IP)
		exists, err := zkutils.PathExists(conn, currentVirtualIPNodePath)
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

// TODO: need to build out this listener and write appropriate tests, but that
// will take a long time, so I am putting this wrapper here as a placeholder
// to mess with later

// VirtualIPListener is the listener object for watching the zk object for
// virtual IP nodes
type VirtualIPListener struct {
	conn coordclient.Connection
}

// NewVirtualIPListener instantiates a new VirtualIPListener object
func NewVirtualIPListener(conn coordclient.Connection) *VirtualIPListener {
	return &VirtualIPListener{conn}
}

// Listen observes changes on the VirtualIP zk node
func (l *VirtualIPListener) Listen(shutdown <-chan interface{}) {
	WatchVirtualIPs(l.conn, shutdown)
}

/*
WatchVirtualIPs monitors the virtual IP nodes in zookeeper, the "leader" agent (the agent that has a lock on the virtual IP),
   binds the virtual IP to the bind address specified by the virtual IP on itself
*/
func WatchVirtualIPs(conn coordclient.Connection, shutdown <-chan interface{}) {
	glog.Info("Watching all virtual IPs (will kick off a go routine per virtual IP)")
	processing := make(map[string]chan int)
	sDone := make(chan string)

	// When this function exits, ensure that any started goroutines get
	// a signal to shutdown
	defer func() {
		glog.Info("Shutting down virtual IP child goroutines")
		for virtualIPAddress, virtualIPShutdown := range processing {
			glog.Infof("Sending shutdown signal for: %v goroutine", virtualIPAddress)
			close(virtualIPShutdown)
		}
	}()

	if err := createNode(conn, virtualIPsPath()); err != nil {
		glog.Errorf("%v", err)
		return
	}

	// remove all virtual IPs that may be present before starting the loop
	if err := RemoveAllVirtualIPs(); err != nil {
		glog.Errorf("RemoveAllVirtualIPs failed: %v", err)
	}

	var oldVirtualIPNodeIDs []string
	var currentVirtualIPNodeIDs []string // these are virtual IP addresses
	var virtualIPsNodeEvent <-chan coordclient.Event
	var err error

	virtualInterfaceIndex := 0

	hostID, err := utils.HostID()
	if err != nil {
		glog.Errorf("HostID failed (must have a host ID to put a virtual IP on): %v", err)
		return
	}

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

		// remove the virtual IPs from the agent that have been removed from the model (VIP node removed from zookeeper)
		// stop the go routine responsible for watching that virtual IP
		removedVirtualIPAddresses := setSubtract(oldVirtualIPNodeIDs, currentVirtualIPNodeIDs)
		for _, virtualIPAddress := range removedVirtualIPAddresses {
			glog.Infof("node %v no longer exists, stopping corresponding goroutine...", virtualIPAddress)
			if err := removeVirtualIP(virtualIPAddress); err != nil {
				glog.Errorf("Failed to remove virtual IP %v: %v", virtualIPAddress, err)
			}
			// stop the go routine responsible for watching this particular VIP
			close(processing[virtualIPAddress])

			select {
			// hang out until the goroutine coupled to virtualIPAddress finishes...
			case virtualIPAddressGoRoutineStopped := <-sDone:
				if virtualIPAddressGoRoutineStopped != virtualIPAddress {
					glog.Errorf("Unexpected go routine stopped")
					break
				}

				delete(processing, virtualIPAddress)

				refreshVirtualIPAddresses := determineVirtualIPAddressesOnMe(conn, currentVirtualIPNodeIDs, hostID)
				for _, virtualIPAddress := range refreshVirtualIPAddresses {
					glog.V(2).Infof("Refresh virtual IP: %v", virtualIPAddress)
					processing[virtualIPAddress] <- 1
				}
			}

		}

		// add a VIP watchers which will configure the virtual IPs on the agent that have been added to the model (new VIP node in zookeeper)
		addedVirtualIPAddresses := setSubtract(currentVirtualIPNodeIDs, oldVirtualIPNodeIDs)
		for _, virtualIPAddress := range addedVirtualIPAddresses {
			glog.V(2).Infof("Agent starting goroutine to watch VIP: %v", virtualIPAddress)
			virtualIPChannel := make(chan int)
			processing[virtualIPAddress] = virtualIPChannel
			myVirtualIP := pool.VirtualIP{PoolID: "", IP: virtualIPAddress, Netmask: "", BindInterface: ""}
			vipNode := virtualIPNode{HostID: "", VirtualIP: myVirtualIP}
			if err := conn.Get(virtualIPsPath(virtualIPAddress), &vipNode); err != nil {
				glog.Warningf("Unable to retrieve node: %v", virtualIPsPath(virtualIPAddress))
			} else {
				// kick off a watcher for this virtual IP
				go watchVirtualIP(virtualIPChannel, sDone, vipNode.VirtualIP, conn, virtualInterfaceIndex, hostID)
				virtualInterfaceIndex = virtualInterfaceIndex + 1
			}
		}

		select {
		// something has changed on the head virtual IP node
		case evt := <-virtualIPsNodeEvent:
			glog.Infof("%v event: %v", virtualIPsPath(), evt)
		case <-shutdown:
			return
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
func GetVirtualIPHostID(conn coordclient.Connection, virtualIPAddress string, hostID *string) error {
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
func watchVirtualIP(request <-chan int, done chan<- string, watchingVirtualIP pool.VirtualIP, conn coordclient.Connection, virtualInterfaceIndex int, hostID string) {
	glog.V(2).Infof(" ### Started watchingVirtualIP: %v", watchingVirtualIP.IP)

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
		case err := <-vipOwnerResponse:
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
		case _, open := <-request:
			if !open {
				// closed
				glog.Infof("Agent stopped virtual IP: %v", virtualIPsPath(watchingVirtualIP.IP))
				return
			}

			// if the primary virtual IP is removed, all other virtual IPs on that subnet are removed
			// this is in place to restore the virtual IPs that were removed soley by the removal of the primary virtual IP
			if err := addVirtualIP(watchingVirtualIP, virtualInterfaceIndex); err != nil {
				glog.Errorf("Failed to configure virtual IP %v: %v", watchingVirtualIP.IP, err)
				break
			}
		}
	}
}

func determineVirtualIPAddressesOnMe(conn coordclient.Connection, currentVirtualIPNodeIDs []string, hostID string) []string {
	virtualIPsToRefresh := []string{}

	for _, virtualIPAddress := range currentVirtualIPNodeIDs {
		myVirtualIP := pool.VirtualIP{PoolID: "", IP: virtualIPAddress, Netmask: "", BindInterface: ""}
		vipNode := virtualIPNode{HostID: "", VirtualIP: myVirtualIP}
		if err := conn.Get(virtualIPsPath(virtualIPAddress), &vipNode); err != nil {
			glog.Warningf("Unable to retrieve node: %v", virtualIPsPath(virtualIPAddress))
		}

		if vipNode.HostID == hostID {
			// this virtual IP is on ME!
			virtualIPsToRefresh = append(virtualIPsToRefresh, virtualIPAddress)
		}
	}

	return virtualIPsToRefresh
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

func createNode(conn coordclient.Connection, path string) error {
	// Make the path if it doesn't exist
	if exists, err := zkutils.PathExists(conn, path); err != nil && err != coordclient.ErrNoNode {
		return fmt.Errorf("Error checking path %s: %s", path, err)
	} else if !exists {
		if err := conn.CreateDir(path); err != nil {
			return fmt.Errorf("Could not create path %s: %s", path, err)
		}
	}
	return nil
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
func createVirtualInterfaceMap() (map[string]pool.VirtualIP, error) {
	interfaceMap := make(map[string]pool.VirtualIP)

	//ip addr show | awk '/zvip/{print $NF}'
	virtualInterfaceNames, err := exec.Command("bash", "-c", "ip addr show | awk '/"+virtualInterfacePrefix+"/{print $NF}'").CombinedOutput()
	if err != nil {
		glog.Warningf("Determining virtual interfaces failed: %v", err)
		return interfaceMap, err
	}
	glog.V(2).Infof("Control plane virtual interfaces: %v", string(virtualInterfaceNames))

	for _, virtualInterfaceName := range strings.Fields(string(virtualInterfaceNames)) {
		bindInterfaceAndIndex := strings.Split(virtualInterfaceName, virtualInterfacePrefix)
		if len(bindInterfaceAndIndex) != 2 {
			err := fmt.Errorf("Unexpected interface format: %v", bindInterfaceAndIndex)
			return interfaceMap, err
		}
		bindInterface := strings.TrimSpace(string(bindInterfaceAndIndex[0]))

		//ip addr show | awk '/virtualInterfaceName/ {print $2}'
		virtualIPAddressAndCIDR, err := exec.Command("bash", "-c", "ip addr show | awk '/"+virtualInterfaceName+"/ {print $2}'").CombinedOutput()
		if err != nil {
			glog.Warningf("Determining IP address of interface %v failed: %v", virtualInterfaceName, err)
			return interfaceMap, err
		}

		virtualIPAddress, network, err := net.ParseCIDR(strings.TrimSpace(string(virtualIPAddressAndCIDR)))
		if err != nil {
			return interfaceMap, err
		}
		netmask := net.IP(network.Mask)

		interfaceMap[virtualInterfaceName] = pool.VirtualIP{PoolID: "", IP: virtualIPAddress.String(), Netmask: netmask.String(), BindInterface: bindInterface}
	}

	return interfaceMap, nil
}

// add (bind) a virtual IP on the agent
func addVirtualIP(virtualIPToAdd pool.VirtualIP, virtualInterfaceIndex int) error {
	// confirm the virtual IP is not already on this host
	virtualIPAlreadyHere := false
	interfaceMap, err := createVirtualInterfaceMap()
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
		glog.Infof("Requested virtual IP: %v is already on this host.", virtualIPToAdd.IP)
		return nil
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
	interfaceMap, err := createVirtualInterfaceMap()
	if err != nil {
		glog.Warningf("Creating virtual interface map failed")
		return err
	}
	for _, virtualIP := range interfaceMap {
		if virtualIPAddress == virtualIP.IP {
			if err := unbindVirtualIP(virtualIP); err != nil {
				return err
			}
			return nil
		}
	}

	glog.Infof("Requested virtual IP address: %v was not on this host.", virtualIPAddress)
	return nil
}

// bind the virtual IP to the agent
func bindVirtualIP(virtualIP pool.VirtualIP, virtualInterfaceName string) error {
	glog.Infof("Adding: %v", virtualIP)
	// ensure that the Bind Address is reported by ifconfig ... ?
	if err := exec.Command("ifconfig", virtualIP.BindInterface).Run(); err != nil {
		return fmt.Errorf("Problem with BindInterface %s", virtualIP.BindInterface)
	}

	binaryNetmask := net.IPMask(net.ParseIP(virtualIP.Netmask).To4())
	cidr, _ := binaryNetmask.Size()

	// ADD THE VIRTUAL INTERFACE
	// sudo ifconfig eth0:1 inet 192.168.1.136 netmask 255.255.255.0
	// ip addr add IPADDRESS/CIDR dev eth1 label BINDINTERFACE:zvip#
	if err := exec.Command("ip", "addr", "add", virtualIP.IP+"/"+strconv.Itoa(cidr), "dev", virtualIP.BindInterface, "label", virtualInterfaceName).Run(); err != nil {
		return fmt.Errorf("Problem with creating virtual interface %s", virtualInterfaceName)
	}

	glog.Infof("Added virtual interface/IP: %v (%+v)", virtualInterfaceName, virtualIP)
	return nil
}

// unbind the virtual IP from the agent
func unbindVirtualIP(virtualIP pool.VirtualIP) error {
	glog.Infof("Removing: %v", virtualIP.IP)

	binaryNetmask := net.IPMask(net.ParseIP(virtualIP.Netmask).To4())
	cidr, _ := binaryNetmask.Size()

	//sudo ip addr del 192.168.0.10/24 dev eth0
	if err := exec.Command("ip", "addr", "del", virtualIP.IP+"/"+strconv.Itoa(cidr), "dev", virtualIP.BindInterface).Run(); err != nil {
		return fmt.Errorf("Problem with removing virtual interface %+v: %v", virtualIP, err)
	}

	glog.Infof("Removed virtual interface: %+v", virtualIP)
	return nil
}
