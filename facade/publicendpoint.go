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
	"strings"

	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/servicedefinition"
	"github.com/zenoss/glog"
)

// Adds a port public endpoint to a service
func (f *Facade) AddPublicEndpointPort(ctx datastore.Context, serviceID, endpointName, portAddr string,
	usetls bool, protocol string, isEnabled bool, restart bool) (*servicedefinition.Port, error) {

	// Validate the port number
	scrubbedPort := service.ScrubPortString(portAddr)
	portParts := strings.Split(scrubbedPort, ":")
	if len(portParts) < 2 {
		err := fmt.Errorf("Invalid port address. Port address must be \":[PORT NUMBER]\" or \"[IP ADDRESS]:[PORT NUMBER]\"")
		glog.Error(err)
		return nil, err
	}

	if portAddr == "0" || strings.HasSuffix(portAddr, ":0") {
		err := fmt.Errorf("Invalid port address. Port 0 is invalid.")
		glog.Error(err)
		return nil, err
	}

	// Check to make sure the port is available.  Don't allow adding a port if it's already being used.
	// This has the added benefit of validating the port address before it gets added to the service
	// definition.
	if err := checkPort("tcp", fmt.Sprintf("%s", scrubbedPort)); err != nil {
		glog.Error(err)
		return nil, err
	}

	// Get the service for this service id.
	svc, err := f.GetService(ctx, serviceID)
	if err != nil {
		err = fmt.Errorf("Could not find service %s: %s", serviceID, err)
		glog.Error(err)
		return nil, err
	}

	// check other ports for redundancy
	services, err := f.GetAllServices(ctx)
	if err != nil {
		err = fmt.Errorf("Could not get the list of services: %s", err)
		glog.Error(err)
		return nil, err
	}

	for _, service := range services {
		if service.Endpoints == nil {
			continue
		}

		for _, endpoint := range service.Endpoints {
			for _, epPort := range endpoint.PortList {
				if scrubbedPort == epPort.PortAddr {
					err := fmt.Errorf("Port %s already defined for service: %s", epPort.PortAddr, service.Name)
					glog.Error(err)
					return nil, err
				}
			}
		}
	}

	// Add the port to the service definition.
	port, err := svc.AddPort(endpointName, portAddr, usetls, protocol, isEnabled)
	if err != nil {
		glog.Error(err)
		return nil, err
	}

	// Make sure no other service currently has zzk data for this port -- this would result
	// in the service getting turned off during restart but not being able to start again.
	if err := f.validateServiceStart(ctx, svc); err != nil {
		// We don't call UpdateService() service here; effectively unwinding the svc.AddPort() call above.
		glog.Error(err)
		return nil, err
	}

	glog.V(2).Infof("Added port public endpoint %s to service %s", portAddr, svc.Name)

	if err = f.UpdateService(ctx, *svc); err != nil {
		glog.Error(err)
		return nil, err
	}

	glog.V(2).Infof("Service (%s) updated", svc.Name)

	// Restart the service if it is running
	if restart && (svc.DesiredState == int(service.SVCRun) || svc.DesiredState == int(service.SVCRestart)) {
		if _, err = f.RestartService(ctx, dao.ScheduleServiceRequest{ServiceID: svc.ID}); err != nil {
			err = fmt.Errorf("Error restarting service %s: %s", svc.Name, err)
			glog.Error(err)
			return nil, err
		}
	}

	glog.V(2).Infof("Service %s updated after adding port public endpoint", svc.Name)
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
	// Get the service for this service id.
	svc, err := f.GetService(ctx, serviceid)
	if err != nil {
		err = fmt.Errorf("Could not find service %s: %s", serviceid, err)
		glog.Error(err)
		return err
	}

	err = svc.RemovePort(endpointName, portAddr)
	if err != nil {
		err = fmt.Errorf("Error removing port %s from service (%s): %v", portAddr, svc.Name, err)
		glog.Error(err)
		return err
	}

	glog.V(2).Infof("Removed port public endpoint %s from service %s", portAddr, svc.Name)

	if err = f.UpdateService(ctx, *svc); err != nil {
		glog.Error(err)
		return err
	}

	glog.V(2).Infof("Service (%s) updated", svc.Name)

	// Restart the service if it is running
	if svc.DesiredState == int(service.SVCRun) || svc.DesiredState == int(service.SVCRestart) {
		if _, err = f.RestartService(ctx, dao.ScheduleServiceRequest{ServiceID: svc.ID}); err != nil {
			err = fmt.Errorf("Error restarting service %s: %s", svc.Name, err)
			glog.Error(err)
			return err
		}
	}

	glog.V(2).Infof("Service %s updated after adding removing public endpoint %s", svc.Name, portAddr)
	return nil
}

