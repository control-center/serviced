// Copyright 2014 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package container

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"

	coordclient "github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/domain/applicationendpoint"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/servicestate"
	"github.com/control-center/serviced/node"
	"github.com/control-center/serviced/zzk"
	"github.com/control-center/serviced/zzk/registry"
	zkservice "github.com/control-center/serviced/zzk/service"
	"github.com/zenoss/glog"
)

var (
	// ErrInvalidZkInfo is returned if the zkDSN is empty or malformed or poolID was not obtained
	ErrInvalidZkInfo = errors.New("container: invalid zookeeper info (dsn/poolID)")
	// ErrInvalidExportedEndpoints is returned if the ExportedEndpoints is empty or malformed
	ErrInvalidExportedEndpoints = errors.New("container: invalid exported endpoints")
	// ErrInvalidImportedEndpoints is returned if the ImportedEndpoints is empty or malformed
	ErrInvalidImportedEndpoints = errors.New("container: invalid imported endpoints")
)

// Functions for evaluating port/virtualAddress templates
var funcmap = template.FuncMap{
	"plus": func(a, b int) int {
		return a + b
	},
}

type export struct {
	endpoint      applicationendpoint.ApplicationEndpoint
	vhosts        []string
	portAddresses []string
	endpointName  string
}

type importedEndpoint struct {
	endpointID     string
	instanceID     string
	virtualAddress string
	purpose        string
	port           uint16
}

// getAgentZkInfo retrieves the agent's zookeeper dsn
func getAgentZkInfo(lbClientPort string) (node.ZkInfo, error) {
	var zkInfo node.ZkInfo
	client, err := node.NewLBClient(lbClientPort)
	if err != nil {
		glog.Errorf("Could not create a client to endpoint: %s, %s", lbClientPort, err)
		return zkInfo, err
	}
	defer client.Close()

	err = client.GetZkInfo(&zkInfo)
	if err != nil {
		glog.Errorf("Error getting zookeeper dsn/poolID, error: %s", err)
		return zkInfo, err
	}

	glog.V(1).Infof("GetZkInfo: %+v", zkInfo)
	return zkInfo, nil
}

// getServiceState gets the service states for a serviceID
func getServiceStates(conn coordclient.Connection, serviceID string) ([]servicestate.ServiceState, error) {
	return zkservice.GetServiceStates(conn, serviceID)
}

// getServiceState gets the service state for a serviceID matching the instance ID specified
func getServiceState(conn coordclient.Connection, serviceID, instanceIDStr string) (*servicestate.ServiceState, error) {
	tmpID, err := strconv.Atoi(instanceIDStr)
	if err != nil {
		glog.Errorf("Unable to interpret InstanceID: %s", instanceIDStr)
		return nil, fmt.Errorf("endpoint.go getServiceState failed: %v", err)
	}
	instanceID := int(tmpID)

	for {
		serviceStates, err := getServiceStates(conn, serviceID)
		if err != nil {
			glog.Errorf("Unable to retrieve running service (%s) states: %v", serviceID, err)
			return nil, fmt.Errorf("endpoint.go getServiceState zzk.GetServiceStates failed: %v", err)
		}

		for ii, ss := range serviceStates {
			if ss.InstanceID == instanceID && ss.PrivateIP != "" {
				return &serviceStates[ii], nil
			}
		}

		glog.V(2).Infof("Polling to retrieve service state instanceID:%d with valid PrivateIP", instanceID)
		time.Sleep(1 * time.Second)
	}
}

