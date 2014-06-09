package container

import (
	// "github.com/fatih/set"
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced"
	"github.com/zenoss/serviced/commons/subprocess"
	coordclient "github.com/zenoss/serviced/coordinator/client"
	"github.com/zenoss/serviced/dao"
	"github.com/zenoss/serviced/domain"
	"github.com/zenoss/serviced/domain/service"
	"github.com/zenoss/serviced/domain/servicedefinition"
	"github.com/zenoss/serviced/domain/servicestate"
	"github.com/zenoss/serviced/zzk"
	"github.com/zenoss/serviced/zzk/registry"

	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// TODO: remove useImportedEndpointServiceDiscovery or set it to true
const useImportedEndpointServiceDiscovery = false

var (
	// ErrInvalidCommand is returned if a command is empty or malformed
	ErrInvalidCommand = errors.New("container: invalid command")
	// ErrInvalidEndpoint is returned if an endpoint is empty or malformed
	ErrInvalidEndpoint = errors.New("container: invalid endpoint")
	// ErrInvalidTenantID is returned if a TenantID is empty or malformed
	ErrInvalidTenantID = errors.New("container: invalid tenant id")
	// ErrInvalidServiceID is returned if a ServiceID is empty or malformed
	ErrInvalidServiceID = errors.New("container: invalid service id")
	// ErrInvalidService is returned if a Service is empty or malformed
	ErrInvalidService = errors.New("container: invalid serviced")
	// ErrInvalidHostID is returned if the host is empty or malformed
	ErrInvalidHostID = errors.New("container: invalid host id")
	// ErrInvalidZkDSN is returned if the zkDSN is empty or malformed
	ErrInvalidZkDSN = errors.New("container: invalid zookeeper dsn")
	// ErrInvalidExportedEndpoints is returned if the ExportedEndpoints is empty or malformed
	ErrInvalidExportedEndpoints = errors.New("container: invalid exported endpoints")
	// ErrInvalidImportedEndpoints is returned if the ImportedEndpoints is empty or malformed
	ErrInvalidImportedEndpoints = errors.New("container: invalid imported endpoints")
)

// containerEnvironmentFile writes out all the environment variables passed to the container so
// that programs that switch users can access those environment strings
const containerEnvironmentFile = "/etc/profile.d/controlcenter.sh"

// ControllerOptions are options to be run when starting a new proxy server
type ControllerOptions struct {
	ServicedEndpoint string
	Service          struct {
		ID          string   // The uuid of the service to launch
		InstanceID  string   // The running instance ID
		Autorestart bool     // Controller will restart the service if it exits
		Command     []string // The command to launch
	}
	Mux struct { // TCPMUX configuration: RFC 1078
		Enabled     bool   // True if muxing is used
		Port        int    // the TCP port to use
		TLS         bool   // True if TLS is used
		KeyPEMFile  string // Path to the key file when TLS is used
		CertPEMFile string // Path to the cert file when TLS is used
	}
	Logforwarder struct { // Logforwarder configuration
		Enabled    bool   // True if enabled
		Path       string // Path to the logforwarder program
		ConfigFile string // Path to the config file for logstash-forwarder
	}
	Metric struct {
		Address       string // TCP port to host the metric service, :22350
		RemoteEndoint string // The url to forward metric queries
	}
}

// Controller is a object to manage the operations withing a container. For example,
// it creates the managed service instance, logstash forwarding, port forwarding, etc.
type Controller struct {
	options            ControllerOptions
	hostID             string
	tenantID           string
	metricForwarder    *MetricForwarder
	logforwarder       *subprocess.Instance
	logforwarderExited chan error
	closing            chan chan error
	prereqs            []domain.Prereq
	zkDSN              string
	cclient            *coordclient.Client
	zkConn             coordclient.Connection
	exportedEndpoints  map[string][]export
	importedEndpoints  map[string]importedEndpoint
}

type export struct {
	endpoint     *dao.ApplicationEndpoint
	vhosts       []string
	instanceID   int
	endpointName string
}

type importedEndpoint struct {
	endpoint *dao.ApplicationEndpoint
}

// Close shuts down the controller
func (c *Controller) Close() error {
	errc := make(chan error)
	c.closing <- errc
	return <-errc
}

// getService retrieves a service

func getService(lbClientPort string, serviceID string) (*service.Service, error) {
	client, err := serviced.NewLBClient(lbClientPort)
	if err != nil {
		glog.Errorf("Could not create a client to endpoint: %s, %s", lbClientPort, err)
		return nil, err
	}
	defer client.Close()

	var service service.Service
	err = client.GetService(serviceID, &service)
	if err != nil {
		glog.Errorf("Error getting service %s  error: %s", serviceID, err)
		return nil, err
	}

	glog.V(1).Infof("getService: service id=%s: %+v", serviceID, service)
	return &service, nil
}

// getServiceTenaneID retrieves a service's tenantID
func getServiceTenantID(lbClientPort string, serviceID string) (string, error) {
	client, err := serviced.NewLBClient(lbClientPort)
	if err != nil {
		glog.Errorf("Could not create a client to endpoint: %s, %s", lbClientPort, err)
		return "", err
	}
	defer client.Close()

	var tenantID string
	err = client.GetTenantId(serviceID, &tenantID)
	if err != nil {
		glog.Errorf("Error getting service %s's tenantID, error: %s", serviceID, err)
		return "", err
	}

	glog.V(1).Infof("getServiceTenantID: service id=%s: %s", serviceID, tenantID)
	return tenantID, nil
}

// getAgentHostID retrieves the agent's host id
func getAgentHostID(lbClientPort string) (string, error) {
	client, err := serviced.NewLBClient(lbClientPort)
	if err != nil {
		glog.Errorf("Could not create a client to endpoint: %s, %s", lbClientPort, err)
		return "", err
	}
	defer client.Close()

	var hostID string
	err = client.GetHostID(&hostID)
	if err != nil {
		glog.Errorf("Error getting host id, error: %s", err)
		return "", err
	}

	glog.V(1).Infof("getAgentHostID: %s", hostID)
	return hostID, nil
}

// getAgentZkDSN retrieves the agent's zookeeper dsn
func getAgentZkDSN(lbClientPort string) (string, error) {
	client, err := serviced.NewLBClient(lbClientPort)
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

// chownConfFile sets the owner and permissions for a file
func chownConfFile(filename, owner, permissions string) error {

	runCommand := func(exe, arg, filename string) error {
		command := exec.Command(exe, arg, filename)
		output, err := command.CombinedOutput()
		if err != nil {
			glog.Errorf("Error running command:'%v' output: %s  error: %s\n", command, output, err)
			return err
		}
		glog.Infof("Successfully ran command:'%v' output: %s\n", command, output)
		return nil
	}

	if owner != "" {
		if err := runCommand("chown", owner, filename); err != nil {
			return err
		}
	}
	if permissions != "" {
		if err := runCommand("chmod", permissions, filename); err != nil {
			return err
		}
	}

	return nil
}

// writeConfFile writes a config file
func writeConfFile(config servicedefinition.ConfigFile) error {
	// write file with default perms
	if err := os.MkdirAll(filepath.Dir(config.Filename), 0755); err != nil {
		glog.Errorf("could not create directories for config file: %s", config.Filename)
		return err
	}
	if err := ioutil.WriteFile(config.Filename, []byte(config.Content), os.FileMode(0664)); err != nil {
		glog.Errorf("Could not write out config file %s", config.Filename)
		return err
	}
	glog.Infof("Wrote config file %s", config.Filename)

	// change owner and permissions
	if err := chownConfFile(config.Filename, config.Owner, config.Permissions); err != nil {
		return err
	}

	return nil
}

// setupConfigFiles sets up config files
func setupConfigFiles(service *service.Service) error {
	// write out config files
	for _, config := range service.ConfigFiles {
		err := writeConfFile(config)
		if err != nil {
			return err
		}
	}
	return nil
}

// setupLogstashFiles sets up logstash files
func setupLogstashFiles(service *service.Service, resourcePath string) error {
	// write out logstash files
	if len(service.LogConfigs) != 0 {
		err := writeLogstashAgentConfig(logstashContainerConfig, service, resourcePath)
		if err != nil {
			return err
		}
	}
	return nil
}

// getServiceState gets the service state for a serviceID
func getServiceState(conn coordclient.Connection, serviceID string) (*servicestate.ServiceState, error) {
	for {
		var serviceStates []*servicestate.ServiceState
		err := zzk.GetServiceStates(conn, &serviceStates, serviceID)
		if err != nil {
			glog.Errorf("Unable to retrieve running service (%s) states: %v", serviceID, err)
			return nil, nil
		}

		if len(serviceStates) > 0 {
			if serviceStates[0].PrivateIP != "" {
				return serviceStates[0], nil
			}
		}

		glog.Info("Polling to retrieve service state with valid PrivateIP")
		time.Sleep(1 * time.Second)
	}

	return nil, fmt.Errorf("unable to retrieve service state")
}

// buildExportedEndpoints
func buildExportedEndpoints(conn coordclient.Connection, tenantID string, service *service.Service) (map[string][]export, error) {
	result := make(map[string][]export)

	state, err := getServiceState(conn, service.Id)
	if err != nil {
		return result, err
	}

	glog.Infof("buildExportedEndpoints state: %+v", state)

	for _, defep := range state.Endpoints {
		if defep.Purpose == "export" {

			exp := export{}
			exp.vhosts = defep.VHosts
			exp.endpointName = defep.Name
			exp.instanceID = state.InstanceID

			exp.endpoint, err = buildApplicationEndpoint(state, &defep)
			if err != nil {
				return result, err
			}

			key := registry.TenantEndpointKey(tenantID, exp.endpoint.Application)
			if _, exists := result[key]; !exists {
				result[key] = make([]export, 0)
			}
			result[key] = append(result[key], exp)
		}
	}

	return result, nil
}

// buildImportedEndpoints
func buildImportedEndpoints(conn coordclient.Connection, tenantID string, service *service.Service) (map[string]importedEndpoint, error) {
	result := make(map[string]importedEndpoint)

	state, err := getServiceState(conn, service.Id)
	if err != nil {
		return result, err
	}

	glog.Infof("buildImportedEndpoints state: %+v", state)

	for _, defep := range state.Endpoints {
		if defep.Purpose == "import" {
			ie := importedEndpoint{}
			ie.endpoint, err = buildApplicationEndpoint(state, &defep)
			if err != nil {
				return result, err
			}

			tenantEndpointKey := registry.TenantEndpointKey(tenantID, ie.endpoint.Application)
			result[tenantEndpointKey] = ie
		}
	}

	return result, nil
}

//
func buildApplicationEndpoint(state *servicestate.ServiceState, endpoint *service.ServiceEndpoint) (*dao.ApplicationEndpoint, error) {
	var ae dao.ApplicationEndpoint

	ae.ServiceID = state.ServiceID
	ae.Application = endpoint.Application
	ae.Protocol = endpoint.Protocol
	ae.ContainerIP = state.PrivateIP
	ae.ContainerPort = endpoint.PortNumber
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

	return &ae, nil
}

// getZkConnection returns the zookeeper connection
func (c *Controller) getZkConnection() (coordclient.Connection, error) {
	if c.cclient == nil {
		var err error
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

// NewController creates a new Controller for the given options
func NewController(options ControllerOptions) (*Controller, error) {
	c := &Controller{
		options: options,
	}
	c.closing = make(chan chan error)

	if len(options.ServicedEndpoint) <= 0 {
		return nil, ErrInvalidEndpoint
	}

	// get service
	service, err := getService(options.ServicedEndpoint, options.Service.ID)
	if err != nil {
		glog.Errorf("Invalid service from serviceID:%s", options.Service.ID)
		return c, ErrInvalidService
	}

	// create config files
	if err := setupConfigFiles(service); err != nil {
		glog.Errorf("Could not setup config files error:%s", err)
		return c, fmt.Errorf("container: invalid ConfigFiles error:%s", err)
	}

	// get service tenantID
	c.tenantID, err = getServiceTenantID(options.ServicedEndpoint, options.Service.ID)
	if err != nil {
		glog.Errorf("Invalid tenantID from serviceID:%s", options.Service.ID)
		return c, ErrInvalidTenantID
	}

	// get host id
	c.hostID, err = getAgentHostID(options.ServicedEndpoint)
	if err != nil {
		glog.Errorf("Invalid hostID")
		return c, ErrInvalidHostID
	}

	if options.Logforwarder.Enabled {
		if err := setupLogstashFiles(service, filepath.Dir(options.Logforwarder.Path)); err != nil {
			glog.Errorf("Could not setup logstash files error:%s", err)
			return c, fmt.Errorf("container: invalid LogStashFiles error:%s", err)
		}

		// make sure we pick up any logfile that was modified within the
		// last three years
		// TODO: Either expose the 3 years a configurable or get rid of it
		logforwarder, exited, err := subprocess.New(time.Second,
			nil,
			options.Logforwarder.Path,
			"-old-files-hours=26280",
			"-config", options.Logforwarder.ConfigFile)
		if err != nil {
			return nil, err
		}
		c.logforwarder = logforwarder
		c.logforwarderExited = exited
	}

	//build metric redirect url -- assumes 8444 is port mapped
	metricRedirect := options.Metric.RemoteEndoint
	if len(metricRedirect) == 0 {
		glog.V(1).Infof("container.Controller does not have metric forwarding")
	} else {
		if len(c.tenantID) <= 0 {
			return nil, ErrInvalidTenantID
		}
		if len(c.hostID) <= 0 {
			return nil, ErrInvalidHostID
		}
		if len(options.Service.ID) <= 0 {
			return nil, ErrInvalidServiceID
		}

		metricRedirect += "?controlplane_tenant_id=" + c.tenantID
		metricRedirect += "&controlplane_service_id=" + options.Service.ID
		metricRedirect += "&controlplane_host_id=" + c.hostID
		metricRedirect += "&controlplane_instance_id=" + options.Service.InstanceID

		//build and serve the container metric forwarder
		forwarder, err := NewMetricForwarder(options.Metric.Address, metricRedirect)
		if err != nil {
			return c, err
		}
		c.metricForwarder = forwarder
	}

	// Keep a copy of the service prerequisites in the Controller object.
	c.prereqs = service.Prereqs

	// get zookeeper connection string
	c.zkDSN, err = getAgentZkDSN(options.ServicedEndpoint)
	if err != nil {
		glog.Errorf("Invalid zk dsn")
		return nil, ErrInvalidZkDSN
	}

	// get zookeeper connection
	conn, err := c.getZkConnection()
	if err != nil {
		return c, err
	}

	// Keep a copy of the service EndPoint exports
	c.exportedEndpoints, err = buildExportedEndpoints(conn, c.tenantID, service)
	if err != nil {
		glog.Errorf("Invalid ExportedEndpoints")
		return c, ErrInvalidExportedEndpoints
	}

	// initialize importedEndpoints
	if useImportedEndpointServiceDiscovery {
		c.importedEndpoints, err = buildImportedEndpoints(conn, c.tenantID, service)
		if err != nil {
			glog.Errorf("Invalid ImportedEndpoints")
			return c, ErrInvalidImportedEndpoints
		}
	} else {
		c.importedEndpoints = make(map[string]importedEndpoint)
	}

	// check command
	glog.Infof("command: %v [%d]", options.Service.Command, len(options.Service.Command))
	if len(options.Service.Command) < 1 {
		glog.Errorf("Invalid commandif ")
		return c, ErrInvalidCommand
	}

	return c, nil
}

func writeEnvFile(env []string) (err error) {
	fo, err := os.Create(containerEnvironmentFile)
	if err != nil {
		glog.Errorf("Could not create container environment file '%s': %s", containerEnvironmentFile, err)
		return err
	}
	defer func() {
		if err != nil {
			fo.Close()
		} else {
			err = fo.Close()
		}
	}()
	w := bufio.NewWriter(fo)
	for _, value := range env {
		if strings.HasPrefix(value, "HOME=") {
			continue
		}
		w.WriteString("export ")
		w.WriteString(value)
		w.WriteString("\n")
	}
	if err := w.Flush(); err != nil {
		return err
	}
	return err
}

// Run executes the controller's main loop and block until the service exits
// according to it's restart policy or Close() is called.
func (c *Controller) Run() (err error) {

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	env := os.Environ()
	env = append(env, "CONTROLPLANE=1")
	env = append(env, fmt.Sprintf("CONTROLPLANE_CONSUMER_URL=http://localhost%s/api/metrics/store", c.options.Metric.Address))
	env = append(env, fmt.Sprintf("CONTROLPLANE_HOST_ID=%s", c.hostID))
	env = append(env, fmt.Sprintf("CONTROLPLANE_TENANT_ID=%s", c.tenantID))

	if err := writeEnvFile(env); err != nil {
		return err
	}

	args := []string{"-c", "exec " + strings.Join(c.options.Service.Command, " ")}

	startService := func() (*subprocess.Instance, chan error) {
		service, serviceExited, _ := subprocess.New(time.Second*10, env, "/bin/sh", args...)
		return service, serviceExited
	}

	prereqsPassed := make(chan bool)
	var startAfter <-chan time.Time
	service := &subprocess.Instance{}
	serviceExited := make(chan error, 1)
	c.handleRemotePorts()
	if useImportedEndpointServiceDiscovery {
		c.watchRemotePorts()
	}
	go c.checkPrereqs(prereqsPassed)
	healthExits := c.kickOffHealthChecks()
	doRegisterEndpoints := true
	for {
		select {
		case sig := <-sigc:
			switch sig {
			case syscall.SIGTERM:
				c.options.Service.Autorestart = false
			case syscall.SIGQUIT:
				c.options.Service.Autorestart = false
			case syscall.SIGINT:
				c.options.Service.Autorestart = false
			}
			glog.Infof("notifying subprocess of signal %v", sig)
			service.Notify(sig)
			select {
			case <-serviceExited:
				return
			default:
			}

		case <-prereqsPassed:
			startAfter = time.After(time.Millisecond * 1)

		case <-time.After(time.Second * 10):
			if !useImportedEndpointServiceDiscovery {
				c.handleRemotePorts()
			}

		case exitError := <-serviceExited:
			glog.Infof("Service process exited.")
			if !c.options.Service.Autorestart {
				exitStatus := getExitStatus(exitError)
				os.Exit(exitStatus)
			}
			glog.Infof("Restarting service process in 10 seconds.")
			startAfter = time.After(time.Second * 10)

		case <-startAfter:
			glog.Infof("Starting service process.")
			service, serviceExited = startService()
			if doRegisterEndpoints {
				c.registerExportedEndpoints()
				doRegisterEndpoints = false
			}
			startAfter = nil
		}
	}
	for _, exitChannel := range healthExits {
		exitChannel <- true
	}
	return
}

func getExitStatus(err error) int {
	if err != nil {
		if e, ok := err.(*exec.ExitError); ok {
			if status, ok := e.Sys().(syscall.WaitStatus); ok {
				return status.ExitStatus()
			}
		}
	}
	return 0
}

func (c *Controller) checkPrereqs(prereqsPassed chan bool) error {
	if len(c.prereqs) == 0 {
		glog.Infof("No prereqs to pass.")
		prereqsPassed <- true
		return nil
	}
	for _ = range time.Tick(1 * time.Second) {
		failedAny := false
		for _, script := range c.prereqs {
			cmd := exec.Command("sh", "-c", script.Script)
			err := cmd.Run()
			if err != nil {
				glog.Warningf("Failed prereq [%s], not starting service.", script.Name)
				failedAny = true
				break
			} else {
				glog.Infof("Passed prereq [%s].", script.Name)
			}
		}
		if !failedAny {
			glog.Infof("Passed all prereqs.")
			prereqsPassed <- true
			return nil
		}
	}
	return nil
}

func (c *Controller) kickOffHealthChecks() map[string]chan bool {
	exitChannels := make(map[string]chan bool)
	client, err := serviced.NewLBClient(c.options.ServicedEndpoint)
	if err != nil {
		glog.Errorf("Could not create a client to endpoint: %s, %s", c.options.ServicedEndpoint, err)
		return nil
	}
	defer client.Close()
	var healthChecks map[string]domain.HealthCheck
	err = client.GetHealthCheck(c.options.Service.ID, &healthChecks)
	if err != nil {
		glog.Errorf("Error getting health checks: %s", err)
		return nil
	}
	for key, mapping := range healthChecks {
		glog.Infof("Kicking off health check %s.", key)
		exitChannels[key] = make(chan bool)
		go c.handleHealthCheck(key, mapping.Script, mapping.Interval, exitChannels[key])
	}
	return exitChannels
}

func (c *Controller) handleHealthCheck(name string, script string, interval time.Duration, exitChannel chan bool) {
	client, err := serviced.NewLBClient(c.options.ServicedEndpoint)
	if err != nil {
		glog.Errorf("Could not create a client to endpoint: %s, %s", c.options.ServicedEndpoint, err)
		return
	}
	defer client.Close()
	scriptFile, err := ioutil.TempFile("", name)
	if err != nil {
		glog.Errorf("Error creating temporary file for health check %s: %s", name, err)
		return
	}
	defer scriptFile.Close()
	defer os.Remove(scriptFile.Name())
	err = ioutil.WriteFile(scriptFile.Name(), []byte(script), os.FileMode(0777))
	if err != nil {
		glog.Errorf("Error writing script for health check %s: %s", name, err)
		return
	}
	scriptFile.Close()
	err = os.Chmod(scriptFile.Name(), os.FileMode(0777))
	if err != nil {
		glog.Errorf("Error setting script executable for health check %s: %s", name, err)
		return
	}
	var unused int
	for {
		select {
		case <-time.After(interval):
			cmd := exec.Command("sh", "-c", scriptFile.Name())
			err = cmd.Run()
			if err == nil {
				glog.V(4).Infof("Health check %s succeeded.", name)
				_ = client.LogHealthCheck(domain.HealthCheckResult{c.options.Service.ID, name, time.Now().String(), "passed"}, &unused)
			} else {
				glog.Warningf("Health check %s failed.", name)
				_ = client.LogHealthCheck(domain.HealthCheckResult{c.options.Service.ID, name, time.Now().String(), "failed"}, &unused)
			}
		case <-exitChannel:
			return
		}
	}
}

func (c *Controller) handleRemotePorts() {
	// get service endpoints
	client, err := serviced.NewLBClient(c.options.ServicedEndpoint)
	if err != nil {
		glog.Errorf("Could not create a client to endpoint: %s, %s", c.options.ServicedEndpoint, err)
		return
	}
	defer client.Close()

	var endpoints map[string][]*dao.ApplicationEndpoint
	err = client.GetServiceEndpoints(c.options.Service.ID, &endpoints)
	if err != nil {
		glog.Errorf("Error getting application endpoints for service %s: %s", c.options.Service.ID, err)
		return
	}

	addImportedEndpoint := func(endpoint *dao.ApplicationEndpoint) {
		// replace or add entries in importedEndpoints
		ie := importedEndpoint{}
		ie.endpoint = endpoint
		key := registry.TenantEndpointKey(c.tenantID, ie.endpoint.Application)
		c.importedEndpoints[key] = ie
	}

	emptyAddressList := []string{}
	for key, endpointList := range endpoints {
		if len(endpointList) <= 0 {
			if proxy, ok := proxies[key]; ok {
				proxy.SetNewAddresses(emptyAddressList)
			}
			continue
		}

		addresses := make([]string, len(endpointList))
		for i, endpoint := range endpointList {
			addresses[i] = fmt.Sprintf("%s:%d", endpoint.HostIP, endpoint.HostPort)
			glog.V(2).Infof("addresses[%d]:%-20s  endpoints[%s]: %+v", i, addresses[i], key, *endpoint)
		}
		sort.Strings(addresses)

		var (
			prxy *proxy
			ok   bool
		)

		if prxy, ok = proxies[key]; !ok {
			glog.Infof("Attempting port map for: %s -> %+v", key, *endpointList[0])

			// setup a new proxy
			listener, err := net.Listen("tcp4", fmt.Sprintf(":%d", endpointList[0].ContainerPort))
			if err != nil {
				glog.Errorf("Could not bind to port: %s", err)
				continue
			}
			prxy, err = newProxy(
				fmt.Sprintf("%v", endpointList[0]),
				uint16(c.options.Mux.Port),
				c.options.Mux.TLS,
				listener)
			if err != nil {
				glog.Errorf("Could not build proxy %s", err)
				continue
			}

			glog.Infof("Success binding port: %s -> %+v", key, prxy)
			proxies[key] = prxy

			if ep := endpointList[0]; ep.VirtualAddress != "" {
				p := strconv.FormatUint(uint64(ep.ContainerPort), 10)
				err := vifs.RegisterVirtualAddress(ep.VirtualAddress, p, ep.Protocol)
				if err != nil {
					glog.Errorf("Error creating virtual address: %+v", err)
				}
			}
		}
		prxy.SetNewAddresses(addresses)

		if useImportedEndpointServiceDiscovery {
			addImportedEndpoint(endpointList[0])

			// TODO: agent needs to register controlplane and controlplane_consumer
			//       but don't do that here in the container code
		}
	}
}

//
func (c *Controller) watchRemotePorts() {
	/*
		watch each tenant endpoint
		    - when endpoints are added, add the endpoint proxy if not already added
			- when endpoints are added, add watch on that endpoint for updates
			TODO: when endpoints are deleted, tell that endpoint proxy to stop proxying
			- when endpoints are deleted, may not need to deal with removing watch on that endpoint since that watch will block forever
			TODO: deal with import regexes, i.e mysql_.*
		- may not need to initially deal with removal of tenant endpoint
	*/
	glog.Infof("watchRemotePorts starting")

	cMuxPort = uint16(c.options.Mux.Port)
	cMuxTLS = c.options.Mux.TLS

	for key, endpoint := range c.importedEndpoints {
		glog.Infof("importedEndpoints[%s]: %+v", key, *endpoint.endpoint)
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
		glog.Infof("processTenantEndpoints for path: %s tenantEndpointIDs: %s", parentPath, tenantEndpointIDs)

		watchTenantEndpoints := func(tenantEndpointKey string) {
			glog.Infof("  watching tenantEndpointKey: %s", tenantEndpointKey)
			if err := endpointRegistry.WatchTenantEndpoint(zkConn, tenantEndpointKey,
				processTenantEndpoint, endpointWatchError); err != nil {
				glog.Errorf("error watching tenantEndpointKey %s: %v", tenantEndpointKey, err)
			}
		}

		for _, id := range tenantEndpointIDs {
			glog.Infof("checking need to watch tenantEndpoint: %s %s", parentPath, id)

			// add watchers if they don't exist for a tenantid_application
			// and if tenant-endpoint is an import
			if _, ok := watchers[id]; !ok {
				if _, ok := c.importedEndpoints[id]; ok {
					watchers[id] = true
					// aSet := set.New(set.ThreadSafe) // thread safe version
					// watchers[id] = aSet.(*set.Set)
					go watchTenantEndpoints(id)
				} else {
					glog.Infof("  no need to add - not imported: %s %s", parentPath, id)
					glog.Infof("    importedEndpoints: %+v", c.importedEndpoints)
				}
			} else {
				glog.Infof("  no need to add - existing watch tenantEndpoint: %s %s", parentPath, id)
			}

			// BEWARE: only need to deal with add, currently no need to deal with deletes
			// since tenant endpoints are currently not deleted.  only the hostid_containerid
			// entries within tenantid_application are added/deleted
		}

	}

	glog.Infof("watching endpointRegistry")
	//TODO: deal with channel if we care
	go endpointRegistry.WatchRegistry(zkConn, make(chan bool), processTenantEndpoints, endpointWatchError)
}

//
func endpointWatchError(path string, err error) {
	glog.Infof("processing endpointWatchError on %s: %v", path, err)
}

//
func processTenantEndpoint(conn coordclient.Connection, parentPath string, hostContainerIDs ...string) {
	glog.Infof("processTenantEndpoint: parentPath:%s hostContainerIDs: %v", parentPath, hostContainerIDs)

	// update the proxy for this tenant endpoint
	endpointRegistry, err := registry.CreateEndpointRegistry(conn)
	if err != nil {
		glog.Errorf("Could not get EndpointRegistry. Endpoints not registered: %v", err)
		return
	}

	parts := strings.Split(parentPath, "/")
	tenantEndpointID := parts[len(parts)-1]

	endpointNodes := make([]registry.EndpointNode, len(hostContainerIDs))
	for ii, hostContainerID := range hostContainerIDs {
		path := fmt.Sprintf("%s/%s", parentPath, hostContainerID)
		endpointNode, err := endpointRegistry.GetItem(conn, path)
		if err != nil {
			glog.Errorf("error getting endpoint node at %s: %v", path, err)
		}
		endpointNodes[ii] = *endpointNode
	}

	setProxyAddresses(tenantEndpointID, endpointNodes)

	/*
		// determine additions and deletions
		setOld := watchers[tenantEndpointID]
		setNew := set.New(set.ThreadSafe) // thread safe version

		// deal with additions
		setAdditions := set.Difference(setNew, setOld)
		additions := set.StringSlice(setAdditions)
		if len(additions) > 0 {
			// TODO: set watch on each hostid_containerid item that was added
			//       update proxy with host_containers from tenant_endpoint when item is changed
		}

		// deal with deletions
		setDeletions := set.Difference(setOld, setNew)
		deletions := set.StringSlice(setDeletions)
		if len(deletions) > 0 {
			// no need to remove watch for deleted items - deleted watches block forever
		}
	*/
}

func setProxyAddresses(tenantEndpointID string, endpointNodes []registry.EndpointNode) {
	glog.Infof("starting setProxyAddresses(tenantEndpointID: %s)", tenantEndpointID)

	if len(endpointNodes) <= 0 {
		if proxy, ok := proxies[tenantEndpointID]; ok {
			glog.Errorf("Setting proxy %s to empty address list", tenantEndpointID)
			emptyAddressList := []string{}
			proxy.SetNewAddresses(emptyAddressList)
		}
		return
	}

	addresses := make([]string, len(endpointNodes))
	for ii, endpoint := range endpointNodes {
		addresses[ii] = fmt.Sprintf("%s:%d", endpoint.HostIP, endpoint.HostPort)
		glog.Infof("addresses[%d]: %s  endpoint: %+v", ii, addresses[ii], endpoint)
	}
	sort.Strings(addresses)

	for ii, pp := range proxies {
		glog.Infof("  proxies[%s]: %+v", ii, *pp)
	}

	prxy, ok := proxies[tenantEndpointID]
	if !ok {
		var err error
		proxies[tenantEndpointID], err = createNewProxy(tenantEndpointID, endpointNodes[0])
		if err != nil {
			return
		}

		if ep := endpointNodes[0]; ep.VirtualAddress != "" {
			p := strconv.FormatUint(uint64(ep.ContainerPort), 10)
			err := vifs.RegisterVirtualAddress(ep.VirtualAddress, p, ep.Protocol)
			if err != nil {
				glog.Errorf("Error creating virtual address: %+v", err)
			}
		}
	}
	glog.Infof("Starting setting proxy %s to addresses %v", tenantEndpointID, addresses)
	prxy.SetNewAddresses(addresses)
	glog.Infof("Finished setting proxy %s to addresses %v", tenantEndpointID, addresses)
}

//
func createNewProxy(key string, endpoint registry.EndpointNode) (*proxy, error) {
	glog.Infof("Attempting port map for: %s -> %+v", key, endpoint)

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

	glog.Infof("Success binding port: %s -> %+v", key, prxy)
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
	// get containerID
	containerID := os.Getenv("HOSTNAME")
	glog.Infof("containerID: %s\n", containerID)

	// register exported endpoints
	for key, exportList := range c.exportedEndpoints {
		for _, export := range exportList {
			endpoint := export.endpoint
			for _, vhost := range export.vhosts {
				epName := fmt.Sprintf("%s_%v", export.endpointName, export.instanceID)
				var path string
				if path, err = vhostRegistry.SetItem(conn, vhost, registry.NewVhostEndpoint(epName, *endpoint)); err != nil {
					glog.Errorf("could not register vhost %s for %s: %v", vhost, epName, err)
				}
				glog.Infof("Registered vhost %s for %s at %s", vhost, epName, path)
			}

			glog.Infof("Registering exported endpoint[%s]: %+v", key, *endpoint)
			path, err := endpointRegistry.SetItem(conn, c.tenantID, export.endpoint.Application, c.hostID, containerID, registry.NewEndpointNode(c.tenantID, export.endpoint.Application, c.hostID, containerID, *endpoint))
			if err != nil {
				glog.Errorf("  unable to add endpoint: %+v %v", *endpoint, err)
				continue
			}

			glog.V(1).Infof("  endpoint successfully added to path: %s", path)
		}
	}
}

var (
	proxies  map[string]*proxy
	vifs     *VIFRegistry
	nextip   int
	watchers map[string]bool
	// watchers map[string]*set.Set
	cMuxPort uint16 // the TCP port to use
	cMuxTLS  bool
)

func init() {
	proxies = make(map[string]*proxy)
	vifs = NewVIFRegistry()
	nextip = 1
	watchers = make(map[string]bool)
	// watchers = make(map[string]*set.Set)
}
