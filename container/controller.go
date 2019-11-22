// Copyright 2014-2018 The Serviced Authors.
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
	log "github.com/Sirupsen/logrus"
	"github.com/control-center/serviced/auth"
	"github.com/control-center/serviced/commons/proc"
	"github.com/control-center/serviced/commons/subprocess"
	coordclient "github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/domain"
	"github.com/control-center/serviced/domain/applicationendpoint"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/servicedefinition"
	"github.com/control-center/serviced/health"
	"github.com/control-center/serviced/node"
	"github.com/control-center/serviced/rpc/master"
	"github.com/control-center/serviced/utils"
	"github.com/control-center/serviced/zzk"
	"github.com/control-center/serviced/zzk/registry"
	zkservice "github.com/control-center/serviced/zzk/service"
	"github.com/docker/docker/pkg/mount"
	"github.com/zenoss/glog"

	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
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
	// ErrNoServiceEndpoints is returned if we can't fetch the service endpoints
	ErrNoServiceEndpoints = errors.New("container: unable to retrieve service endpoints")
)

const (
	// containerEnvironmentFile writes out all the environment variables passed to the container so
	// that programs that switch users can access those environment strings
	containerEnvironmentFile = "/etc/profile.d/controlcenter.sh"
	// ContainerKeysDir holds the delegate's private key and auth token
	containerDelegateKeyFile = "/etc/serviced/delegate.keys"
	containerTokenFile       = "/etc/serviced/auth.token"
)

// Logforwarder configuration for filebeat
type LogforwarderOptions struct {
	Enabled    bool          // True if enabled
	Path       string        // Path to the logforwarder program (e.g. filebeat)
	ConfigFile string        // Path to the config file for filebeat
	SettleTime time.Duration // time to wait for forwarder to flush its buffer before exiting
}

// ControllerOptions are options to be run when starting a new proxy server
type ControllerOptions struct {
	ServicedEndpoint string
	RPCDisableTLS    bool
	Service          struct {
		ID          string   // The uuid of the service to launch
		InstanceID  string   // The running instance ID
		Autorestart bool     // Controller will restart the service if it exits
		Command     []string // The command to launch
		RunAs       string   // Run command as user or user:group
	}
	Mux struct { // TCPMUX configuration: RFC 1078
		Enabled     bool   // True if muxing is used
		Port        int    // the TCP port to use
		DisableTLS  bool   // True if TLS is disabled
		KeyPEMFile  string // Path to the key file when TLS is used
		CertPEMFile string // Path to the cert file when TLS is used
	}
	Logforwarder LogforwarderOptions
	Metric       struct {
		Address       string // TCP port to host the metric service, :22350
		RemoteEndoint string // The url to forward metric queries
	}
	VirtualAddressSubnet string // The subnet of virtual addresses, 10.3
	MetricForwarding     bool   // Whether or not the Controller should forward metrics
	HostIPs              string // The ip addresses of the host
	ServiceNamePath      string // Path of the service
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
	zkInfo             node.ZkInfo
	zkConn             coordclient.Connection
	PIDFile            string
	exitStatus         int
	endpoints          *ContainerEndpoints
	healthChecks       map[string]health.HealthCheck
	ccApiProxy         *servicedApiProxy
}

// Close shuts down the controller
func (c *Controller) Close() error {
	errc := make(chan error)
	c.closing <- errc
	return <-errc
}

