package serviced

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/dao"
	"github.com/zenoss/serviced/utils"
)

func populateVirtualInterfaceNames(virtualIPsToAdd []dao.VirtualIP, interfaceMap []dao.VirtualIP) []dao.VirtualIP {
	glog.Info("******************** started populateVirtualInterfaceNames")
	defer glog.Info("******************** finished populateVirtualInterfaceNames")

	proposedInterfaceName := ""
	proposedInterfaceNameIsAcceptable := true
	MAX_INDEX := 100
	interfaceIndex := 0

	virtualIPsReadyToAdd := []dao.VirtualIP{}

	for _, virtualIPToAdd := range virtualIPsToAdd {
		for interfaceIndex = 0; interfaceIndex < MAX_INDEX; interfaceIndex++ {
			virtualIPToAdd.Index = strconv.Itoa(interfaceIndex)
			proposedInterfaceName = generateInterfaceName(virtualIPToAdd)

			proposedInterfaceNameIsAcceptable = true
			for _, currentInterface := range interfaceMap {
				glog.Infof(" ########## checking %v === %v", proposedInterfaceName, generateInterfaceName(currentInterface))
				if proposedInterfaceName == generateInterfaceName(currentInterface) {
					glog.Warningf("Proposed interface name: %v is already taken...", proposedInterfaceName)
					proposedInterfaceNameIsAcceptable = false
					break
				}
			}
			if proposedInterfaceNameIsAcceptable {
				virtualIPToAdd.Index = strconv.Itoa(interfaceIndex)
				virtualIPsReadyToAdd = append(virtualIPsReadyToAdd, virtualIPToAdd)
				break
			}
		}
		if interfaceIndex == MAX_INDEX {
			glog.Warningf("There are over %v virtual IP interfaces", MAX_INDEX)
		}
	}

	return virtualIPsReadyToAdd
}

var VIRTUAL_INTERFACE_PREFIX = ":cpvip"

// create a map of [bindaddress][interface_name] = ip_address
func createVirtualInterfaceMap() (error, []dao.VirtualIP) {
	glog.Info("Creating Virtual Interface Map...")
	interfaceMap := []dao.VirtualIP{}

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
		interfaceMap = append(interfaceMap, dao.VirtualIP{"", "", strings.TrimSpace(string(virtualIP)), "", bindInterface, interfaceIndex})
	}

	glog.Infof(" ********** Virtual Interface Map: %v", interfaceMap)

	return nil, interfaceMap
}

func generateInterfaceName(virtualIP dao.VirtualIP) string {
	if virtualIP.Index == "" {
		glog.Errorf("Virtual IP: %v has no Index... cannot generate its interface name.", virtualIP.IP)
	}
	return virtualIP.BindInterface + VIRTUAL_INTERFACE_PREFIX + virtualIP.Index
}

func addVirtualIPToLeader(virtualIP dao.VirtualIP) {
	glog.Infof("Adding: %v", virtualIP)
	// ensure that the Bind Address is reported by ifconfig ... ?
	if err := exec.Command("ifconfig", virtualIP.BindInterface).Run(); err != nil {
		glog.Warningf("Problem with BindInterface %s", virtualIP.BindInterface)
		return
	}

	virtualInterfaceName := generateInterfaceName(virtualIP)
	// ADD THE VIRTUAL INTERFACE
	// sudo ifconfig eth0:1 inet 192.168.1.136 netmask 255.255.255.0
	if err := exec.Command("ifconfig", virtualInterfaceName, "inet", virtualIP.IP, "netmask", virtualIP.Netmask).Run(); err != nil {
		glog.Warningf("Problem with creating virtual interface %s", virtualInterfaceName)
		return
	}

	glog.Infof("Added: %v", virtualIP)
}

func removeVirtualIPToLeader(virtualIP dao.VirtualIP) {
	glog.Infof("Removing: %v", virtualIP)
	virtualInterfaceName := generateInterfaceName(virtualIP)
	// ifconfig eth0:1 down
	if err := exec.Command("ifconfig", virtualInterfaceName, "down").Run(); err != nil {
		glog.Warningf("Problem with removing virtual interface %v: %v", virtualInterfaceName, err)
		return
	}
	glog.Infof("Removed interface: %s %v", virtualInterfaceName, virtualIP)
}