// getEndpoints builds exportedEndpoints and importedEndpoints
func (c *Controller) getEndpoints(service *service.Service) error {
	var err error
	c.zkInfo, err = getAgentZkInfo(c.options.ServicedEndpoint)
	if err != nil {
		glog.Errorf("Invalid zk info: %v", err)
		return err //ErrInvalidZkInfo
	}
	glog.Infof(" c.zkInfo: %+v", c.zkInfo)

	// endpoints are created at the root level (not pool aware)
	rootBasePath := ""
	zClient, err := coordclient.New("zookeeper", c.zkInfo.ZkDSN, rootBasePath, nil)
	if err != nil {
		glog.Errorf("failed create a new coordclient: %v", err)
		return err
	}

	zzk.InitializeLocalClient(zClient)

	// get zookeeper connection
	conn, err := zzk.GetLocalConnection(zzk.GeneratePoolPath(service.PoolID))
	if err != nil {
		return fmt.Errorf("getEndpoints zzk.GetLocalConnection failed: %v", err)
	}

	if os.Getenv("SERVICED_IS_SERVICE_SHELL") == "true" {
		// this is not a running service, i.e. serviced shell/run
		if hostname, err := os.Hostname(); err != nil {
			glog.Errorf("could not get hostname: %s", err)
			return fmt.Errorf("getEndpoints failed could not get hostname: %v", err)
		} else {
			c.dockerID = hostname
		}

		// TODO: deal with exports in the future when there is a use case for it

		sstate, err := servicestate.BuildFromService(service, c.hostID)
		if err != nil {
			return fmt.Errorf("Unable to create temporary service state")
		}
		// initialize importedEndpoints
		err = buildImportedEndpoints(c, conn, sstate)
		if err != nil {
			glog.Errorf("Invalid ImportedEndpoints")
			return ErrInvalidImportedEndpoints
		}
	} else {
		// get service state
		glog.Infof("getting service state: %s %v", c.options.Service.ID, c.options.Service.InstanceID)
		sstate, err := getServiceState(conn, c.options.Service.ID, c.options.Service.InstanceID)
		if err != nil {
			return fmt.Errorf("getEndpoints getServiceState failed: %v", err)
		}
		c.dockerID = sstate.DockerID

		// keep a copy of the service EndPoint exports
		c.exportedEndpoints, err = buildExportedEndpoints(conn, c.tenantID, sstate)
		if err != nil {
			glog.Errorf("Invalid ExportedEndpoints")
			return ErrInvalidExportedEndpoints
		}

		// initialize importedEndpoints
		err = buildImportedEndpoints(c, conn, sstate)
		if err != nil {
			glog.Errorf("Invalid ImportedEndpoints")
			return ErrInvalidImportedEndpoints
		}
	}

	return nil
}

// buildExportedEndpoints builds the map to exported endpoints
func buildExportedEndpoints(conn coordclient.Connection, tenantID string, state *servicestate.ServiceState) (map[string][]export, error) {
	glog.V(2).Infof("buildExportedEndpoints state: %+v", state)
	result := make(map[string][]export)

	for _, defep := range state.Endpoints {
		if defep.Purpose == "export" {

			exp := export{}
			if len(defep.VHostList) > 0 {
				exp.vhosts = []string{}
				for _, vhost := range defep.VHostList {
					exp.vhosts = append(exp.vhosts, vhost.Name)
				}
			}
			if len(defep.PortList) > 0 {
				exp.portAddresses = []string{}
				for _, port := range defep.PortList {
					exp.portAddresses = append(exp.portAddresses, port.PortAddr)
				}
			}
			exp.endpointName = defep.Name

			var err error
			ep, err := applicationendpoint.BuildApplicationEndpoint(state, &defep)
			if err != nil {
				return result, err
			}
			exp.endpoint = ep

			key := registry.TenantEndpointKey(tenantID, exp.endpoint.Application)
			if _, exists := result[key]; !exists {
				result[key] = make([]export, 0)
			}
			result[key] = append(result[key], exp)

			glog.V(2).Infof("  cached exported endpoint[%s]: %+v", key, exp)
		}
	}

	return result, nil
}

