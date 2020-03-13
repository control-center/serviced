// Copyright 2016 The Serviced Authors.
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
	"fmt"
	"net"
	"regexp"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/control-center/serviced/audit"
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/servicedefinition"
	"github.com/zenoss/glog"
)

var vhostNameRegex = regexp.MustCompile("^(([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9-]*[a-zA-Z0-9]).)*([A-Za-z0-9]|[A-Za-z0-9][A-Za-z0-9-]*[A-Za-z0-9])$")

// Adds a port public endpoint to a service
func (f *Facade) AddPublicEndpointPort(ctx datastore.Context, serviceID, endpointName, portAddr string,
	usetls bool, protocol string, isEnabled bool, restart bool) (*servicedefinition.Port, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.AddPublicEndpointPort"))
	alog := f.auditLogger.Message(ctx, "Adding Public Endpoint Port").Action(audit.Update).ID(serviceID).
		WithFields(logrus.Fields{
			"endpointname": endpointName,
			"portaddr":     portAddr,
			"usetls":       usetls,
			"protocol":     protocol,
			"isenabled":    isEnabled,
		})
	// Scrub the port for all checks, as this is what gets stored against the service.
	portAddr = service.ScrubPortString(portAddr)

	// Validate the port number
	portParts := strings.Split(portAddr, ":")
	if len(portParts) < 2 {
		err := fmt.Errorf("Invalid port address. Port address must be \":[PORT NUMBER]\" or \"[IP ADDRESS]:[PORT NUMBER]\"")
		glog.Error(err)
		return nil, alog.Error(err)
	}

	if portAddr == "0" || strings.HasSuffix(portAddr, ":0") {
		err := fmt.Errorf("Invalid port address. Port 0 is invalid.")
		glog.Error(err)
		return nil, alog.Error(err)
	}

	// Check to make sure the port is available.  Don't allow adding a port if it's already being used.
	// This has the added benefit of validating the port address before it gets added to the service
	// definition.
	if err := checkPort("tcp", fmt.Sprintf("%s", portAddr)); err != nil {
		glog.Error(err)
		return nil, alog.Error(err)
	}

	// Get the service for this service id.
	svc, err := f.GetService(ctx, serviceID)
	if err != nil {
		err = fmt.Errorf("Could not find service %s: %s", serviceID, err)
		glog.Error(err)
		return nil, alog.Error(err)
	}
	alog = alog.Entity(svc)

	// check other ports for redundancy
	// FIXME: GetAllServices is really expensive. Can this be replaced
	//        with a more targeted method like get-all-services with portAddr "blah"?
	services, err := f.GetAllServices(ctx)
	if err != nil {
		err = fmt.Errorf("Could not get the list of services: %s", err)
		glog.Error(err)
		return nil, alog.Error(err)
	}

	for _, service := range services {
		if service.Endpoints == nil {
			continue
		}

		for _, endpoint := range service.Endpoints {
			for _, epPort := range endpoint.PortList {
				if portAddr == epPort.PortAddr {
					err := fmt.Errorf("Port %s already defined for service: %s", epPort.PortAddr, service.Name)
					glog.Error(err)
					return nil, alog.Error(err)
				}
			}
		}
	}

	// Add the port to the service definition.
	port, err := svc.AddPort(endpointName, portAddr, usetls, protocol, isEnabled)
	if err != nil {
		glog.Error(err)
		return nil, alog.Error(err)
	}

	// Make sure no other service currently has zzk data for this port -- this would result
	// in the service getting turned off during restart but not being able to start again.
	if err := f.validateServiceStart(ctx, svc); err != nil {
		// We don't call UpdateService() service here; effectively unwinding the svc.AddPort() call above.
		glog.Error(err)
		return nil, alog.Error(err)
	}

	glog.V(2).Infof("Added port public endpoint %s to service %s", portAddr, svc.Name)

	if err = f.UpdateService(ctx, *svc); err != nil {
		glog.Error(err)
		return nil, alog.Error(err)
	}

	glog.V(2).Infof("Service (%s) updated", svc.Name)
	alog.Succeeded()
	return port, nil
}

