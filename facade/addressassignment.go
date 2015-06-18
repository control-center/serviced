// Copyright 2015 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package facade

import (
	"errors"
	"math/rand"
	"time"

	"github.com/control-center/serviced/commons"
	"github.com/control-center/serviced/datastore"
	aa "github.com/control-center/serviced/domain/addressassignment"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/utils"
	"github.com/zenoss/glog"
)

var (
	ErrAddrAssignExists  = errors.New("facade: address assignment exists")
	ErrMultiPorts        = errors.New("facade: multiple endpoints share the same port for service")
	ErrNoIPs             = errors.New("facade: no IPs available for assignment")
	ErrMissingAddrAssign = errors.New("facade: service endpoint is missing an address assignment")
)

type IPInfo struct {
	IP     string
	Type   string
	HostID string
}

type Ports map[uint16]struct{}

func getPorts(endpoints []service.ServiceEndpoint) (Ports, error) {
	ports := make(map[uint16]struct{})
	for _, ep := range endpoints {
		if ep.IsConfigurable() {
			port := ep.AddressConfig.Port
			if _, ok := ports[port]; ok {
				return nil, ErrMultiPorts
			}
			ports[port] = struct{}{}
		}
	}
	return Ports(ports), nil
}

func (p Ports) List() []uint16 {
	ports := make([]uint16, 0)
	for port := range p {
		ports = append(ports, port)
	}
	return ports
}

func (p Ports) GetIP(assignments []aa.AddressAssignment) (string, []uint16) {
	ipaddr := ""
	allports := p.List()
	for _, a := range assignments {
		if ipaddr == "" {
			ipaddr = a.IPAddr
		} else if ipaddr != a.IPAddr {
			return "", allports
		}
		delete(p, a.Port)
	}
	return ipaddr, p.List()
}

func (p Ports) SetIP(ipaddr string, assignments []aa.AddressAssignment) []uint16 {
	for _, a := range assignments {
		if a.IPAddr == ipaddr {
			delete(p, a.Port)
		}
	}
	return p.List()
}

// addAddrAssignment creates a single address assignment.
func (f *Facade) addAddrAssignment(ctx datastore.Context, assign aa.AddressAssignment) error {
	store := aa.NewStore()

	if a, err := f.GetAddrAssignmentByServiceEndpoint(ctx, assign.ServiceID, assign.EndpointName); err != nil {
		glog.Errorf("Could not look up address assignment by service %s and endpoint %s: %s", assign.ServiceID, assign.EndpointName, err)
		return err
	} else if a != nil {
		glog.Errorf("Found address assignment for service %s and endpoint %s", assign.ServiceID, assign.EndpointName)
		return ErrAddrAssignExists
	}
	if a, err := f.GetAddrAssignmentByIPPort(ctx, assign.IPAddr, assign.Port); err != nil {
		glog.Errorf("Could not look up address assignment at %s:%s: %s", assign.IPAddr, assign.Port, err)
		return err
	} else if a != nil {
		glog.Errorf("Found address assignment at %s:%s", assign.IPAddr, assign.Port)
		return ErrAddrAssignExists
	}

	var err error
	if assign.ID, err = utils.NewUUID36(); err != nil {
		glog.Errorf("Could not create address assignment: %s", err)
		return err
	} else if err = store.Put(ctx, aa.Key(assign.ID), &assign); err != nil {
		glog.Errorf("Could not create address assignment: %s", err)
		return err
	}
	return nil
}

// removeAddrAssignment deletes a single address assignment.
func (f *Facade) removeAddrAssignment(ctx datastore.Context, assignID string) error {
	store := aa.NewStore()

	if err := store.Delete(ctx, aa.Key(assignID)); err != nil {
		glog.Errorf("Could not delete assignment %s: %s", assignID, err)
		return err
	}
	return nil
}

// RemoveAddrAssignmentsByService deletes all the address assignments for a service.
func (f *Facade) RemoveAddrAssignmentsByService(ctx datastore.Context, serviceID string) error {
	glog.V(2).Infof("Facade.RemoveAddrAssignmentsByService: %s", serviceID)
	assignments, err := f.GetAddrAssignmentsByService(ctx, serviceID)
	if err != nil {
		glog.Errorf("Could not look up address assignment for service %s: %s", serviceID, err)
		return err
	}
	// TODO: this should be transactional
	for _, assign := range assignments {
		if err := f.removeAddrAssignment(ctx, assign.ID); err != nil {
			glog.Errorf("Could not remove address assignments for service %s: %s", serviceID, err)
			return err
		}
	}

	// stop the service if it is running
	f.StopService(ctx, serviceID, false)
	return nil
}