// getService retrieves a service
func getService(lbClientPort string, serviceID string, instanceID int) (*service.Service, string, string, error) {
	client, err := node.NewLBClient(lbClientPort)
	if err != nil {
		glog.Errorf("Could not create a client to endpoint: %s, %s", lbClientPort, err)
		return nil, "", "", err
	}
	defer client.Close()

	var evaluatedServiceResponse node.EvaluateServiceResponse
	err = client.GetEvaluatedService(node.EvaluateServiceRequest{serviceID, instanceID}, &evaluatedServiceResponse)

	if err != nil {
		glog.Errorf("Error getting service %s  error: %s", serviceID, err)
		return nil, "", "", err
	}

	glog.V(2).Infof("getService: serviceID=%s, tenantID=%s, serviceNamePath=%s: %+v", serviceID, evaluatedServiceResponse.TenantID, evaluatedServiceResponse.ServiceNamePath, evaluatedServiceResponse.Service)
	return &evaluatedServiceResponse.Service, evaluatedServiceResponse.TenantID, evaluatedServiceResponse.ServiceNamePath, nil
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
func setupConfigFiles(svc *service.Service) error {
	// write out config files
	for _, config := range svc.ConfigFiles {
		err := writeConfFile(config)
		if err != nil {
			return err
		}
	}
	return nil
}

// setupLogstashFiles sets up logstash files
func setupLogstashFiles(hostID string, hostIPs string, svcPath string, service *service.Service, instanceID string, logforwarderOptions LogforwarderOptions) error {
	// write out logstash files
	if len(service.LogConfigs) != 0 {
		err := writeLogstashAgentConfig(hostID, hostIPs, svcPath, service, instanceID, logforwarderOptions)
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

	// Load keys
	keyshutdown := make(chan interface{})
	go func() {
		// This is a ridiculous shutdown pattern I'll follow for the sake of
		// not destabilizing the controller this late in a release.
		// TODO: Make this not stupid
		errc := <-c.closing
		close(keyshutdown)
		errc <- nil
	}()

	go auth.WatchDelegateKeyFile(containerDelegateKeyFile, keyshutdown)
	go auth.WatchTokenFile(containerTokenFile, keyshutdown)

	// Load the delegate keys and auth tokens first so there's no race btwn starting the watcher routines
	//    and making the first RPC call in getService()
	plog.Debug("Awaiting for the auth token")
	<-auth.WaitForAuthToken(nil)

	// get service
	instanceID, err := strconv.Atoi(options.Service.InstanceID)
	if err != nil {
		glog.Errorf("Invalid instance from instanceID:%s", options.Service.InstanceID)
		return c, fmt.Errorf("Invalid instance from instanceID:%s", options.Service.InstanceID)
	}
	service, tenantID, svcPath, err := getService(options.ServicedEndpoint, options.Service.ID, instanceID)
	if err != nil {
		glog.Errorf("%+v", err)
		glog.Errorf("Invalid service from serviceID:%s", options.Service.ID)
		return c, ErrInvalidService
	}
	c.healthChecks = service.HealthChecks
	c.tenantID = tenantID
	options.ServiceNamePath = svcPath

	if service.PIDFile != "" {
		if strings.HasPrefix(service.PIDFile, "exec ") {
			cmd := service.PIDFile[5:len(service.PIDFile)]
			out, err := exec.Command("sh", "-c", cmd).Output()
			if err != nil {
				glog.Errorf("Unable to run command '%s'", cmd)
			} else {
				c.PIDFile = strings.Trim(string(out), "\n ")
			}
		} else {
			c.PIDFile = service.PIDFile
		}
	}

	runAs := service.RunAs
	if service.RunAs == "root" {
		runAs = ""
	}
	if runAs != "" && c.options.Service.Command[0] == service.Startup {
		c.options.Service.RunAs = runAs
	}

	// create config files
	if err := setupConfigFiles(service); err != nil {
		glog.Errorf("Could not setup config files error:%s", err)
		return c, fmt.Errorf("container: invalid ConfigFiles error:%s", err)
	}

	// get host id
	c.hostID, err = getAgentHostID(options.ServicedEndpoint)
	if err != nil {
		glog.Errorf("Invalid hostID")
		return c, ErrInvalidHostID
	}

	if options.Logforwarder.Enabled && len(service.LogConfigs) > 0 {
		if err := setupLogstashFiles(c.hostID, options.HostIPs, options.ServiceNamePath, service,
			options.Service.InstanceID, options.Logforwarder); err != nil {
			glog.Errorf("Could not setup logstash files error:%s", err)
			return c, fmt.Errorf("container: invalid LogStashFiles error:%s", err)
		}

		logforwarder, exited, err := subprocess.New(
			time.Second,
			nil,
			"", // runas
			options.Logforwarder.Path, // command
			"-e", // Log to stderr
			"-c", options.Logforwarder.ConfigFile)
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
	} else if !options.MetricForwarding {
		glog.V(1).Infof("Not forwarding metrics for this container (%v)", c.tenantID)
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

		// setup network stats
		destination := fmt.Sprintf("http://localhost%s/api/metrics/store", options.Metric.Address)
		glog.Infof("pushing network stats to: %s", destination)
		go statReporter(destination, time.Second*15)
	}

	// Keep a copy of the service prerequisites in the Controller object.
	c.prereqs = service.Prereqs

	// set up the zookeeper client
	c.zkInfo, err = getAgentZkInfo(options.ServicedEndpoint)
	if err != nil {
		glog.Errorf("Invalid zk info: %v", err)
		return c, err
	}
	glog.Infof(" c.zkInfo: %+v", c.zkInfo)

	// endpoints are created at the root level (not pool aware)
	rootBasePath := ""
	zClient, err := coordclient.New("zookeeper", c.zkInfo.ZkDSN, rootBasePath, nil)
	if err != nil {
		glog.Errorf("failed create a new coordclient: %v", err)
		return c, err
	}

	zzk.InitializeLocalClient(zClient)

	// get endpoints
	opts := ContainerEndpointsOptions{
		HostID:               c.hostID,
		TenantID:             c.tenantID,
		InstanceID:           instanceID,
		IsShell:              os.Getenv("SERVICED_IS_SERVICE_SHELL") == "true",
		TCPMuxPort:           uint16(options.Mux.Port),
		UseTLS:               !options.Mux.DisableTLS,
		VirtualAddressSubnet: options.VirtualAddressSubnet,
	}
	c.endpoints, err = NewContainerEndpoints(service, opts)
	if err != nil {
		return c, err
	}

	// CC Rest API proxy
	c.ccApiProxy = newServicedApiProxy()

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

func (c *Controller) forwardSignal(sig os.Signal) {
	pidBuffer, err := ioutil.ReadFile(c.PIDFile)
	if err != nil {
		glog.Errorf("Error reading PID file while forwarding signal: %v", err)
		return
	}
	pid, err := strconv.Atoi(strings.Trim(string(pidBuffer), "\n "))
	if err != nil {
		glog.Errorf("Error reading PID file while forwarding signal: %v", err)
		return
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		glog.Errorf("Error finding process while forwarding signal: %v", err)
		return
	}
	glog.Infof("Sending signal %v to pid %d provided by PIDFile.", sig, pid)
	err = process.Signal(sig)
	if err != nil {
		glog.Errorf("Encountered error sending signal %v to pid %d: %v", sig, pid, err)
	}
}

// rpcHealthCheck returns a channel that will close when it not longer possible
// to ping the RPC server
func (c *Controller) rpcHealthCheck() (chan struct{}, error) {
	gone := make(chan struct{})

	client, err := node.NewLBClient(c.options.ServicedEndpoint)
	if err != nil {
		return nil, err
	}
	go func() {
		var ts time.Time
		retries := 3
		failures := 0
		for {
			err := client.Ping(2*time.Second, &ts)
			if err != nil {
				failures++
				glog.Warningf("RPC Server healthcheck ping to delegate failed. Error: %v", err)
				if failures == retries {
					glog.Error("RPC Server healthcheck retries exhausted, marking it dead and closing channel.")
					close(gone)
					return
				}
			} else {
				failures = 0
			}
			select {
			case <-time.After(10 * time.Second):
				continue

			}
		}
	}()
	return gone, nil
}

// storageHealthCheck returns a channel that will close when the distributed
// storage is no longer accessible (i.e., stale NFS mount)
func (c *Controller) storageHealthCheck() (chan struct{}, error) {
	gone := make(chan struct{})
	mounts, err := mount.GetMounts()
	if err != nil {
		return nil, err
	}
	nfsMountPoints := []string{}
	for _, minfo := range mounts {
		if strings.HasPrefix(minfo.Fstype, "nfs") {
			nfsMountPoints = append(nfsMountPoints, minfo.Mountpoint)
		}
	}
	if len(nfsMountPoints) > 0 {
		// Start polling
		go func() {
			for {
				for _, mp := range nfsMountPoints {
					if utils.IsNFSMountStale(mp) {
						close(gone)
						return
					}
				}
				<-time.After(5 * time.Second)
			}
		}()
	}
	return gone, nil
}

func (c *Controller) shutdown() {
	glog.V(1).Infof("controller for %v shutting down\n", c.dockerID)
	//defers run in LIFO order
	defer os.Exit(c.exitStatus)
	defer zzk.ShutdownConnections()
}

func (c *Controller) reapZombies(close chan struct{}) {
	for {
		select {
		case <-close:
			return
		case <-time.After(time.Second * 10):
			glog.V(5).Info("reaping zombies")
			proc.ReapZombies()
		}
	}
}

// Run executes the controller's main loop and block until the service exits
// according to it's restart policy or Close() is called.
func (c *Controller) Run() (err error) {
	defer c.shutdown()
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	instID, err := strconv.Atoi(c.options.Service.InstanceID)
	if err != nil {
		return err
	}

	env := os.Environ()
	env = append(env, "CONTROLPLANE=1")
	env = append(env, fmt.Sprintf("CONTROLPLANE_CONSUMER_URL=http://localhost%s/api/metrics/store", c.options.Metric.Address))
	env = append(env, fmt.Sprintf("CONTROLPLANE_HOST_ID=%s", c.hostID))
	env = append(env, fmt.Sprintf("CONTROLPLANE_TENANT_ID=%s", c.tenantID))
	env = append(env, fmt.Sprintf("CONTROLPLANE_INSTANCE_ID=%s", c.options.Service.InstanceID))
	env = append(env, fmt.Sprintf("CONTROLPLANE_SERVICED_ID=%s", c.options.Service.ID))

	if err := writeEnvFile(env); err != nil {
		return err
	}

	args := []string{}
	var command string
	if c.options.Service.RunAs == "" {
		command = "/bin/sh"
	} else {
		args = append(args, "-l")
		command = "/bin/bash"
	}
	args = append(args, "-c")
	args = append(args, "exec "+strings.Join(c.options.Service.Command, " "))

	startService := func() (*subprocess.Instance, chan error) {
		service, serviceExited, _ := subprocess.New(
			time.Second*10, env, c.options.Service.RunAs, command, args...,
		)
		return service, serviceExited
	}

	sendSignal := func(service *subprocess.Instance, sig os.Signal) bool {
		switch {
		case c.PIDFile != "":
			c.forwardSignal(sig)
		case service != nil:
			service.Notify(sig)
		default:
			return false
		}
		return true
	}

	rpcDead, err := c.rpcHealthCheck()
	if err != nil {
		glog.Errorf("Could not setup RPC ping check: %s", err)
		return err
	}

	storageDead, err := c.storageHealthCheck()
	if err != nil {
		glog.Errorf("Could not set up storage check: %s", err)
		return err
	}

	prereqsPassed := make(chan bool)
	var startAfter <-chan time.Time
	var exitAfter <-chan time.Time
	var service *subprocess.Instance = nil
	serviceExited := make(chan error, 1)
	endpointExit := make(chan struct{})
	c.endpoints.Run(endpointExit)

	// Start CC Rest API Proxy
	go c.ccApiProxy.run()

	// HACK: I guess this is how it used to work?  This code is horrible.
	go func() {
		errc := <-c.closing
		close(endpointExit)
		errc <- nil
	}()

	if err := c.handleControlCenterImports(rpcDead); err != nil {
		glog.Error("Could not setup Control Center specific imports: ", err)
		return err
	}
	go c.checkPrereqs(prereqsPassed, rpcDead)
	go c.reapZombies(rpcDead)
	healthExit := make(chan struct{})
	defer close(healthExit)

	if os.Getenv("SERVICED_IS_SERVICE_SHELL") != "true" {
		c.kickOffHealthChecks(healthExit)
	}

	exited := false

	var shutdownService = func(service *subprocess.Instance, sig os.Signal) {
		c.options.Service.Autorestart = false
		if sendSignal(service, sig) {
			// nil out all other channels because we're shutting down
			sigc = nil
			prereqsPassed = nil
			startAfter = nil
			rpcDead = nil
			storageDead = nil

			// exitAfter is the deadman switch for unresponsive processes and any processes that have already exited
			exitAfter = time.After(time.Second * 30)

			glog.Infof("Closing healthExit on signal %v for %q", sig, c.options.Service.ID)
			close(healthExit)
			glog.Infof("Closed healthExit on signal %v for %q", sig, c.options.Service.ID)
		} else {
			c.exitStatus = 1
			exited = true
			glog.Infof("Exited due to sendSignal(%v) failed for %q", sig, c.options.Service.ID)
		}
	}

	for !exited {
		select {
		case sig := <-sigc:
			glog.Infof("Notifying subprocess of signal %v for service %s", sig, c.options.Service.ID)
			shutdownService(service, sig)
			glog.Infof("Notification complete for signal %v for service %s", sig, c.options.Service.ID)

		case <-exitAfter:
			glog.Infof("Killing unresponsive subprocess for service %s", c.options.Service.ID)
			sendSignal(service, syscall.SIGKILL)
			glog.Infof("Kill signal sent for service %s", c.options.Service.ID)
			c.exitStatus = 1
			exited = true

		case <-prereqsPassed:
			startAfter = time.After(time.Millisecond * 1)
			prereqsPassed = nil

		case exitError := <-serviceExited:
			if !c.options.Service.Autorestart {
				exitStatus, _ := utils.GetExitStatus(exitError)
				if c.options.Logforwarder.Enabled {
					time.Sleep(c.options.Logforwarder.SettleTime)
				}
				glog.Infof("Service %s Exited with status:%d due to %+v", c.options.Service.ID, exitStatus, exitError)
				//set loop to end
				exited = true
				//exit with exit code, defer so that other cleanup can happen
				c.exitStatus = exitStatus

			} else {
				glog.Infof("Restarting service process for service %s in 10 seconds.", c.options.Service.ID)
				service = nil
				startAfter = time.After(time.Second * 10)
			}

		case <-startAfter:
			glog.Infof("Starting service process for service %s", c.options.Service.ID)
			service, serviceExited = startService()
			startAfter = nil
		case <-rpcDead:
			glog.Infof("RPC Server has gone away, cleaning up service %s", c.options.Service.ID)
			shutdownService(service, syscall.SIGTERM)
			glog.Infof("RPC Server shutdown for service %s complete", c.options.Service.ID)
		case <-storageDead:
			glog.Infof("Distributed storage for service %s has gone away; shutting down", c.options.Service.ID)
			shutdownService(service, syscall.SIGTERM)
			glog.Infof("Distributed storage shutdown for service %s complete", c.options.Service.ID)
		}
	}
	// Signal to health check registry that this instance is giving up the ghost.
	client, err := node.NewLBClient(c.options.ServicedEndpoint)
	if err != nil {
		glog.Errorf("Could not create a client to endpoint: %s, %s", c.options.ServicedEndpoint, err)
		return nil
	}
	defer client.Close()
	c.Close()
	req := master.ServiceInstanceRequest{
		ServiceID:  c.options.Service.ID,
		InstanceID: instID,
	}
	client.ReportInstanceDead(req, nil)
	return nil
}

func (c *Controller) checkPrereqs(prereqsPassed chan bool, rpcDead chan struct{}) error {
	if len(c.prereqs) == 0 {
		glog.Infof("No prereqs to pass.")
		prereqsPassed <- true
		return nil
	}
	healthCheckInterval := time.Tick(1 * time.Second)
	for {
		select {
		case <-rpcDead:
			glog.Fatalf("Exiting, RPC server has gone away")
		case <-healthCheckInterval:
			failedAny := false
			for _, script := range c.prereqs {
				glog.Infof("Running prereq command: %s", script.Script)
				out, err := exec.Command("sh", "-c", script.Script).CombinedOutput()
				if err != nil {
					msg := fmt.Sprintf("Service %s not starting. Output: %s; error: %s", script.Name, out, err)
					glog.Warning(msg)
					fmt.Fprintln(os.Stderr, msg)
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
	}
}

func (c *Controller) kickOffHealthChecks(healthExit chan struct{}) {
	client, err := node.NewLBClient(c.options.ServicedEndpoint)
	if err != nil {
		glog.Errorf("Could not create a client to endpoint: %s, %s", c.options.ServicedEndpoint, err)
		return
	}
	defer client.Close()

	instanceID, err := strconv.Atoi(c.options.Service.InstanceID)
	if err != nil {
		glog.Errorf("Invalid instance from instanceID:%s", c.options.Service.InstanceID)
		return
	}
	for name, hc := range c.healthChecks {
		glog.Infof("Kicking off health check %s.", name)
		glog.Infof("Setting up health check: %s", hc.Script)
		key := health.HealthStatusKey{
			ServiceID:       c.options.Service.ID,
			InstanceID:      instanceID,
			HealthCheckName: name,
		}
		go c.doHealthCheck(healthExit, key, hc)
	}
	return
}

func (c *Controller) doHealthCheck(cancel <-chan struct{}, key health.HealthStatusKey, hc health.HealthCheck) {
	logger := plog.WithFields(log.Fields{
		"service":     key.ServiceID,
		"instance":    key.InstanceID,
		"healthcheck": key.HealthCheckName,
	})
	hc.Ping(cancel, key, func(stat health.HealthStatus) {
		req := master.HealthStatusRequest{
			Key:     key,
			Value:   stat,
			Expires: hc.Expires(),
		}
		client, err := node.NewLBClient(c.options.ServicedEndpoint)
		if err != nil {
			glog.Errorf("Could not create a client to endpoint: %s, %s", c.options.ServicedEndpoint, err)
			return
		}
		defer client.Close()
		client.ReportHealthStatus(req, nil)
		if stat.KillFlag {
			logger.WithField("kill_count_limit", hc.KillCountLimit).Infof("KillFlag has been set. Shutting down the controller.")
			c.shutdown()
		}
	})
}

func (c *Controller) handleControlCenterImports(rpcdead chan struct{}) error {
	// this function is currently needed to handle special control center imports
	// from GetISvcEndpoints() that do not exist in endpoints from getServiceState
	// get service endpoints
	client, err := node.NewLBClient(c.options.ServicedEndpoint)
	if err != nil {
		glog.Errorf("Could not create a client to endpoint: %s, %s", c.options.ServicedEndpoint, err)
		return err
	}
	defer client.Close()

	epchan := make(chan map[string][]applicationendpoint.ApplicationEndpoint)
	timeout := make(chan struct{})

	go func(c *node.LBClient, svcid string, epc chan map[string][]applicationendpoint.ApplicationEndpoint, timeout chan struct{}) {
		var endpoints map[string][]applicationendpoint.ApplicationEndpoint
	RetryGetISvcEndpoints:
		for {
			err = c.GetISvcEndpoints(svcid, &endpoints)
			if err != nil {
				select {
				case <-time.After(1 * time.Second):
					glog.V(3).Info("Couldn't retrieve service endpoints, trying again")
					continue RetryGetISvcEndpoints
				case <-timeout:
					glog.V(3).Info("Timed out trying to retrieve service endpoints")
					return
				}
			}
			break
		}

		// deal with the race between the one minute timeout in handleControlCenterImports() and the
		// call to GetISvcEndpoints() - the timeout may happen between GetISvcEndpoints() completing
		// and sending the result via the epc channel.
		select {
		case _, ok := <-epc:
			if ok {
				panic("should never receive anything on the endpoints channel")
			}
			glog.V(3).Info("Endpoint channel closed, giving up")
			return
		default:
			epc <- endpoints
		}
	}(client, c.options.Service.ID, epchan, timeout)

	var endpoints map[string][]applicationendpoint.ApplicationEndpoint
	select {
	case <-time.After(1 * time.Minute):
		close(epchan)
		timeout <- struct{}{}
		client.SendLogMessage(node.ServiceLogInfo{ServiceID: c.options.Service.ID, Message: "unable to retrieve service endpoints"}, nil)
		return ErrNoServiceEndpoints
	case <-rpcdead:
		close(epchan)
		timeout <- struct{}{}
		return fmt.Errorf("RPC Service has gone away")
	case endpoints = <-epchan:
		glog.Infof("Got service endpoints for %s: %+v", c.options.Service.ID, endpoints)
	}

	for _, eps := range endpoints {
		bind := zkservice.ImportBinding{
			Application:    eps[0].Application,
			Purpose:        "import", // Punting on control center dynamic imports for now
			PortNumber:     eps[0].ProxyPort,
			VirtualAddress: eps[0].VirtualAddress,
		}
		exports := make([]registry.ExportDetails, len(eps))
		for i, ep := range eps {
			exports[i] = registry.ExportDetails{
				ExportBinding: zkservice.ExportBinding{
					Application: ep.Application,
					Protocol:    ep.Protocol,
					PortNumber:  ep.ContainerPort,
				},
				PrivateIP:  ep.ContainerIP,
				HostIP:     ep.HostIP,
				InstanceID: i,
			}
		}
		c.endpoints.UpdateRemoteExports(bind, exports)
	}
	// TODO: agent needs to register controlplane and controlplane_consumer
	//       but don't do that here in the container code

	return nil
}