// buildImportedEndpoints builds the map to imported endpoints
func buildImportedEndpoints(c *Controller, conn coordclient.Connection, state *servicestate.ServiceState) error {
	glog.V(2).Infof("buildImportedEndpoints state: %+v", state)

	for _, defep := range state.Endpoints {
		if defep.Purpose == "import" || defep.Purpose == "import_all" {
			endpoint, err := applicationendpoint.BuildApplicationEndpoint(state, &defep)
			if err != nil {
				return err
			}
			instanceIDStr := fmt.Sprintf("%d", endpoint.InstanceID)
			setImportedEndpoint(c, endpoint.Application,
				instanceIDStr, endpoint.VirtualAddress, defep.Purpose, endpoint.ContainerPort)
		}
	}
	return nil
}

// setImportedEndpoint sets an imported endpoint
func setImportedEndpoint(c *Controller, endpointID, instanceID, virtualAddress, purpose string, port uint16) {
	ie := importedEndpoint{}
	ie.endpointID = endpointID
	ie.virtualAddress = virtualAddress
	ie.purpose = purpose
	ie.instanceID = instanceID
	ie.port = port
	key := registry.TenantEndpointKey(c.tenantID, endpointID)
	c.importedEndpointsLock.Lock()
	c.importedEndpoints[key] = ie
	c.importedEndpointsLock.Unlock()
	glog.Infof("  cached imported endpoint[%s]: %+v", key, ie)
}

func (c *Controller) getMatchingEndpoint(id string) *importedEndpoint {
	c.importedEndpointsLock.RLock()
	defer c.importedEndpointsLock.RUnlock()
	for _, ie := range c.importedEndpoints {
		endpointPattern := fmt.Sprintf("^%s$", registry.TenantEndpointKey(c.tenantID, ie.endpointID))
		glog.V(2).Infof("  checking tenantEndpointID %s against pattern %s", id, endpointPattern)
		endpointRegex, err := regexp.Compile(endpointPattern)
		if err != nil {
			glog.Warningf("  unable to check tenantEndpointID %s against imported endpoint %s", id, ie.endpointID)
			continue //Don't spam error message; it was reported at validation time
		}

		if endpointRegex.MatchString(id) {
			glog.V(2).Infof("  tenantEndpointID:%s matched imported endpoint pattern:%s for %+v", id, endpointPattern, ie)
			return &ie
		}
	}
	return nil
}