// RemoveAddrAssignmentsByIP deletes all the address assignments per ip address
func (f *Facade) RemoveAddrAssignmentsByIP(ctx datastore.Context, ipAddr string) error {
	glog.V(2).Infof("Facade.RemoveAddrAssignmentsByIP: %s", ipAddr)
	assignments, err := f.GetAddrAssignmentsByIP(ctx, ipAddr)
	if err != nil {
		glog.Errorf("Could not look up address assignment for ip %s: %s", ipAddr, err)
		return err
	}
	// TODO: this should be transactional
	serviceIDs := make(map[string]struct{})
	for _, assign := range assignments {
		if err := f.removeAddrAssignment(ctx, assign.ID); err != nil {
			glog.Errorf("Could not remove address assignments for ip %s: %s", ipAddr, err)
			return err
		}
		serviceIDs[assign.ServiceID] = struct{}{}
	}

	// stop all affected services
	for serviceID := range serviceIDs {
		f.StopService(ctx, serviceID, false)
	}
	return nil
}

// RemoveAddrAssignmentsByHost deletes all the address assignments per host id.
func (f *Facade) RemoveAddrAssignmentsByHost(ctx datastore.Context, hostID string) error {
	glog.V(2).Infof("Facade.RemoveAddrAssignmentsByHost: %s", hostID)
	assignments, err := f.GetAddrAssignmentsByHost(ctx, hostID)
	if err != nil {
		glog.Errorf("Could not look up address assignment for host %s: %s", hostID, err)
		return err
	}
	// TODO: this should be transactional
	serviceIDs := make(map[string]struct{})
	for _, assign := range assignments {
		if err := f.removeAddrAssignment(ctx, assign.ID); err != nil {
			glog.Errorf("Could not remove address assignments for host id %s: %s", hostID, err)
			return err
		}
		serviceIDs[assign.ServiceID] = struct{}{}
	}

	// stop all affected services
	for serviceID := range serviceIDs {
		f.StopService(ctx, serviceID, false)
	}
	return nil
}

// AssignIPs assigns ips to a service and its children.
func (f *Facade) AssignIPs(ctx datastore.Context, serviceID, ipAddr string) error {
	glog.V(2).Infof("Facade.AssignIPs: serviceID=%s, ipAddr=%s", serviceID, ipAddr)
	assignIP := func(svc *service.Service) error {
		// get all the ports for the service
		portmap, err := getPorts(svc.Endpoints)
		if err != nil {
			glog.Errorf("Found duplicate ports for service %s (%s): %s", svc.Name, svc.ID, err)
			return err
		} else if len(portmap) == 0 {
			return nil
		}

		glog.V(1).Infof("Found ports at %+v for service %s (%s)", portmap.List(), svc.Name, svc.ID)
		// get the address assignments for the service
		assignments, err := f.GetAddrAssignmentsByService(ctx, svc.ID)
		if err != nil {
			glog.Errorf("Could not look up address assignments for service %s (%s): %s", svc.Name, svc.ID, err)
			return err
		}
		var ipinfo *IPInfo
		if ipAddr == "" {
			allports := portmap.List()

			// this is an auto assignment
			if ip, ports := portmap.GetIP(assignments); ip != "" {
				ipinfo, _ = f.getManualAssignment(ctx, svc.PoolID, ip, ports...)
			}
			if ipinfo == nil {
				var err error
				if ipinfo, err = f.getAutoAssignment(ctx, svc.PoolID, allports...); err != nil {
					glog.Errorf("Could not automatically assign an ip to service %s (%s): %s", svc.Name, svc.ID, err)
					return err
				}
			}
		} else {
			// this is a manual assignment
			ports := portmap.SetIP(ipAddr, assignments)
			var err error
			if ipinfo, err = f.getManualAssignment(ctx, svc.PoolID, ipAddr, ports...); err != nil {
				glog.Errorf("Could not assign %s to service %s (%s): %s", svc.Name, svc.ID, ipAddr, err)
				return err
			}
		}
		// clean up address assignments for non-matching ips
		exclude := make(map[string]struct{})
		for _, a := range assignments {
			if a.IPAddr == ipinfo.IP {
				exclude[a.EndpointName] = struct{}{}
			} else if err := f.removeAddrAssignment(ctx, a.ID); err != nil {
				glog.Errorf("Error removing address assignment %s for %s (%s): %s", a.EndpointName, svc.Name, svc.ID, err)
				return err
			}
		}
		// create the new address assignments for the remaining endpoints
		restart := false
		for _, ep := range svc.Endpoints {
			if _, ok := exclude[ep.Name]; !ok && ep.IsConfigurable() {
				assign := aa.AddressAssignment{
					AssignmentType: ipinfo.Type,
					HostID:         ipinfo.HostID,
					PoolID:         svc.PoolID,
					IPAddr:         ipinfo.IP,
					Port:           ep.AddressConfig.Port,
					ServiceID:      svc.ID,
					EndpointName:   ep.Name,
				}
				if err := f.addAddrAssignment(ctx, assign); err != nil {
					glog.Errorf("Error creating address assignment %+v: %s", assign, err)
					return err
				}
				glog.Infof("Created address assignment for endpoint %s of service %s (%s) at %s:%d", ep.Name, svc.Name, svc.ID, ipinfo.IP, assign.Port)
				restart = true
			}
		}
		// restart the service if it is running and endpoints were added
		if restart && svc.DesiredState == int(service.SVCRun) {
			f.RestartService(ctx, svc.ID, false)
		}
		return nil
	}

	// traverse all child services
	return f.walkServices(ctx, serviceID, true, assignIP)
}