// Try to open the port.  If the port opens, we're good. Otherwise return the error.
func checkPort(network string, laddr string) error {
	glog.V(2).Infof("Checking %s port %s", network, laddr)
	listener, err := net.Listen(network, laddr)
	if err != nil {
		// Port isn't available.
		glog.V(2).Infof("Port Listen failed; something else is using this port.")
		return err
	} else {
		// Port was opened. Make sure we close it.
		glog.V(2).Infof("Port Listen succeeded. Closing the listener.")
		listener.Close()
	}
	return nil
}

// Remove the port public endpoint from a service.
func (f *Facade) RemovePublicEndpointPort(ctx datastore.Context, serviceid, endpointName, portAddr string) error {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.RemovePublicEndpointPort"))
	alog := f.auditLogger.Message(ctx, "Removing Public Endpoint Port").ID(serviceid).Action(audit.Update).
		WithFields(logrus.Fields{
			"endpointname": endpointName,
			"portaddr":     portAddr,
		})
	// Scrub the port for all checks, as this is what gets stored against the service.
	portAddr = service.ScrubPortString(portAddr)

	// Get the service for this service id.
	svc, err := f.GetService(ctx, serviceid)
	if err != nil {
		err = fmt.Errorf("Could not find service %s: %s", serviceid, err)
		glog.Error(err)
		return alog.Error(err)
	}
	alog = alog.Entity(svc)

	err = svc.RemovePort(endpointName, portAddr)
	if err != nil {
		err = fmt.Errorf("Error removing port %s from service (%s): %v", portAddr, svc.Name, err)
		glog.Error(err)
		return alog.Error(err)
	}

	glog.V(2).Infof("Removed port public endpoint %s from service %s", portAddr, svc.Name)

	if err = f.UpdateService(ctx, *svc); err != nil {
		glog.Error(err)
		return alog.Error(err)
	}

	glog.V(2).Infof("Service (%s) updated", svc.Name)
	alog.Succeeded()
	return nil
}

// SetAddressConfig sets the AddressConfig to the endpoint
func (f *Facade) SetAddressConfig(ctx datastore.Context, serviceid, endpointName string, sa servicedefinition.AddressResourceConfig) error {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.SetAddressConfig"))
	svc, err := f.GetService(ctx, serviceid)
	if err != nil {
		return err
	}

	alog := f.auditLogger.Action(audit.Update).Message(ctx, "Remove IP Assignment").
		WithFields(logrus.Fields{
			"servicename": svc.Name,
			"endpoint":    endpointName,
		}).Entity(svc)

	err = svc.SetAddressConfig(endpointName, sa)
	if err != nil {
		err = fmt.Errorf("error setting AddressConfig to the service (%s): %v", svc.Name, err)
		return alog.Error(err)
	}

	if err = f.UpdateService(ctx, *svc); err != nil {
		return alog.Error(err)
	}

	alog.Succeeded()
	return nil
}

