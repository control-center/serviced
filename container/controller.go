package container

import (
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/commons/subprocess"
	coordclient "github.com/zenoss/serviced/coordinator/client"
	"github.com/zenoss/serviced/dao"
	"github.com/zenoss/serviced/domain"
	"github.com/zenoss/serviced/domain/service"
	"github.com/zenoss/serviced/domain/servicedefinition"
	"github.com/zenoss/serviced/domain/servicestate"
	"github.com/zenoss/serviced/node"
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
	"regexp"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// TODO: remove useImportedEndpointServiceDiscovery or set it to true
const useImportedEndpointServiceDiscovery = true

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
	dockerID           string
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
	endpointID     string
	virtualAddress string
}

// Close shuts down the controller
func (c *Controller) Close() error {
	errc := make(chan error)
	c.closing <- errc
	return <-errc
}

// getService retrieves a service
func getService(lbClientPort string, serviceID string) (*service.Service, error) {
	client, err := node.NewLBClient(lbClientPort)
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

// getServiceTenantID retrieves a service's tenantID
func getServiceTenantID(lbClientPort string, serviceID string) (string, error) {
	client, err := node.NewLBClient(lbClientPort)
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
	client, err := node.NewLBClient(lbClientPort)
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
func getServiceState(conn coordclient.Connection, serviceID, instanceIDStr string) (*servicestate.ServiceState, error) {

	tmpID, err := strconv.Atoi(instanceIDStr)
	if err != nil {
		glog.Errorf("Unable to interpret InstanceID: %s", instanceIDStr)
		return nil, err
	}
	instanceID := int(tmpID)

	for {
		var serviceStates []*servicestate.ServiceState
		err := zzk.GetServiceStates(conn, &serviceStates, serviceID)
		if err != nil {
			glog.Errorf("Unable to retrieve running service (%s) states: %v", serviceID, err)
			return nil, nil
		}

		for ii, ss := range serviceStates {
			if ss.InstanceID == instanceID && ss.PrivateIP != "" {
				return serviceStates[ii], nil
			}
		}

		glog.Infof("Polling to retrieve service state instanceID:%d with valid PrivateIP", instanceID)
		time.Sleep(1 * time.Second)
	}

	return nil, fmt.Errorf("unable to retrieve service state")
}

// buildExportedEndpoints builds the map to exported endpoints
func buildExportedEndpoints(conn coordclient.Connection, tenantID string, state *servicestate.ServiceState) (map[string][]export, error) {
	glog.Infof("buildExportedEndpoints state: %+v", state)
	result := make(map[string][]export)

	for _, defep := range state.Endpoints {
		if defep.Purpose == "export" {

			exp := export{}
			exp.vhosts = defep.VHosts
			exp.endpointName = defep.Name
			exp.instanceID = state.InstanceID

			var err error
			exp.endpoint, err = buildApplicationEndpoint(state, &defep)
			if err != nil {
				return result, err
			}

			key := registry.TenantEndpointKey(tenantID, exp.endpoint.Application)
			if _, exists := result[key]; !exists {
				result[key] = make([]export, 0)
			}
			result[key] = append(result[key], exp)

			glog.Infof("  cached exported endpoint[%s]: %+v", key, exp)
		}
	}

	return result, nil
}

// buildImportedEndpoints builds the map to imported endpoints
func buildImportedEndpoints(conn coordclient.Connection, tenantID string, state *servicestate.ServiceState) (map[string]importedEndpoint, error) {
	glog.Infof("buildImportedEndpoints state: %+v", state)
	result := make(map[string]importedEndpoint)

	for _, defep := range state.Endpoints {
		if defep.Purpose == "import" {
			endpoint, err := buildApplicationEndpoint(state, &defep)
			if err != nil {
				return result, err
			}

			ie := importedEndpoint{}
			ie.endpointID = endpoint.Application
			ie.virtualAddress = endpoint.VirtualAddress

			tenantEndpointKey := registry.TenantEndpointKey(tenantID, ie.endpointID)
			result[tenantEndpointKey] = ie

			glog.Infof("  cached imported endpoint[%s]: %+v", tenantEndpointKey, ie)
		}
	}

	return result, nil
}

// buildApplicationEndpoint converts a ServiceEndpoint to an ApplicationEndpoint
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

	glog.Infof("  built ApplicationEndpoint: %+v", ae)

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

	// get service state
	sstate, err := getServiceState(conn, service.Id, options.Service.InstanceID)
	if err != nil {
		return c, err
	}
	c.dockerID = sstate.DockerID

	// keep a copy of the service EndPoint exports
	c.exportedEndpoints, err = buildExportedEndpoints(conn, c.tenantID, sstate)
	if err != nil {
		glog.Errorf("Invalid ExportedEndpoints")
		return c, ErrInvalidExportedEndpoints
	}

	// initialize importedEndpoints
	if useImportedEndpointServiceDiscovery {
		c.importedEndpoints, err = buildImportedEndpoints(conn, c.tenantID, sstate)
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
	env = append(env, fmt.Sprintf("CONTROLPLANE_INSTANCE_ID=%s", c.options.Service.InstanceID))

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
	client, err := node.NewLBClient(c.options.ServicedEndpoint)
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
	client, err := node.NewLBClient(c.options.ServicedEndpoint)
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
	// this function is currently needed to handle special control plane imports
	// from GetServiceEndpoints() that does not exist in endpoints from getServiceState

	// get service endpoints
	client, err := node.NewLBClient(c.options.ServicedEndpoint)
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
	if useImportedEndpointServiceDiscovery {
		tmp := make(map[string][]*dao.ApplicationEndpoint)
		for key, endpointList := range endpoints {
			if len(endpointList) <= 0 {
				glog.Warningf("ignoring key: %s with empty endpointList", key)
				continue
			}

			tenantEndpointID := registry.TenantEndpointKey(c.tenantID, endpointList[0].Application)
			glog.Infof("changing key from %s to %s: %+v", key, tenantEndpointID, endpointList[0])
			tmp[tenantEndpointID] = endpoints[key]
		}
		endpoints = tmp
	}

	for key, endpointList := range endpoints {
		if useImportedEndpointServiceDiscovery {
			ignorePrefix := fmt.Sprintf("%s_controlplane", c.tenantID)
			if !strings.HasPrefix(key, ignorePrefix) {
				continue
			}
		}

		setProxyAddresses(key, endpointList, endpointList[0].VirtualAddress)

		if useImportedEndpointServiceDiscovery {
			// add/replace entries in importedEndpoints
			ie := importedEndpoint{}
			ie.endpointID = endpointList[0].Application
			ie.virtualAddress = endpointList[0].VirtualAddress
			key := registry.TenantEndpointKey(c.tenantID, ie.endpointID)
			c.importedEndpoints[key] = ie

			// TODO: agent needs to register controlplane and controlplane_consumer
			//       but don't do that here in the container code
		}
	}
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
	glog.Infof("watchRemotePorts starting")

	cMuxPort = uint16(c.options.Mux.Port)
	cMuxTLS = c.options.Mux.TLS

	for key, endpoint := range c.importedEndpoints {
		glog.Infof("importedEndpoints[%s]: %+v", key, endpoint)
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

		// cancel watcher on top level /endpoints if all watchers on imported endpoints have been set up
		{
			ignorePrefix := fmt.Sprintf("%s_controlplane", c.tenantID)
			missingWatchers := false
			for id, _ := range c.importedEndpoints {
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
				glog.Infof("all imports are being watched - cancelling watcher on /endpoints")
				endpointsWatchCanceller <- true
				return
			}
		}

		// setup watchers for each imported tenant endpoint
		watchTenantEndpoints := func(tenantEndpointKey string) {
			glog.Infof("  watching tenantEndpointKey: %s", tenantEndpointKey)
			if err := endpointRegistry.WatchTenantEndpoint(zkConn, tenantEndpointKey,
				c.processTenantEndpoint, endpointWatchError); err != nil {
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
					go watchTenantEndpoints(id)
				} else {
					// look for imports with regexes that match each tenantEndpointID
					matched := false
					for _, ie := range c.importedEndpoints {
						endpointPattern := fmt.Sprintf("^%s$", registry.TenantEndpointKey(c.tenantID, ie.endpointID))
						glog.Infof("  checking tenantEndpointID %s against pattern %s", id, endpointPattern)
						endpointRegex, err := regexp.Compile(endpointPattern)
						if err != nil {
							glog.Warningf("  unable to check tenantEndpointID %s against imported endpoint %s", id, ie.endpointID)
							continue //Don't spam error message; it was reported at validation time
						}

						if endpointRegex.MatchString(id) {
							glog.Infof("  tenantEndpointID:%s matched imported endpoint pattern:%s for %+v", id, endpointPattern, ie)
							matched = true
							watchers[id] = true
							go watchTenantEndpoints(id)
						}
					}

					if !matched {
						glog.Infof("  no need to add - not imported: %s %s for importedEndpoints: %+v", parentPath, id, c.importedEndpoints)
					}
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
	go endpointRegistry.WatchRegistry(zkConn, endpointsWatchCanceller, processTenantEndpoints, endpointWatchError)
}

// endpointWatchError shows errors with watches
func endpointWatchError(path string, err error) {
	glog.Infof("processing endpointWatchError on %s: %v", path, err)
}

// processTenantEndpoint updates the addresses for an imported endpoint
func (c *Controller) processTenantEndpoint(conn coordclient.Connection, parentPath string, hostContainerIDs ...string) {
	glog.Infof("processTenantEndpoint: parentPath:%s hostContainerIDs: %v", parentPath, hostContainerIDs)

	// update the proxy for this tenant endpoint
	endpointRegistry, err := registry.CreateEndpointRegistry(conn)
	if err != nil {
		glog.Errorf("Could not get EndpointRegistry. Endpoints not registered: %v", err)
		return
	}

	parts := strings.Split(parentPath, "/")
	tenantEndpointID := parts[len(parts)-1]

	endpoints := make([]*dao.ApplicationEndpoint, len(hostContainerIDs))
	for ii, hostContainerID := range hostContainerIDs {
		path := fmt.Sprintf("%s/%s", parentPath, hostContainerID)
		endpointNode, err := endpointRegistry.GetItem(conn, path)
		if err != nil {
			glog.Errorf("error getting endpoint node at %s: %v", path, err)
		}
		endpoints[ii] = &endpointNode.ApplicationEndpoint
	}

	setProxyAddresses(tenantEndpointID, endpoints, c.importedEndpoints[tenantEndpointID].virtualAddress)
}

// setProxyAddresses tells the proxies to update with addresses
func setProxyAddresses(tenantEndpointID string, endpoints []*dao.ApplicationEndpoint, importVirtualAddress string) {
	glog.Infof("starting setProxyAddresses(tenantEndpointID: %s)", tenantEndpointID)

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

	addresses := make([]string, len(endpoints))
	for ii, endpoint := range endpoints {
		addresses[ii] = fmt.Sprintf("%s:%d", endpoint.HostIP, endpoint.HostPort)
		glog.Infof("  addresses[%d]: %s  endpoint: %+v", ii, addresses[ii], endpoint)
	}
	sort.Strings(addresses)
	glog.Infof("  endpoint key:%s addresses:%+v", tenantEndpointID, addresses)

	for ii, pp := range proxies {
		glog.Infof("  proxies[%s]: %+v", ii, *pp)
	}

	prxy, ok := proxies[tenantEndpointID]
	if !ok {
		var err error
		prxy, err = createNewProxy(tenantEndpointID, endpoints[0])
		if err != nil {
			glog.Errorf("error with createNewProxy(%s, %+v) %v", tenantEndpointID, endpoints[0], err)
			return
		}
		proxies[tenantEndpointID] = prxy

		for _, virtualAddress := range []string{importVirtualAddress, endpoints[0].VirtualAddress} {
			if virtualAddress != "" {
				ep := endpoints[0]
				p := strconv.FormatUint(uint64(ep.ContainerPort), 10)
				err := vifs.RegisterVirtualAddress(virtualAddress, p, ep.Protocol)
				if err != nil {
					glog.Errorf("Error creating virtual address: %+v", err)
				}
				glog.Infof("created virtual address %s: %+v", virtualAddress, endpoints)
			}
		}
	}
	glog.Infof("Setting proxy %s to addresses %v", tenantEndpointID, addresses)
	prxy.SetNewAddresses(addresses)
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
				epName := fmt.Sprintf("%s_%v", export.endpointName, export.instanceID)
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
	proxies                 map[string]*proxy
	vifs                    *VIFRegistry
	nextip                  int
	watchers                map[string]bool
	endpointsWatchCanceller chan bool
	// watchers map[string]*set.Set
	cMuxPort uint16 // the TCP port to use
	cMuxTLS  bool
)

func init() {
	proxies = make(map[string]*proxy)
	vifs = NewVIFRegistry()
	nextip = 1
	watchers = make(map[string]bool)
	endpointsWatchCanceller = make(chan bool)
	// watchers = make(map[string]*set.Set)
}