// watchRemotePorts watches imported endpoints and updates proxies
func (c *Controller) watchRemotePorts() {
	/*
		watch each tenant endpoint
			- when endpoints are added, add the endpoint proxy if not already added
			- when endpoints are added, add watch on that endpoint for updates
			- when endpoints are deleted, tell that endpoint proxy to stop proxying - done with ephemeral znodes
			- when endpoints are deleted, may not need to deal with removing watch on that endpoint since that watch will block forever
			- deal with import regexes, i.e mysql_.*
		- may not need to initially deal with removal of tenant endpoint
	*/
	cMuxPort = uint16(c.options.Mux.Port)
	cMuxTLS = c.options.Mux.TLS

	for key, endpoint := range c.importedEndpoints {
		glog.V(2).Infof("importedEndpoints[%s]: %+v", key, endpoint)
	}

	var err error
	c.zkInfo, err = getAgentZkInfo(c.options.ServicedEndpoint)
	if err != nil {
		glog.Errorf("Invalid zk info: %v", err)
		return
	}

	zkConn, err := zzk.GetLocalConnection("/")
	if err != nil {
		glog.Errorf("watchRemotePorts - error getting zk connection: %v", err)
		return
	}
	endpointRegistry, err := registry.CreateEndpointRegistry(zkConn)
	if err != nil {
		glog.Errorf("watchRemotePorts - error getting endpoint registry: %v", err)
		return
	}
	//translate closing call to endpoint cancel
	cancelEndpointWatch := make(chan interface{})
	go func() {
		select {
		case errc := <-c.closing:
			glog.Infof("Closing endpoint watchers")
			select {
			case endpointsWatchCanceller <- true:
			default:
			}
			close(cancelEndpointWatch)
			errc <- nil
		}
	}()

	processTenantEndpoints := func(conn coordclient.Connection, parentPath string, tenantEndpointIDs ...string) {
		glog.V(2).Infof("processTenantEndpoints for path: %s tenantEndpointIDs: %s", parentPath, tenantEndpointIDs)

		// cancel watcher on top level /endpoints if all watchers on imported endpoints have been set up
		{
			ignorePrefix := fmt.Sprintf("%s_controlplane", c.tenantID)
			missingWatchers := false
			for id := range c.importedEndpoints {
				if strings.HasPrefix(id, ignorePrefix) {
					// ignore controlplane special imports for now - handleRemotePorts starts proxies for those right now
					// TODO: register controlplane special imports in isvcs and watch for them
					continue
				}
				if _, ok := watchers[id]; !ok {
					missingWatchers = true
				}
			}
			if !missingWatchers {
				glog.V(2).Infof("all imports are being watched - cancelling watcher on /endpoints")
				select {
				case endpointsWatchCanceller <- true:
					return
				default:
					return
				}
			}
		}

		// setup watchers for each imported tenant endpoint
		watchTenantEndpoints := func(tenantEndpointKey string) {
			glog.V(2).Infof("  watching tenantEndpointKey: %s", tenantEndpointKey)
			for {

				glog.Infof("Starting watch for tenantEndpointKey %s: %v", tenantEndpointKey, err)
				if err := endpointRegistry.WatchTenantEndpoint(zkConn, tenantEndpointKey,
					c.processTenantEndpoint, endpointWatchError, cancelEndpointWatch); err != nil {
					glog.Errorf("error watching tenantEndpointKey %s: %v", tenantEndpointKey, err)
				}
				select {
				case <-cancelEndpointWatch:
					glog.Infof("Closing watch for tenantEndpointKey %s", tenantEndpointKey)
					return
				case <-time.After(500 * time.Millisecond): //prevent tight loop
				}
			}
		}

		for _, id := range tenantEndpointIDs {
			glog.V(2).Infof("checking need to watch tenantEndpoint: %s %s", parentPath, id)

			// add watchers if they don't exist for a tenantid_application
			// and if tenant-endpoint is an import
			if _, ok := watchers[id]; !ok {
				if _, ok := c.importedEndpoints[id]; ok {
					watchers[id] = true
					go watchTenantEndpoints(id)
				} else {
					// look for imports with regexes that match each tenantEndpointID
					ep := c.getMatchingEndpoint(id)
					if ep != nil {
						watchers[id] = true
						go watchTenantEndpoints(id)
					} else {
						glog.V(2).Infof("  no need to add - not imported: %s %s for importedEndpoints: %+v", parentPath, id, c.importedEndpoints)
					}
				}
			} else {
				glog.V(2).Infof("  no need to add - existing watch tenantEndpoint: %s %s", parentPath, id)
			}

			// BEWARE: only need to deal with add, currently no need to deal with deletes
			// since tenant endpoints are currently not deleted.  only the hostid_containerid
			// entries within tenantid_application are added/deleted
		}

	}
	glog.V(2).Infof("watching endpointRegistry")
	go endpointRegistry.WatchRegistry(zkConn, endpointsWatchCanceller, processTenantEndpoints, endpointWatchError)
}

// endpointWatchError shows errors with watches
func endpointWatchError(path string, err error) {
	glog.Warningf("processing endpointWatchError on %s: %v", path, err)
}