// Enable/Disable a port public endpoint.
func (f *Facade) EnablePublicEndpointPort(ctx datastore.Context, serviceid, endpointName, portAddr string, isEnabled bool) error {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.EnablePublicEndpointPort"))
	var auditMsg string
	if isEnabled {
		auditMsg = "Enabling Public Endpoint Port"
	} else {
		auditMsg = "Disabling Public Endpoing Port"
	}
	alog := f.auditLogger.Message(ctx, auditMsg).Action(audit.Update).ID(serviceid).
		WithFields(logrus.Fields{
			"endpointname": endpointName,
			"portaddr":     portAddr,
		})
	// Scrub the port for all checks, as this is what gets stored against the service.
	portAddr = service.ScrubPortString(portAddr)

	// Get the service for this service id.
	svc, err := f.GetService(ctx, serviceid)
	if err != nil {
		err = fmt.Errorf("Could not find service %s: %s", serviceid, err)
		glog.Error(err)
		return alog.Error(err)
	}
	alog = alog.Entity(svc)

	// Find the port so we can check the current enabled state.
	port := svc.GetPort(endpointName, portAddr)
	if port == nil {
		err = fmt.Errorf("Port %s not found in service %s:%s", portAddr, svc.ID, svc.Name)
		glog.Error(err)
		return alog.Error(err)
	}

	// If the port is already in the same state, don't do anything.
	if port.Enabled == isEnabled {
		err = fmt.Errorf("Port %s enabled state is already set to %t for service (%s).", portAddr, isEnabled, svc.Name)
		glog.Warning(err)
		return alog.Error(err)
	}

	glog.V(0).Infof("Setting enabled=%t for service (%s) port %s", isEnabled, svc.Name, portAddr)

	// If they're trying to enable the port, check to make sure the port is valid and available.
	if isEnabled {
		// Validate the port number
		portParts := strings.Split(portAddr, ":")
		if len(portParts) < 2 {
			err = fmt.Errorf("Invalid port address. Port address must be \":[PORT NUMBER]\" or \"[IP ADDRESS]:[PORT NUMBER]\"")
			glog.Error(err)
			return alog.Error(err)
		}

		if portAddr == "0" || strings.HasSuffix(portAddr, ":0") {
			err = fmt.Errorf("Invalid port address. Port 0 is invalid.")
			glog.Error(err)
			return alog.Error(err)
		}

		if err = checkPort("tcp", fmt.Sprintf("%s", portAddr)); err != nil {
			glog.Error(err)
			return alog.Error(err)
		}
	}

	err = svc.EnablePort(endpointName, portAddr, isEnabled)
	if err != nil {
		err = fmt.Errorf("error setting enabled=%t for port %s, service (%s): %v", isEnabled, portAddr, svc.Name, err)
		return alog.Error(err)
	}

	glog.V(2).Infof("Port (%s) enable state set to %t for service (%s)", portAddr, isEnabled, svc.Name)

	if err = f.UpdateService(ctx, *svc); err != nil {
		glog.Error(err)
		return alog.Error(err)
	}

	glog.V(2).Infof("Service (%s) updated", svc.Name)
	alog.Succeeded()
	return nil
}

// Adds a vhost public endpoint to a service
func (f *Facade) AddPublicEndpointVHost(ctx datastore.Context, serviceid, endpointName, vhostName string, isEnabled, restart bool) (*servicedefinition.VHost, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.AddPublicEndpointVHost"))
	alog := f.auditLogger.Message(ctx, "Adding Public Endpoint VHost").Action(audit.Update).ID(serviceid).
		WithFields(logrus.Fields{
			"endpointname": endpointName,
			"vhostname":    vhostName,
			"isenabled":    isEnabled,
		})
	// Get the service for this service id.
	svc, err := f.GetService(ctx, serviceid)
	if err != nil {
		err = fmt.Errorf("Could not find service %s: %s", serviceid, err)
		glog.Error(err)
		return nil, alog.Error(err)
	}
	alog = alog.Entity(svc)

	// Make sure the name doesn't contain invalid characters.
	if !vhostNameRegex.MatchString(vhostName) {
		err = fmt.Errorf("Invalid virtual host name: %s", vhostName)
		glog.Error(err)
		return nil, alog.Error(err)
	}

	// check other virtual hosts for redundancy
	vhostLowerName := strings.ToLower(vhostName)

	// FIXME: GetAllServices is really expensive. Can this be replaced
	//        with a more targeted method like get-all-services with vhost "blah"?
	services, err := f.GetAllServices(ctx)
	if err != nil {
		err = fmt.Errorf("Could not get the list of services: %s", err)
		glog.Error(err)
		return nil, alog.Error(err)
	}

	for _, otherService := range services {
		if otherService.Endpoints == nil {
			continue
		}
		for _, endpoint := range otherService.Endpoints {
			for _, vhost := range endpoint.VHostList {
				if strings.ToLower(vhost.Name) == vhostLowerName {
					err := fmt.Errorf("vhost %s already defined for service: %s", vhostName, otherService.Name)
					glog.Error(err)
					return nil, alog.Error(err)
				}
			}
		}
	}

	vhost, err := svc.AddVirtualHost(endpointName, vhostName, isEnabled)
	if err != nil {
		err := fmt.Errorf("Error adding vhost (%s) to service (%s): %v", vhostName, svc.Name, err)
		glog.Error(err)
		return nil, alog.Error(err)
	}

	// Make sure no other service currently has zzk data for this vhost -- this would result
	// in the service getting turned off during restart but not being able to start again.
	if err := f.validateServiceStart(ctx, svc); err != nil {
		// We don't call UpdateService() service here; effectively unwinding the svc.AddVirtualHost() call above.
		glog.Error(err)
		return nil, alog.Error(err)
	}

	glog.V(2).Infof("Added vhost public endpoint %s to service %s", vhost.Name, svc.Name)

	if err = f.UpdateService(ctx, *svc); err != nil {
		glog.Error(err)
		return nil, alog.Error(err)
	}

	glog.V(2).Infof("Service (%s) updated", svc.Name)
	alog.Succeeded()
	return vhost, nil
}

