package container

import (
	"bytes"

	"github.com/zenoss/glog"
	coordclient "github.com/zenoss/serviced/coordinator/client"
	"github.com/zenoss/serviced/dao"
	"github.com/zenoss/serviced/domain/service"
	"github.com/zenoss/serviced/domain/servicestate"
	"github.com/zenoss/serviced/node"
	"github.com/zenoss/serviced/zzk"
	"github.com/zenoss/serviced/zzk/registry"

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
)

var (
	// ErrInvalidZkDSN is returned if the zkDSN is empty or malformed
	ErrInvalidZkDSN = errors.New("container: invalid zookeeper dsn")
	// ErrInvalidExportedEndpoints is returned if the ExportedEndpoints is empty or malformed
	ErrInvalidExportedEndpoints = errors.New("container: invalid exported endpoints")
	// ErrInvalidImportedEndpoints is returned if the ImportedEndpoints is empty or malformed
	ErrInvalidImportedEndpoints = errors.New("container: invalid imported endpoints")
)

type export struct {
	endpoint     *dao.ApplicationEndpoint
	vhosts       []string
	endpointName string
}

type importedEndpoint struct {
	endpointID     string
	instanceID     string
	virtualAddress string
	purpose        string
	port           uint16
}

// getAgentZkDSN retrieves the agent's zookeeper dsn
func getAgentZkDSN(lbClientPort string) (string, error) {
	client, err := node.NewLBClient(lbClientPort)
	if err != nil {
		glog.Errorf("Could not create a client to endpoint: %s, %s", lbClientPort, err)
		return "", err
	}
	defer client.Close()

	var dsn string
	err = client.GetZkDSN(&dsn)
	if err != nil {
		glog.Errorf("Error getting zookeeper dsn, error: %s", err)
		return "", err
	}

	glog.V(1).Infof("getAgentZkDSN: %s", dsn)
	return dsn, nil
}

// getServiceState gets the service states for a serviceID
func getServiceStates(conn coordclient.Connection, serviceID string) ([]*servicestate.ServiceState, error) {
	var serviceStates []*servicestate.ServiceState
	err := zzk.GetServiceStates(conn, &serviceStates, serviceID)
	if err != nil {
		return nil, err
	}
	return serviceStates, nil
}

// getServiceState gets the service state for a serviceID matching the instance ID specified
func getServiceState(conn coordclient.Connection, serviceID, instanceIDStr string) (*servicestate.ServiceState, error) {

	tmpID, err := strconv.Atoi(instanceIDStr)
	if err != nil {
		glog.Errorf("Unable to interpret InstanceID: %s", instanceIDStr)
		return nil, err
	}
	instanceID := int(tmpID)

	for {
		serviceStates, err := getServiceStates(conn, serviceID)
		if err != nil {
			glog.Errorf("Unable to retrieve running service (%s) states: %v", serviceID, err)
			return nil, err
		}

		for ii, ss := range serviceStates {
			if ss.InstanceID == instanceID && ss.PrivateIP != "" {
				return serviceStates[ii], nil
			}
		}

		glog.V(2).Infof("Polling to retrieve service state instanceID:%d with valid PrivateIP", instanceID)
		time.Sleep(1 * time.Second)
	}

	return nil, fmt.Errorf("unable to retrieve service state")
}

// getZkConnection returns the zookeeper connection
func (c *Controller) getZkConnection() (coordclient.Connection, error) {
	if c.cclient == nil {
		var err error
		c.zkDSN, err = getAgentZkDSN(c.options.ServicedEndpoint)
		if err != nil {
			glog.Errorf("Invalid zk dsn")
			return nil, ErrInvalidZkDSN
		}

		c.cclient, err = coordclient.New("zookeeper", c.zkDSN, "", nil)
		if err != nil {
			glog.Errorf("could not connect to zookeeper: %s", c.zkDSN)
			return nil, err
		}

		c.zkConn, err = c.cclient.GetConnection()
		if err != nil {
			return nil, err
		}
	}

	return c.zkConn, nil
}