func virtualIPExists(aVirtualIP dao.VirtualIP, virtualIPs []dao.VirtualIP) bool {
	for _, virtualIP := range virtualIPs {
		//aVirtualIP.PoolId == virtualIP.PoolId &&
		//aVirtualIP.Netmask == virtualIP.Netmask &&
		//aVirtualIP.Index == virtualIP.Index
		if aVirtualIP.IP == virtualIP.IP && aVirtualIP.BindInterface == virtualIP.BindInterface {
			return true
		}
	}
	return false
}

func watchVirtualIPs(cpDao dao.ControlPlane) {
	glog.Info("******************** started watchVirtualIPs")
	defer glog.Info("******************** finished watchVirtualIPs")

	hostId, err := utils.HostID()
	if err != nil {
		glog.Errorf("Could not get host ID: %v", err)
		return
	}

	host := dao.Host{}
	if err := cpDao.GetHost(hostId, &host); err != nil {
		glog.Errorf("Cannot retrieve host information for pool host %s (%v)", hostId, err)
		return
	}

	var pool dao.ResourcePool
	if err := cpDao.GetResourcePool(host.PoolId, &pool); err != nil {
		glog.Errorf("Unable to load resource pool: %v", host.PoolId)
		return
	}

	err, interfaceMap := createVirtualInterfaceMap()
	if err != nil {
		glog.Warningf("Creating virtual interface map failed: %v", err)
		return
	}

	if len(pool.VirtualIPs) == 0 && len(interfaceMap) == 0 {
		glog.Infof("There are 0 virtual IP address in pool: %v (there are also 0 virtual IP addresses on host: %v)", pool.Id, host.Name)
		return
	}

	addVirtualIP := true
	virtualIPsToAdd := []dao.VirtualIP{}
	virtualIPsToKeep := []dao.VirtualIP{}
	for _, virtualIP := range pool.VirtualIPs { // add these if they do not already exist
		addVirtualIP = true
		for _, virtualInterface := range interfaceMap { // these already exist on the leader
			glog.Infof(" ++++++++++++++++ Checking virtualIP: %v against virtualInterface: %v", virtualIP, virtualInterface)
			if virtualIP.IP == virtualInterface.IP {
				glog.Warningf("Virtual interface %v is already set to %v", virtualInterface, virtualIP)
				addVirtualIP = false
				break
			}
		}
		if addVirtualIP {
			glog.Infof(" ^^^^^^^^^^ Need to add: %v", virtualIP)
			virtualIPsToAdd = append(virtualIPsToAdd, virtualIP)
		} else {
			glog.Infof(" ^^^^^^^^^ Need to keep: %v", virtualIP)
			virtualIPsToKeep = append(virtualIPsToKeep, virtualIP)
		}
	}

	virtualIPsReadyToAdd := populateVirtualInterfaceNames(virtualIPsToAdd, interfaceMap)
	glog.Infof("pool.VirtualIPs     : %v", pool.VirtualIPs)
	glog.Infof("virtualIPsToAdd     : %v", virtualIPsToAdd)
	glog.Infof("virtualIPsToKeep    : %v", virtualIPsToKeep)
	glog.Infof("virtualIPsReadyToAdd: %v", virtualIPsReadyToAdd)

	for _, virtualIP := range virtualIPsReadyToAdd {
		addVirtualIPToLeader(virtualIP)
	}

	for _, aVirtualIP := range interfaceMap {
		// check to see if the virtual interface on this host should be kept
		// aVirtualIP was discovered on this host (IP, BindAddress, Index)
		// virtualIPsToKeep was derived from pool.VirtualIPs (IP, Netmask, BindAddress)
		keepVirtualIp := virtualIPExists(aVirtualIP, virtualIPsToKeep)
		if !keepVirtualIp {
			removeVirtualIPToLeader(aVirtualIP)
		}
	}
}