// processTenantEndpoint updates the addresses for an imported endpoint
func (c *Controller) processTenantEndpoint(conn coordclient.Connection, parentPath string, hostContainerIDs ...string) {
	glog.V(2).Infof("processTenantEndpoint: parentPath:%s hostContainerIDs: %v", parentPath, hostContainerIDs)

	// update the proxy for this tenant endpoint
	endpointRegistry, err := registry.CreateEndpointRegistry(conn)
	if err != nil {
		glog.Errorf("Could not get EndpointRegistry. Endpoints not registered: %v", err)
		return
	}

	parts := strings.Split(parentPath, "/")
	tenantEndpointID := parts[len(parts)-1]

	if ep := c.getMatchingEndpoint(tenantEndpointID); ep != nil {
		endpoints := make(map[int]applicationendpoint.ApplicationEndpoint, len(hostContainerIDs))
		for ii, hostContainerID := range hostContainerIDs {
			path := fmt.Sprintf("%s/%s", parentPath, hostContainerID)
			endpointNode, err := endpointRegistry.GetItem(conn, path)
			if err != nil {
				glog.Errorf("error getting endpoint node at %s: %v", path, err)
				continue
			}
			endpoint := endpointNode.ApplicationEndpoint
			if ep.port != 0 {
				glog.V(2).Infof("overriding ProxyPort with imported port:%v for endpoint: %+v", ep.port, endpointNode)
				endpoint.ProxyPort = ep.port
			} else {
				glog.V(2).Infof("not overriding ProxyPort with imported port:%v for endpoint: %+v", ep.port, endpointNode)
				endpoint.ProxyPort = endpoint.ContainerPort
			}
			endpoints[ii] = endpoint
		}
		c.setProxyAddresses(tenantEndpointID, endpoints, ep.virtualAddress, ep.purpose)
	}
}