// getEndpoints builds exportedEndpoints and importedEndpoints
func (c *Controller) getEndpoints(service *service.Service) error {
	// get zookeeper connection
	conn, err := c.getZkConnection()
	if err != nil {
		return err
	}

	if os.Getenv("SERVICED_IS_SERVICE_SHELL") == "true" {
		// this is not a running service, i.e. serviced shell/run
		if hostname, err := os.Hostname(); err != nil {
			glog.Errorf("could not get hostname : %s", err)
			return err
		} else {
			c.dockerID = hostname
		}

		// TODO: deal with exports in the future when there is a use case for it

		sstate, err := servicestate.BuildFromService(service, c.hostID)
		if err != nil {
			return fmt.Errorf("Unable to create temporary service state")
		}
		// initialize importedEndpoints
		c.importedEndpoints, err = buildImportedEndpoints(conn, c.tenantID, sstate)
		if err != nil {
			glog.Errorf("Invalid ImportedEndpoints")
			return ErrInvalidImportedEndpoints
		}
	} else {
		// get service state
		sstate, err := getServiceState(conn, c.options.Service.ID, c.options.Service.InstanceID)
		if err != nil {
			return err
		}
		c.dockerID = sstate.DockerID

		// keep a copy of the service EndPoint exports
		c.exportedEndpoints, err = buildExportedEndpoints(conn, c.tenantID, sstate)
		if err != nil {
			glog.Errorf("Invalid ExportedEndpoints")
			return ErrInvalidExportedEndpoints
		}

		// initialize importedEndpoints
		c.importedEndpoints, err = buildImportedEndpoints(conn, c.tenantID, sstate)
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
			exp.vhosts = defep.VHosts
			exp.endpointName = defep.Name

			var err error
			ep, err := buildApplicationEndpoint(state, &defep)
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
func buildImportedEndpoints(conn coordclient.Connection, tenantID string, state *servicestate.ServiceState) (map[string]importedEndpoint, error) {
	glog.V(2).Infof("buildImportedEndpoints state: %+v", state)
	result := make(map[string]importedEndpoint)

	for _, defep := range state.Endpoints {
		if defep.Purpose == "import" || defep.Purpose == "import_all" {
			endpoint, err := buildApplicationEndpoint(state, &defep)
			if err != nil {
				return result, err
			}
			instanceIDStr := fmt.Sprintf("%d", endpoint.InstanceID)
			setImportedEndpoint(&result, tenantID, endpoint.Application,
				instanceIDStr, endpoint.VirtualAddress, defep.Purpose, endpoint.ContainerPort)
		}
	}

	return result, nil
}

func plus(a, b int) int {
	return a + b
}

// buildApplicationEndpoint converts a ServiceEndpoint to an ApplicationEndpoint
func buildApplicationEndpoint(state *servicestate.ServiceState, endpoint *service.ServiceEndpoint) (*dao.ApplicationEndpoint, error) {
	var ae dao.ApplicationEndpoint

	ae.ServiceID = state.ServiceID
	ae.Application = endpoint.Application
	ae.Protocol = endpoint.Protocol
	ae.ContainerIP = state.PrivateIP
	if endpoint.PortTemplate != "" {
		funcmap := template.FuncMap{
			"plus": plus,
		}
		t := template.Must(template.New("PortTemplate").Funcs(funcmap).Parse(endpoint.PortTemplate))
		b := bytes.Buffer{}
		err := t.Execute(&b, state)
		if err == nil {
			i, err := strconv.Atoi(b.String())
			if err != nil {
				glog.Errorf("%+v", err)
			} else {
				ae.ContainerPort = uint16(i)
			}
		}
	} else {
		ae.ContainerPort = endpoint.PortNumber
	}
	ae.HostIP = state.HostIP
	if len(state.PortMapping) > 0 {
		pmKey := fmt.Sprintf("%d/%s", ae.ContainerPort, ae.Protocol)
		pm := state.PortMapping[pmKey]
		if len(pm) > 0 {
			port, err := strconv.Atoi(pm[0].HostPort)
			if err != nil {
				glog.Errorf("Unable to interpret HostPort: %s", pm[0].HostPort)
				return nil, err
			}
			ae.HostPort = uint16(port)
		}
	}
	ae.VirtualAddress = endpoint.VirtualAddress
	ae.InstanceID = state.InstanceID

	glog.V(2).Infof("  built ApplicationEndpoint: %+v", ae)

	return &ae, nil
}

// setImportedEndpoint sets an imported endpoint
func setImportedEndpoint(importedEndpoints *map[string]importedEndpoint, tenantID, endpointID, instanceID, virtualAddress, purpose string, port uint16) {
	ie := importedEndpoint{}
	ie.endpointID = endpointID
	ie.virtualAddress = virtualAddress
	ie.purpose = purpose
	ie.instanceID = instanceID
	ie.port = port
	key := registry.TenantEndpointKey(tenantID, endpointID)
	(*importedEndpoints)[key] = ie
	glog.V(2).Infof("  cached imported endpoint[%s]: %+v", key, ie)
}

func (c *Controller) getMatchingEndpoint(id string) *importedEndpoint {
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

	zkConn, err := c.cclient.GetConnection()
	if err != nil {
		glog.Errorf("watchRemotePorts - error getting zk connection: %v", err)
		return
	}

	endpointRegistry, err := registry.CreateEndpointRegistry(zkConn)
	if err != nil {
		glog.Errorf("watchRemotePorts - error getting vhost registry: %v", err)
		return
	}

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
				endpointsWatchCanceller <- true
				return
			}
		}

		// setup watchers for each imported tenant endpoint
		watchTenantEndpoints := func(tenantEndpointKey string) {
			glog.V(2).Infof("  watching tenantEndpointKey: %s", tenantEndpointKey)
			if err := endpointRegistry.WatchTenantEndpoint(zkConn, tenantEndpointKey,
				c.processTenantEndpoint, endpointWatchError); err != nil {
				glog.Errorf("error watching tenantEndpointKey %s: %v", tenantEndpointKey, err)
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
		endpoints := make([]*dao.ApplicationEndpoint, len(hostContainerIDs))
		for ii, hostContainerID := range hostContainerIDs {
			path := fmt.Sprintf("%s/%s", parentPath, hostContainerID)
			endpointNode, err := endpointRegistry.GetItem(conn, path)
			if err != nil {
				glog.Errorf("error getting endpoint node at %s: %v", path, err)
			}
			endpoints[ii] = &endpointNode.ApplicationEndpoint
			endpoints[ii].ContainerPort = ep.port
		}

		c.setProxyAddresses(tenantEndpointID, endpoints, ep.virtualAddress, ep.purpose)
	}
}

// setProxyAddresses tells the proxies to update with addresses
func (c *Controller) setProxyAddresses(tenantEndpointID string, endpoints []*dao.ApplicationEndpoint, importVirtualAddress, purpose string) {
	glog.Infof("starting setProxyAddresses(tenantEndpointID: %s, purpose: %s)", tenantEndpointID, purpose)
	proxiesLock.Lock()
	defer proxiesLock.Unlock()
	glog.Infof("starting setProxyAddresses(tenantEndpointID: %s) locked", tenantEndpointID)

	if len(endpoints) <= 0 {
		if prxy, ok := proxies[tenantEndpointID]; ok {
			glog.Errorf("Setting proxy %s to empty address list", tenantEndpointID)
			emptyAddressList := []string{}
			prxy.SetNewAddresses(emptyAddressList)
		} else {
			glog.Errorf("No proxy for %s - no need to set empty address list", tenantEndpointID)
		}
		return
	}

	// First pass of endpoints creates a map of instanceID to array of addresses
	addressMap := make(map[int][]string, len(endpoints))
	for _, endpoint := range endpoints {
		address := fmt.Sprintf("%s:%d", endpoint.HostIP, endpoint.HostPort)
		addressMap[endpoint.InstanceID] = append(addressMap[endpoint.InstanceID], address)
		glog.V(2).Infof("  addresses[%d]: %s  endpoint: %+v", endpoint.InstanceID, addressMap[endpoint.InstanceID], endpoint)
	}

	// Populate a map represnting the exports in this container, so we don't conflict
	exported := map[uint16]struct{}{}
	for _, explist := range c.exportedEndpoints {
		for _, exp := range explist {
			exported[exp.endpoint.ContainerPort] = struct{}{}
		}
	}

	// Populate a map representing the proxies that are to be created, again with instanceID as the key
	proxyKeys := map[int]string{}
	if purpose == "import" {
		// We're doing a normal, load-balanced endpoint import
		proxyKeys[0] = tenantEndpointID
	} else if purpose == "import_all" {
		// Need to create a proxy per instance of the service whose endpoint is
		// being imported
		for _, instance := range endpoints {
			// Port for this instance is base port + instanceID
			containerPort := instance.ContainerPort + uint16(instance.InstanceID)
			if _, conflict := exported[containerPort]; conflict {
				glog.Infof("Skipping import at port %d because it conflicts with a port exported by this container", containerPort)
				continue
			}
			proxyKeys[instance.InstanceID] = fmt.Sprintf("%s_%d", tenantEndpointID, instance.InstanceID)
			instance.ContainerPort = containerPort
		}
	}

	// Now iterate over all the keys, create the proxies, and feed the the addresses for each instance
	for instanceID, proxyKey := range proxyKeys {
		prxy, ok := proxies[proxyKey]
		if !ok {
			var endpoint *dao.ApplicationEndpoint
			for _, ep := range endpoints {
				if ep.InstanceID == instanceID {
					endpoint = ep
					break
				}
			}
			var err error
			prxy, err = createNewProxy(proxyKey, endpoint)
			if err != nil {
				glog.Errorf("error with createNewProxy(%s, %+v) %v", proxyKey, endpoint, err)
				return
			}
			proxies[proxyKey] = prxy

			funcmap := template.FuncMap{
				"plus": plus,
			}
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
					p := strconv.FormatUint(uint64(endpoint.ContainerPort), 10)
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
func createNewProxy(tenantEndpointID string, endpoint *dao.ApplicationEndpoint) (*proxy, error) {
	glog.Infof("Attempting port map for: %s -> %+v", tenantEndpointID, endpoint)

	// setup a new proxy
	listener, err := net.Listen("tcp4", fmt.Sprintf(":%d", endpoint.ContainerPort))
	if err != nil {
		glog.Errorf("Could not bind to port %d: %s", endpoint.ContainerPort, err)
		return nil, err
	}
	prxy, err := newProxy(
		fmt.Sprintf("%v", endpoint),
		cMuxPort,
		cMuxTLS,
		listener)
	if err != nil {
		glog.Errorf("Could not build proxy: %s", err)
		return nil, err
	}

	glog.Infof("Success binding port: %s -> %+v", tenantEndpointID, prxy)
	return prxy, nil
}

// registerExportedEndpoints registers exported ApplicationEndpoints with zookeeper
func (c *Controller) registerExportedEndpoints() {
	// get zookeeper connection
	conn, err := c.getZkConnection()
	if err != nil {
		return
	}

	endpointRegistry, err := registry.CreateEndpointRegistry(conn)
	if err != nil {
		glog.Errorf("Could not get EndpointRegistry. Endpoints not registered: %v", err)
		return
	}

	var vhostRegistry *registry.VhostRegistry
	vhostRegistry, err = registry.VHostRegistry(conn)
	if err != nil {
		glog.Errorf("Could not get vhost registy. Endpoints not registered: %v", err)
		return
	}

	// register exported endpoints
	for key, exportList := range c.exportedEndpoints {
		for _, export := range exportList {
			endpoint := export.endpoint
			for _, vhost := range export.vhosts {
				epName := fmt.Sprintf("%s_%v", export.endpointName, export.endpoint.InstanceID)
				var path string
				if path, err = vhostRegistry.SetItem(conn, vhost, registry.NewVhostEndpoint(epName, *endpoint)); err != nil {
					glog.Errorf("could not register vhost %s for %s: %v", vhost, epName, err)
				}
				glog.Infof("Registered vhost %s for %s at %s", vhost, epName, path)
			}

			glog.Infof("Registering exported endpoint[%s]: %+v", key, *endpoint)
			path, err := endpointRegistry.SetItem(conn, registry.NewEndpointNode(c.tenantID, export.endpoint.Application, c.hostID, c.dockerID, *endpoint))
			if err != nil {
				glog.Errorf("  unable to add endpoint: %+v %v", *endpoint, err)
				continue
			}

			glog.V(1).Infof("  endpoint successfully added to path: %s", path)
		}
	}
}

var (
	proxiesLock             sync.RWMutex
	proxies                 map[string]*proxy
	vifs                    *VIFRegistry
	nextip                  int
	watchers                map[string]bool
	endpointsWatchCanceller chan bool
	cMuxPort                uint16 // the TCP port to use
	cMuxTLS                 bool
)

func init() {
	proxies = make(map[string]*proxy)
	vifs = NewVIFRegistry()
	nextip = 1
	watchers = make(map[string]bool)
	endpointsWatchCanceller = make(chan bool)
}