// RestoreIPs restores ips for a service
func (f *Facade) RestoreIPs(ctx datastore.Context, svc *service.Service) error {
	glog.V(2).Infof("Facade.AssignIPs: %+v", svc)
	for _, ep := range svc.Endpoints {
		if ep.AddressAssignment.IPAddr != "" {
			if assign, err := f.GetAddrAssignmentByServiceEndpoint(ctx, svc.ID, ep.Name); err != nil {
				glog.Errorf("Could not look up address assignment %s for service %s (%s): %s", ep.Name, svc.Name, svc.ID, err)
				return err
			} else if assign == nil || !assign.EqualIP(ep.AddressAssignment) {
				ipinfo, err := f.getManualAssignment(ctx, svc.PoolID, ep.AddressAssignment.IPAddr, ep.AddressConfig.Port)
				if err != nil {
					glog.Warningf("Could not assign ip %s to endpoint %s for service %s (%s): %s", ep.AddressAssignment.IPAddr, ep.Name, svc.Name, svc.ID, err)
					continue
				}
				*assign = aa.AddressAssignment{
					AssignmentType: ipinfo.Type,
					HostID:         ipinfo.HostID,
					PoolID:         svc.PoolID,
					IPAddr:         ipinfo.IP,
					Port:           ep.AddressConfig.Port,
					ServiceID:      svc.ID,
					EndpointName:   ep.Name,
				}
				if err := f.addAddrAssignment(ctx, *assign); err != nil {
					glog.Errorf("Error creating address assignment %+v: %s", assign, err)
					return err
				}
				glog.Infof("Restored address assignment for endpoint %s of service %s (%s) at %s:%d", assign.EndpointName, svc.Name, svc.ID, assign.IPAddr, assign.Port)
			} else {
				glog.Infof("Endpoint %s for service %s (%s) already assigned; skipping", assign.EndpointName, svc.Name, svc.ID)
			}
		}
	}
	return nil
}

// GetAddrAssignmentByServiceEndpoint gets an address assignment by service ID and endpoint
func (f *Facade) GetAddrAssignmentByServiceEndpoint(ctx datastore.Context, serviceID, endpoint string) (*aa.AddressAssignment, error) {
	glog.V(2).Infof("Facade.GetAddrAssignmentByServiceEndpoint: serviceID=%s endpoint=%s", serviceID, endpoint)
	store := aa.NewStore()
	assign, err := store.FindAssignmentByServiceEndpoint(ctx, serviceID, endpoint)
	if err != nil {
		glog.Errorf("Could not look up address assignment with service %s and endpoint %s: %s", serviceID, endpoint)
		return nil, err
	}
	return assign, nil
}