// Enable/Disable a port public endpoint.
func (f *Facade) EnablePublicEndpointPort(ctx datastore.Context, serviceid, endpointName, portAddr string, isEnabled bool) error {
	// Get the service for this service id.
	svc, err := f.GetService(ctx, serviceid)
	if err != nil {
		err = fmt.Errorf("Could not find service %s: %s", serviceid, err)
		glog.Error(err)
		return err
	}

	var enableString string
	if isEnabled {
		enableString = "enabling"
	} else {
		enableString = "disabling"
	}

	// If they're trying to enable the port, check to make sure the port is valid and available.
	if isEnabled {
		// Validate the port number
		scrubbedPort := service.ScrubPortString(portAddr)
		portParts := strings.Split(scrubbedPort, ":")
		if len(portParts) < 2 {
			err = fmt.Errorf("Invalid port address. Port address must be \":[PORT NUMBER]\" or \"[IP ADDRESS]:[PORT NUMBER]\"")
			glog.Error(err)
			return err
		}

		if portAddr == "0" || strings.HasSuffix(portAddr, ":0") {
			err = fmt.Errorf("Invalid port address. Port 0 is invalid.")
			glog.Error(err)
			return err
		}

		if err = checkPort("tcp", fmt.Sprintf("%s", scrubbedPort)); err != nil {
			glog.Error(err)
			return err
		}
	}

	err = svc.EnablePort(endpointName, portAddr, isEnabled)
	if err != nil {
		err = fmt.Errorf("Error %s port %s for service (%s): %v", enableString, portAddr, svc.Name, err)
		glog.Error(err)
		return err
	}

	glog.V(2).Infof("Port public endpoint %s has been %s for service %s", portAddr, enableString, svc.Name)

	if err = f.UpdateService(ctx, *svc); err != nil {
		glog.Error(err)
		return err
	}

	glog.V(2).Infof("Service (%s) updated", svc.Name)
	return nil
}

// Adds a vhost public endpoint to a service
func (f *Facade) AddPublicEndpointVHost(ctx datastore.Context, serviceid, endpointName, vhostName string, isEnabled, restart bool) (*servicedefinition.VHost, error) {
	// Get the service for this service id.
	svc, err := f.GetService(ctx, serviceid)
	if err != nil {
		err = fmt.Errorf("Could not find service %s: %s", serviceid, err)
		glog.Error(err)
		return nil, err
	}

	// check other virtual hosts for redundancy
	vhostLowerName := strings.ToLower(vhostName)
	services, err := f.GetAllServices(ctx)
	if err != nil {
		err = fmt.Errorf("Could not get the list of services: %s", err)
		glog.Error(err)
		return nil, err
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
					return nil, err
				}
			}
		}
	}

	vhost, err := svc.AddVirtualHost(endpointName, vhostName)
	if err != nil {
		err := fmt.Errorf("Error adding vhost (%s) to service (%s): %v", vhostName, svc.Name, err)
		glog.Error(err)
		return nil, err
	}

	// Make sure no other service currently has zzk data for this vhost -- this would result
	// in the service getting turned off during restart but not being able to start again.
	if err := f.validateServiceStart(ctx, svc); err != nil {
		// We don't call UpdateService() service here; effectively unwinding the svc.AddVirtualHost() call above.
		glog.Error(err)
		return nil, err
	}

	glog.V(2).Infof("Added vhost public endpoint %s to service %s", vhost.Name, svc.Name)

	if err = f.UpdateService(ctx, *svc); err != nil {
		glog.Error(err)
		return nil, err
	}

	glog.V(2).Infof("Service (%s) updated", svc.Name)

	// Restart the service if it is running
	if restart && (svc.DesiredState == int(service.SVCRun) || svc.DesiredState == int(service.SVCRestart)) {
		if _, err = f.RestartService(ctx, dao.ScheduleServiceRequest{ServiceID: svc.ID}); err != nil {
			err = fmt.Errorf("Error restarting service %s: %s", svc.Name, err)
			glog.Error(err)
			return nil, err
		}
	}

	glog.V(2).Infof("Service %s updated after adding vhost public endpoint (%s)", svc.Name, vhost.Name)
	return vhost, nil
}