// setProxyAddresses tells the proxies to update with addresses
func (c *Controller) setProxyAddresses(tenantEndpointID string, endpoints map[int]applicationendpoint.ApplicationEndpoint, importVirtualAddress, purpose string) {
	glog.V(1).Info("starting setProxyAddresses(tenantEndpointID: %s, purpose: %s)", tenantEndpointID, purpose)
	proxiesLock.Lock()
	defer proxiesLock.Unlock()
	glog.V(1).Infof("starting setProxyAddresses(tenantEndpointID: %s) locked", tenantEndpointID)

	if len(endpoints) <= 0 {
		if prxy, ok := proxies[tenantEndpointID]; ok {
			glog.Errorf("Setting proxy %s to empty address list", tenantEndpointID)
			emptyAddressList := []addressTuple{}
			prxy.SetNewAddresses(emptyAddressList)
		} else {
			glog.Errorf("No proxy for %s - no need to set empty address list", tenantEndpointID)
		}
		return
	}

	// First pass of endpoints creates a map of proxy index (which is
	// instanceID in an import_all scenario) to array of addresses
	addressMap := make(map[int][]addressTuple, len(endpoints))
	for _, endpoint := range endpoints {
		address := addressTuple{
			host:          endpoint.HostIP,
			containerAddr: fmt.Sprintf("%s:%d", endpoint.ContainerIP, endpoint.ContainerPort),
		}
		if purpose == "import" {
			// If we're a load-balanced endpoint, we don't care about instance
			// ID; just put everything on 0, since we will have 1 proxy
			addressMap[0] = append(addressMap[0], address)
			glog.V(2).Infof("  addresses[%d]: %s  endpoint: %+v", 0, addressMap[0], endpoint)
		} else if purpose == "import_all" {
			// In this case, we care about instance ID -> address, because we
			// will have a proxy per instance
			addressMap[endpoint.InstanceID] = append(addressMap[endpoint.InstanceID], address)
			glog.V(2).Infof("  addresses[%d]: %s  endpoint: %+v", endpoint.InstanceID, addressMap[endpoint.InstanceID], endpoint)
		}
	}

	// Build a list of ports exported by this container, so we can check for
	// conflicts when we do imports later.
	exported := map[uint16]struct{}{}
	for _, explist := range c.exportedEndpoints {
		for _, exp := range explist {
			exported[exp.endpoint.ContainerPort] = struct{}{}
		}
	}

	// Populate a map representing the proxies that are to be created, again
	// with proxy index as the key
	proxyKeys := map[int]string{}
	if purpose == "import" {
		// We're doing a normal, load-balanced endpoint import
		proxyKeys[0] = tenantEndpointID
	} else if purpose == "import_all" {
		// Need to create a proxy per instance of the service whose endpoint is
		// being imported
		for key, instance := range endpoints {
			// Port for this instance is base port + instanceID
			proxyPort := instance.ProxyPort + uint16(instance.InstanceID)
			if _, conflict := exported[proxyPort]; conflict {
				glog.Warningf("Skipping import at port %d because it conflicts with a port exported by this container", proxyPort)
				continue
			}
			proxyKeys[instance.InstanceID] = fmt.Sprintf("%s_%d", tenantEndpointID, instance.InstanceID)
			instance.ProxyPort = proxyPort
			endpoints[key] = instance
		}
	}

	// Now iterate over all the keys, create the proxies, and feed in the
	// addresses for each instance
	for instanceID, proxyKey := range proxyKeys {
		prxy, ok := proxies[proxyKey]
		if !ok {
			var endpoint applicationendpoint.ApplicationEndpoint
			if purpose == "import" {
				endpoint = endpoints[0]
			} else {
				for _, ep := range endpoints {
					if ep.InstanceID == instanceID {
						endpoint = ep
						break
					}
				}
			}

			var err error
			prxy, err = createNewProxy(proxyKey, endpoint, c.allowDirectConn)
			if err != nil {
				glog.Errorf("error with createNewProxy(%s, %+v) %v", proxyKey, endpoint, err)
				return
			}
			proxies[proxyKey] = prxy

			for _, vaddr := range []string{importVirtualAddress, endpoint.VirtualAddress} {
				// Evaluate virtual address template
				t := template.Must(template.New(endpoint.Application).Funcs(funcmap).Parse(vaddr))
				var buffer bytes.Buffer
				if err := t.Execute(&buffer, endpoint); err != nil {
					glog.Errorf("Failed to evaluate VirtualAddress template")
					return
				}
				virtualAddress := buffer.String()
				// Now actually make the thing
				if virtualAddress != "" {
					p := strconv.FormatUint(uint64(endpoint.ProxyPort), 10)
					err := vifs.RegisterVirtualAddress(virtualAddress, p, endpoint.Protocol)
					if err != nil {
						glog.Errorf("Error creating virtual address %s: %+v", virtualAddress, err)
					}
				}
			}
		}
		prxy.SetNewAddresses(addressMap[instanceID])
	}
}

// createNewProxy creates a new proxy
func createNewProxy(tenantEndpointID string, endpoint applicationendpoint.ApplicationEndpoint, allowDirect bool) (*proxy, error) {
	glog.Infof("Attempting port map for: %s -> %+v", tenantEndpointID, endpoint)

	// setup a new proxy
	listener, err := net.Listen("tcp4", fmt.Sprintf(":%d", endpoint.ProxyPort))
	if err != nil {
		glog.Errorf("Could not bind to port %d: %s", endpoint.ProxyPort, err)
		return nil, err
	}
	prxy, err := newProxy(
		fmt.Sprintf("%v", endpoint),
		tenantEndpointID,
		cMuxPort,
		cMuxTLS,
		listener,
		allowDirect)
	if err != nil {
		glog.Errorf("Could not build proxy: %s", err)
		return nil, err
	}

	glog.Infof("Success binding port: %s -> %+v", tenantEndpointID, prxy)
	return prxy, nil
}