// GetAddrAssignmentByIPPort gets an address assignment by ip address and port
func (f *Facade) GetAddrAssignmentByIPPort(ctx datastore.Context, ipAddr string, port uint16) (*aa.AddressAssignment, error) {
	glog.V(2).Infof("Facade.GetAddrAssignmentByIPPort: %s:%d", ipAddr, port)
	store := aa.NewStore()
	assign, err := store.FindAssignmentByHostPort(ctx, ipAddr, port)
	if err != nil {
		glog.Errorf("Could not look up address assignment at %s:%d", ipAddr, port)
		return nil, err
	}
	return assign, nil
}

// GetAddrAssignmentByService gets address assignments for a service id.
func (f *Facade) GetAddrAssignmentsByService(ctx datastore.Context, serviceID string) ([]aa.AddressAssignment, error) {
	glog.V(2).Infof("Facade.GetAddrAssignmentByService: %s", serviceID)
	store := aa.NewStore()
	assigns, err := store.GetServiceAddressAssignments(ctx, serviceID)
	if err != nil {
		glog.Errorf("Could not look up address assignments for service %s: %s", serviceID, err)
		return nil, err
	}
	return assigns, nil
}

// GetAddrAssignmentsByIP gets address assignments for an IP
func (f *Facade) GetAddrAssignmentsByIP(ctx datastore.Context, ipAddr string) ([]aa.AddressAssignment, error) {
	glog.V(2).Infof("Facade.GetAddrAssignmentsByIP: %s", ipAddr)
	store := aa.NewStore()
	assigns, err := store.GetServiceAddressAssignmentsByIP(ctx, ipAddr)
	if err != nil {
		glog.Errorf("Could not look up address assignments for ip %s: %s", ipAddr, err)
		return nil, err
	}
	return assigns, nil
}

// GetAddrAssignmentsByHost gets address assignments for a host
func (f *Facade) GetAddrAssignmentsByHost(ctx datastore.Context, hostID string) ([]aa.AddressAssignment, error) {
	glog.V(2).Infof("Facade.GetAddrAssignmentsByHost: %s", hostID)
	store := aa.NewStore()
	assigns, err := store.GetServiceAddressAssignmentsByHost(ctx, hostID)
	if err != nil {
		glog.Errorf("Could not look up address assignments for host id %s: %s", hostID, err)
		return nil, err
	}
	return assigns, nil
}

// GetAddrAssignmentsByPort gets address assignments for a port
func (f *Facade) GetAddrAssignmentsByPort(ctx datastore.Context, port uint16) ([]aa.AddressAssignment, error) {
	glog.V(2).Infof("Facade.GetAddrAssignmentsByPort: %d", port)
	store := aa.NewStore()
	assigns, err := store.GetServiceAddressAssignmentsByPort(ctx, port)
	if err != nil {
		glog.Errorf("Could not look up address assignments for port %d: %s", port, err)
		return nil, err
	}
	return assigns, nil
}

// getAutoAssignment provides an ip assignment given a list of ports.
func (f *Facade) getAutoAssignment(ctx datastore.Context, poolID string, ports ...uint16) (*IPInfo, error) {
	pool, err := f.GetResourcePool(ctx, poolID)
	if err != nil {
		glog.Errorf("Could not look up resource pool %s: %s", poolID, err)
		return nil, err
	}

	ignoreips := make(map[string]struct{})
	for _, port := range ports {
		// get all the address assignments for port
		assignments, err := f.GetAddrAssignmentsByPort(ctx, port)
		if err != nil {
			glog.Errorf("Error while looking up address assignments for port %d: %s", port, err)
			return nil, err
		}

		// filter all of the ips that cannot be used
		for _, assign := range assignments {
			ignoreips[assign.IPAddr] = struct{}{}
		}
	}

	var ipinfos []IPInfo

	// filter the virtual ips
	for _, vip := range pool.VirtualIPs {
		if _, ok := ignoreips[vip.IP]; !ok {
			ipinfos = append(ipinfos, IPInfo{vip.IP, commons.VIRTUAL, ""})
		}
	}

	// get all the static ips
	hosts, err := f.FindHostsInPool(ctx, poolID)
	if err != nil {
		glog.Errorf("Error while looking up hosts in pool %s: %s", poolID, err)
		return nil, err
	}

	// filter the static ips
	for _, host := range hosts {
		for _, hostIP := range host.IPs {
			if _, ok := ignoreips[hostIP.IPAddress]; !ok {
				ipinfos = append(ipinfos, IPInfo{hostIP.IPAddress, commons.STATIC, hostIP.HostID})
			}
		}
	}

	// pick an ip at random
	if total := len(ipinfos); total > 0 {
		rand.Seed(time.Now().UnixNano())
		return &ipinfos[rand.Intn(total)], nil
	}

	glog.Errorf("Could not find an ip assignment for pool %s with ports %+v", poolID, ports)
	return nil, ErrNoIPs
}

