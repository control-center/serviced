package container

import (
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced"
	"github.com/zenoss/serviced/commons/subprocess"
	"github.com/zenoss/serviced/dao"
	"github.com/zenoss/serviced/domain"

	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
	"os/exec"
)

var (
	// ErrInvalidCommand is returned if a command is empty or malformed
	ErrInvalidCommand = errors.New("container: invalid command")
	// ErrInvalidEndpoint is returned if an endpoint is empty or malformed
	ErrInvalidEndpoint = errors.New("container: invalid endpoint")
	// ErrInvalidTenantID is returned if a TenantID is empty or malformed
	ErrInvalidTenantID = errors.New("container: invalid tenant id")
	// ErrInvalidServicedID is returned if a ServiceID is empty or malformed
	ErrInvalidServicedID = errors.New("container: invalid serviced id")
)

// ControllerOptions are options to be run when starting a new proxy server
type ControllerOptions struct {
	ServicedEndpoint string
	Service          struct {
		ID          string   // The uuid of the service to launch
		TenantID    string   // The tentant ID of the service
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
	metricForwarder    *MetricForwarder
	logforwarder       *subprocess.Instance
	logforwarderExited chan error
	closing            chan chan error
}

// Close shuts down the controller
func (c *Controller) Close() error {
	errc := make(chan error)
	c.closing <- errc
	return <-errc
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

	if options.Logforwarder.Enabled {
		// make sure we pick up any logfile that was modified within the
		// last three years
		// TODO: Either expose the 3 years a configurable or get rid of it
		logforwarder, exited, err := subprocess.New(time.Second,
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
		if len(options.Service.TenantID) == 0 {
			return nil, ErrInvalidTenantID
		}
		if len(options.Service.ID) > 0 {
			return nil, ErrInvalidServicedID
		}
		metricRedirect += "&controlplane_service_id=" + options.Service.ID
		metricRedirect += "?controlplane_tenant_id=" + options.Service.TenantID
		//build and serve the container metric forwarder
		forwarder, err := NewMetricForwarder(options.Metric.Address, metricRedirect)
		if err != nil {
			return c, err
		}
		c.metricForwarder = forwarder
	}

	glog.Infof("command: %v [%d]", options.Service.Command, len(options.Service.Command))
	if len(options.Service.Command) < 1 {
		glog.Errorf("Invalid commandif ")
		return c, ErrInvalidCommand
	}

	return c, nil
}

// Run executes the controller's main loop and block until the service exits
// according to it's restart policy or Close() is called.
func (c *Controller) Run() (err error) {

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	args := []string{"-c", "exec " + strings.Join(c.options.Service.Command, " ")}

	service, serviceExited, _ := subprocess.New(time.Second*10, "/bin/sh", args...)
	
	healthExits := c.kickOffHealthChecks()

	var restartAfter <-chan time.Time
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

		case <-time.After(time.Second * 10):
			c.handleRemotePorts()

		case <-serviceExited:
			if !c.options.Service.Autorestart {
				return
			}
			restartAfter = time.After(time.Second * 10)

		case <-restartAfter:
			if !c.options.Service.Autorestart {
				return
			}
			glog.Infof("restarting service process")
			service, serviceExited, _ = subprocess.New(time.Second*10, c.options.Service.Command[0], args...)
			restartAfter = nil
		}
	}
	for _, exitChannel := range healthExits {
		exitChannel <- true
	}
	return
}

func (c *Controller) kickOffHealthChecks() map[string]chan bool {
	exitChannels := make(map[string] chan bool)
	client, err := serviced.NewLBClient(c.options.ServicedEndpoint)
	if err != nil {
		glog.Errorf("Could not create a client to endpoint: %s, %s", c.options.ServicedEndpoint, err)
		return nil
	}
	defer client.Close()
	var healthChecks map[string]domain.HealthCheck;
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
	return exitChannels;
}

func (c *Controller) handleHealthCheck(name string, script string, interval time.Duration, exitChannel chan bool) {
	client, err := serviced.NewLBClient(c.options.ServicedEndpoint)
	if err != nil {
		glog.Errorf("Could not create a client to endpoint: %s, %s", c.options.ServicedEndpoint, err)
		return
	}
	defer client.Close()
	for {
		select {
		case <-time.After(interval):
			script_file, err := ioutil.TempFile("", name)
			if err != nil {
				glog.Errorf("Error creating temporary file for health check %s: %s", name, err)
				continue
			}
			err = ioutil.WriteFile(script_file.Name(), []byte(script), os.FileMode(0777))
			if err != nil {
				glog.Errorf("Error writing script for health check %s: %s", name, err)
				script_file.Close()
				continue
			}
			script_file.Close()
			err = os.Chmod(script_file.Name(), os.FileMode(0777))
			if err != nil {
				glog.Errorf("Error setting script executable for health check %s: %s", name, err)
				script_file.Close()
				continue
			}
			cmd := exec.Command("sh", "-c", script_file.Name())
			err = cmd.Run()
			if err == nil {
				glog.Infof("Health check %s succeeded.", name)
				_ = client.LogHealthCheck(domain.HealthCheckResult{c.options.Service.ID, name, true}, nil)
			} else {
				glog.Infof("Health check %s failed.", name)
				_ = client.LogHealthCheck(domain.HealthCheckResult{c.options.Service.ID, name, false}, nil)
			}
		case <- exitChannel:
			return
		}
	}
}

func (c *Controller) handleRemotePorts() {
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

	for key, endpointList := range endpoints {
		if len(endpointList) <= 0 {
			if proxy, ok := proxies[key]; ok {
				emptyAddressList := make([]string, 0)
				proxy.SetNewAddresses(emptyAddressList)
			}
			continue
		}

		addresses := make([]string, len(endpointList))
		for i, endpoint := range endpointList {
			glog.Infof("endpoints: %s, %v", key, *endpoint)
			addresses[i] = fmt.Sprintf("%s:%d", endpoint.HostIp, endpoint.HostPort)
		}
		sort.Strings(addresses)

		var (
			proxy *serviced.Proxy
			ok    bool
		)

		if proxy, ok = proxies[key]; !ok {
			glog.Infof("Attempting port map for: %s -> %+v", key, *endpointList[0])

			// setup a new proxy
			listener, err := net.Listen("tcp4", fmt.Sprintf(":%d", endpointList[0].ContainerPort))
			if err != nil {
				glog.Errorf("Could not bind to port: %s", err)
				continue
			}
			proxy, err = serviced.NewProxy(
				fmt.Sprintf("%v", endpointList[0]),
				uint16(c.options.Mux.Port),
				c.options.Mux.TLS,
				listener)
			if err != nil {
				glog.Errorf("Could not build proxy %s", err)
				continue
			}

			glog.Infof("Success binding port: %s -> %+v", key, proxy)
			proxies[key] = proxy

			if ep := endpointList[0]; ep.VirtualAddress != "" {
				p := strconv.FormatUint(uint64(ep.ContainerPort), 10)
				err := vifs.RegisterVirtualAddress(ep.VirtualAddress, p, ep.Protocol)
				if err != nil {
					glog.Errorf("Error creating virtual address: %+v", err)
				}
			}
		}
		proxy.SetNewAddresses(addresses)
	}

}

var (
	proxies map[string]*serviced.Proxy
	vifs    *VIFRegistry
	nextip  int
)

func init() {
	proxies = make(map[string]*serviced.Proxy)
	vifs = NewVIFRegistry()
	nextip = 1
}