func (c *Controller) watchregistry() <-chan struct{} {
	alert := make(chan struct{}, 1)

	go func() {

		paths := append(c.publicEndpointZKPaths, c.exportedEndpointZKPaths...)
		if len(paths) == 0 {
			return
		}

		conn, err := zzk.GetLocalConnection("/")
		if err != nil {
			return
		}

		endpointRegistry, err := registry.CreateEndpointRegistry(conn)
		if err != nil {
			glog.Errorf("Could not get EndpointRegistry. Endpoints not checked: %v", err)
			return
		}

		interval := time.Tick(60 * time.Second)

		defer func() { alert <- struct{}{} }()
		for {
			select {
			case <-interval:
				for _, path := range paths {
					if _, err := endpointRegistry.GetItem(conn, path); err != nil {
						glog.Errorf("Could not get endpoint. %v", err)
						return
					}
				}
			}
		}
	}()

	return alert
}

// registerExportedEndpoints registers exported ApplicationEndpoints with zookeeper
func (c *Controller) registerExportedEndpoints() error {
	// TODO: accumulate the errors so that some endpoints get registered
	conn, err := zzk.GetLocalConnection("/")
	if err != nil {
		return err
	}

	endpointRegistry, err := registry.CreateEndpointRegistry(conn)
	if err != nil {
		glog.Errorf("Could not get EndpointRegistry. Endpoints not registered: %v", err)
		return err
	}

	var publicEndpointRegistry *registry.PublicEndpointRegistryType
	publicEndpointRegistry, err = registry.PublicEndpointRegistry(conn)
	if err != nil {
		glog.Errorf("Could not get public endpoint registy. Endpoints not registered: %v", err)
		return err
	}

	c.publicEndpointZKPaths = []string{}
	c.exportedEndpointZKPaths = []string{}

	// register exported endpoints
	for key, exportList := range c.exportedEndpoints {
		for _, export := range exportList {
			endpoint := export.endpoint

			epName := fmt.Sprintf("%s_%v", export.endpointName, export.endpoint.InstanceID)
			//register vhosts
			for _, vhost := range export.vhosts {
				glog.V(1).Infof("registerExportedEndpoints: vhost epName=%s", epName)
				//delete any existing vhost that hasn't been cleaned up
				vhostEndpoint := registry.NewPublicEndpoint(epName, endpoint)
				pepKey := registry.GetPublicEndpointKey(vhost, registry.EPTypeVHost)

				if paths, err := publicEndpointRegistry.GetChildren(conn, pepKey); err != nil {
					glog.Errorf("error trying to get previous vhosts: %s", err)
				} else {
					glog.V(1).Infof("cleaning vhost paths %v", paths)
					//clean paths
					for _, path := range paths {
						if vep, err := publicEndpointRegistry.GetItem(conn, path); err != nil {
							glog.V(1).Infof("Could not read %s", path)
						} else {
							glog.V(4).Infof("checking instance id of %#v equal %v", vep, c.options.Service.InstanceID)
							if strconv.Itoa(vep.InstanceID) == c.options.Service.InstanceID {
								glog.V(1).Infof("Deleting stale vhost registration for %v at %v ", vhost, path)
								conn.Delete(path)
							}
						}
					}
				}

				// TODO: avoid set if item already exist with data we want
				var path string
				if path, err = publicEndpointRegistry.SetItem(conn, pepKey, vhostEndpoint); err != nil {
					glog.Errorf("could not register vhost %s for %s: %v", vhost, epName, err)
					return err
				} else {
					glog.Infof("Registered vhost %s for %s at %s", vhost, epName, path)
					c.publicEndpointZKPaths = append(c.publicEndpointZKPaths, path)
				}
			}

			//register ports
			for _, port := range export.portAddresses {
				glog.V(1).Infof("registerExportedEndpoints: vhost epName=%s", epName)
				//delete any existing vhost that hasn't been cleaned up
				vhostEndpoint := registry.NewPublicEndpoint(epName, endpoint)
				pepKey := registry.GetPublicEndpointKey(port, registry.EPTypePort)

				if paths, err := publicEndpointRegistry.GetChildren(conn, pepKey); err != nil {
					glog.Errorf("error trying to get previous vhosts: %s", err)
				} else {
					glog.V(1).Infof("cleaning vhost paths %v", paths)
					//clean paths
					for _, path := range paths {
						if vep, err := publicEndpointRegistry.GetItem(conn, path); err != nil {
							glog.V(1).Infof("Could not read %s", path)
						} else {
							glog.V(4).Infof("checking instance id of %#v equal %v", vep, c.options.Service.InstanceID)
							if strconv.Itoa(vep.InstanceID) == c.options.Service.InstanceID {
								glog.V(1).Infof("Deleting stale port registration for %v at %v ", port, path)
								conn.Delete(path)
							}
						}
					}
				}

				// TODO: avoid set if item already exist with data we want
				var path string
				if path, err = publicEndpointRegistry.SetItem(conn, pepKey, vhostEndpoint); err != nil {
					glog.Errorf("could not register port %s for %s: %v", port, epName, err)
					return err
				} else {
					glog.Infof("Registered port %s for %s at %s", port, epName, path)
					c.publicEndpointZKPaths = append(c.publicEndpointZKPaths, path)
				}
			}

			// delete any existing endpoint that hasn't been cleaned up
			if paths, err := endpointRegistry.GetChildren(conn, c.tenantID, export.endpoint.Application); err != nil {
				glog.Errorf("error trying to get endpoints: %s", err)
			} else {
				glog.V(1).Infof("cleaning endpoint paths %v", paths)
				//clean paths
				for _, path := range paths {
					if epn, err := endpointRegistry.GetItem(conn, path); err != nil {
						glog.Errorf("Could not read %s", path)
					} else {
						glog.V(4).Infof("checking instance id of %#v equal %v", epn, c.options.Service.InstanceID)
						if strconv.Itoa(epn.InstanceID) == c.options.Service.InstanceID {
							glog.V(1).Infof("Deleting stale endpoint registration for %v at %v ", export.endpointName, path)
							conn.Delete(path)
						}
					}
				}
			}
			glog.Infof("Registering exported endpoint[%s]: %+v", key, endpoint)
			endpoint.HostID = c.hostID
			endpoint.ContainerID = c.dockerID
			path, err := endpointRegistry.SetItem(conn, registry.NewEndpointNode(c.tenantID, export.endpoint.Application, endpoint))
			if err != nil {
				glog.Errorf("  unable to add endpoint: %+v %v", endpoint, err)
				return err
			}
			c.exportedEndpointZKPaths = append(c.exportedEndpointZKPaths, path)
			glog.V(1).Infof("  endpoint successfully added to path: %s", path)
		}
	}
	return nil
}

func (c *Controller) unregisterPublicEndpoints() {
	conn, err := zzk.GetLocalConnection("/")
	if err != nil {
		return
	}
	for _, path := range c.publicEndpointZKPaths {
		glog.V(1).Infof("controller shutdown deleting public endpoint %v", path)
		conn.Delete(path)
	}
}

func (c *Controller) unregisterEndpoints() {
	conn, err := zzk.GetLocalConnection("/")
	if err != nil {
		return
	}

	for _, path := range c.exportedEndpointZKPaths {
		glog.V(1).Infof("controller shutdown deleting endpoint %v", path)
		conn.Delete(path)
	}
}

var (
	proxiesLock             sync.RWMutex
	proxies                 map[string]*proxy
	vifs                    *VIFRegistry
	nextip                  int
	watchers                map[string]bool
	endpointsWatchCanceller chan interface{}
	cMuxPort                uint16 // the TCP port to use
	cMuxTLS                 bool
)

func init() {
	proxies = make(map[string]*proxy)
	vifs = NewVIFRegistry()
	nextip = 1
	watchers = make(map[string]bool)
	endpointsWatchCanceller = make(chan interface{})
}