// getManualAssignment verifies that an ip assignment is available for a list of ports.
func (f *Facade) getManualAssignment(ctx datastore.Context, poolID string, ipAddress string, ports ...uint16) (*IPInfo, error) {
	// is there already an ip assignment at that address and port?
	for _, port := range ports {
		if assign, err := f.GetAddrAssignmentByIPPort(ctx, ipAddress, port); err != nil {
			glog.Errorf("Could not look up address assignment for %s:%d: %s", ipAddress, port, err)
			return nil, err
		} else if assign != nil {
			glog.Errorf("Assignment found for endpoint %s on service %s: %s", assign.EndpointName, assign.ServiceID, ErrAddrAssignExists)
			return nil, ErrAddrAssignExists
		}
	}

	// check the static ips
	if host, err := f.GetHostByIP(ctx, ipAddress); err != nil {
		glog.Errorf("Error looking up host with IP %s: %s", ipAddress, err)
		return nil, err
	} else if host == nil {
		glog.Errorf("Host not found with ip %s", ipAddress)
		return nil, ErrHostNotExists
	} else if host.PoolID != poolID {
		glog.Errorf("Host %s with ip %s is in pool %s and not pool %s", host.ID, ipAddress, host.PoolID, poolID)
		return nil, ErrHostNotInPool
	} else {
		for _, hostIP := range host.IPs {
			if hostIP.IPAddress == ipAddress {
				return &IPInfo{hostIP.IPAddress, commons.STATIC, hostIP.HostID}, nil
			}
		}
	}

	// check the virtual ips
	if pool, err := f.GetResourcePool(ctx, poolID); err != nil {
		glog.Errorf("Error looking up pool %s: %s", poolID, err)
		return nil, err
	} else {
		for _, vip := range pool.VirtualIPs {
			if vip.IP == ipAddress {
				return &IPInfo{vip.IP, commons.VIRTUAL, ""}, nil
			}
		}
	}

	glog.Errorf("UNEXPECTED RESULT CHECKING IP ASSIGNMENT FOR %s and %+v", ipAddress, ports)
	return nil, ErrNoIPs
}

// setAddrAssignment sets the ip assignments for the service.  This will also clean up
// orphaned ip addresses or addresses that are mismatched on a service.
func (f *Facade) setAddrAssignment(ctx datastore.Context, svc *service.Service) error {
	// get all the address assignments for the service
	assignments, err := f.GetAddrAssignmentsByService(ctx, svc.ID)
	if err != nil {
		glog.Errorf("Could not look up assignments for service %s (%s): %s", svc.Name, svc.ID, err)
		return err
	}

	ips := make(map[string]struct{})
	assignmap := make(map[string]aa.AddressAssignment)

	for _, assign := range assignments {
		ips[assign.IPAddr] = struct{}{}
		assignmap[assign.EndpointName] = assign
	}

	if ipcount := len(ips); ipcount == 1 {
		for i := range svc.Endpoints {
			ep := &svc.Endpoints[i]
			if ep.IsConfigurable() {
				if assign, ok := assignmap[ep.Name]; ok {
					if assign.Port == ep.AddressConfig.Port {
						ep.AddressAssignment = assign
						delete(assignmap, ep.Name)
					}
				}
			}
		}
	} else if ipcount > 1 {
		glog.Warningf("Found multiple ips for service %s (%s); removing all address assignments")
	}

	// delete orphaned assignments
	for _, assign := range assignmap {
		glog.V(1).Infof("Removing assignment %+v: %s", assign, err)
		if err := f.removeAddrAssignment(ctx, assign.ID); err != nil {
			glog.Warningf("Could not remove assignment %+v: %s", assign, err)
		}
	}

	return nil
}