// Remove the vhost public endpoint from a service.
func (f *Facade) RemovePublicEndpointVHost(ctx datastore.Context, serviceid, endpointName, vhost string) error {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.RemovePublicEndpointVHost"))
	alog := f.auditLogger.Message(ctx, "Removing Public Endpoint VHost").Action(audit.Update).ID(serviceid).
		WithFields(logrus.Fields{
			"endpointname": endpointName,
			"vhost":        vhost,
		})
	// Get the service for this service id.
	svc, err := f.GetService(ctx, serviceid)
	if err != nil {
		err = fmt.Errorf("Could not find service %s: %s", serviceid, err)
		glog.Error(err)
		return alog.Error(err)
	}
	alog = alog.Entity(svc)

	err = svc.RemoveVirtualHost(endpointName, vhost)
	if err != nil {
		err = fmt.Errorf("Error removing vhost %s from service (%s): %v", vhost, svc.Name, err)
		glog.Error(err)
		return alog.Error(err)
	}

	glog.V(2).Infof("Removed vhost public endpoint %s from service %s", vhost, svc.Name)

	if err = f.UpdateService(ctx, *svc); err != nil {
		glog.Error(err)
		return alog.Error(err)
	}

	glog.V(2).Infof("Service (%s) updated", svc.Name)
	alog.Succeeded()
	return nil
}

func (f *Facade) EnablePublicEndpointVHost(ctx datastore.Context, serviceid, endpointName, vhost string, isEnabled bool) error {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.EnablePublicEndpointVHost"))
	var auditMsg string
	if isEnabled {
		auditMsg = "Enabling Public Endpoint VHost"
	} else {
		auditMsg = "Disabling Public Endpoing Vhost"
	}
	alog := f.auditLogger.Message(ctx, auditMsg).Action(audit.Update).ID(serviceid).
		WithFields(logrus.Fields{
			"endpointname": endpointName,
			"vhost":        vhost,
		})
	// Get the service for this service id.
	svc, err := f.GetService(ctx, serviceid)
	if err != nil {
		err = fmt.Errorf("Could not find service %s: %s", serviceid, err)
		glog.Error(err)
		return alog.Error(err)
	}
	alog = alog.Entity(svc)

	// Find the vhost so we can check the current enabled state.
	existingVHost := svc.GetVirtualHost(endpointName, vhost)
	if existingVHost == nil {
		err = fmt.Errorf("VHost %s not found in service %s:%s", vhost, svc.ID, svc.Name)
		glog.Error(err)
		return alog.Error(err)
	}

	// If the vhost is already in the same state, don't do anything.
	if existingVHost.Enabled == isEnabled {
		err = fmt.Errorf("VHost %s enabled state is already set to %t for service (%s).", vhost, isEnabled, svc.Name)
		glog.Warning(err)
		return alog.Error(err)
	}

	glog.V(0).Infof("Setting enabled=%t for service (%s) vhost %s", isEnabled, svc.Name, vhost)

	err = svc.EnableVirtualHost(endpointName, vhost, isEnabled)
	if err != nil {
		err = fmt.Errorf("Error setting vhost (%s) enable state to %t for service (%s): %v", vhost, isEnabled, svc.Name, err)
		glog.Error(err)
		return alog.Error(err)
	}

	glog.V(2).Infof("VHost (%s) enable state set to %t for service (%s)", vhost, isEnabled, svc.Name)

	if err = f.UpdateService(ctx, *svc); err != nil {
		glog.Error(err)
		return alog.Error(err)
	}

	glog.V(2).Infof("Service (%s) updated", svc.Name)
	alog.Succeeded()
	return nil
}

// GetAllPublicEndpoints returns all the public endpoints in the system
func (f *Facade) GetAllPublicEndpoints(ctx datastore.Context) ([]service.PublicEndpoint, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("GetAllPublicEndpoints"))
	return f.serviceStore.GetAllPublicEndpoints(ctx)
}
