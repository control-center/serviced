package container

import (
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/commons/subprocess"
	coordclient "github.com/zenoss/serviced/coordinator/client"
	"github.com/zenoss/serviced/dao"
	"github.com/zenoss/serviced/domain"
	"github.com/zenoss/serviced/domain/service"
	"github.com/zenoss/serviced/domain/servicedefinition"
	"github.com/zenoss/serviced/node"
	"github.com/zenoss/serviced/utils"
	"github.com/zenoss/serviced/zzk/registry"

	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

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
	VirtualAddressSubnet string // The subnet of virtual addresses, 10.3
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

// NewController creates a new Controller for the given options
func NewController(options ControllerOptions) (*Controller, error) {
	c := &Controller{
		options: options,
	}
	c.closing = make(chan chan error)

	if len(options.ServicedEndpoint) <= 0 {
		return nil, ErrInvalidEndpoint
	}

	// set vifs subnet
	if err := vifs.SetSubnet(options.VirtualAddressSubnet); err != nil {
		glog.Errorf("Could not set VirtualAddressSubnet:%s %s", options.VirtualAddressSubnet, err)
		return c, fmt.Errorf("container: invalid VirtualAddressSubnet:%s error:%s", options.VirtualAddressSubnet, err)
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

	// get endpoints
	if err := c.getEndpoints(service); err != nil {
		return c, err
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
	c.handleControlCenterImports()
	c.watchRemotePorts()
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

		case exitError := <-serviceExited:
			if !c.options.Service.Autorestart {
				exitStatus, _ := utils.GetExitStatus(exitError)
				glog.Infof("Exiting with status:%d due to %+v", exitStatus, exitError)
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
				glog.Warningf("Not starting service yet, waiting on prereq: %s", script.Name)
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

func (c *Controller) handleControlCenterImports() {
	// this function is currently needed to handle special control plane imports
	// from GetServiceEndpoints() that does not exist in endpoints from getServiceState

	// get service endpoints
	client, err := node.NewLBClient(c.options.ServicedEndpoint)
	if err != nil {
		glog.Errorf("Could not create a client to endpoint: %s, %s", c.options.ServicedEndpoint, err)
		return
	}
	defer client.Close()

	// TODO: instead of getting all endpoints, via GetServiceEndpoints(), create a new call
	//       that returns only special "controlplane" imported endpoints
	var endpoints map[string][]*dao.ApplicationEndpoint
	err = client.GetServiceEndpoints(c.options.Service.ID, &endpoints)
	if err != nil {
		glog.Errorf("Error getting application endpoints for service %s: %s", c.options.Service.ID, err)
		return
	}

	// convert keys set by GetServiceEndpoints to tenantID_endpointID
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

	for key, endpointList := range endpoints {
		// ignore endpoints that are not special controlplane imports
		ignorePrefix := fmt.Sprintf("%s_controlplane", c.tenantID)
		if !strings.HasPrefix(key, ignorePrefix) {
			continue
		}

		// set proxy addresses
		setProxyAddresses(key, endpointList, endpointList[0].VirtualAddress)

		// add/replace entries in importedEndpoints
		setImportedEndpoint(&c.importedEndpoints, c.tenantID, endpointList[0].Application, endpointList[0].VirtualAddress)

		// TODO: agent needs to register controlplane and controlplane_consumer
		//       but don't do that here in the container code
	}
}
