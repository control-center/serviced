package container

import (
	//"github.com/zenoss/glog"
	"github.com/zenoss/serviced/commons/subprocess"

	//"net"
	"time"
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
		ConfigFile string //
	}
	Metric struct {
		Address       string // TCP port to host the metric service, :22350
		RemoteEndoint string // The url to forward metric queries
	}
}

// Controller is a object to manage the operations withing a container. For example,
// it creates the managed service instance, logstash forwarding, port forwarding, etc.
type Controller struct {
	options         ControllerOptions
	service         *subprocess.Instance
	metricForwarder *MetricForwarder
	logforwarder    *subprocess.Instance
}

type Closer interface {
	Close() error
}

func (c *Controller) Close() error {
	for _, s := range []Closer{c.service, c.metricForwarder, c.logforwarder} {
		if s != nil {
			s.Close()
		}
	}
	c.service = nil
	c.metricForwarder = nil
	c.logforwarder = nil
	return nil
}

// NewController
func NewController(options ControllerOptions) (*Controller, error) {
	c := &Controller{}

	if options.Logforwarder.Enabled {
		// make sure we pick up any logfile that was modified within the
		// last three years
		// TODO: Either expose the 3 years a configurable or get rid of it
		logforwarder, err := subprocess.New(time.Millisecond, time.Second,
			options.Logforwarder.Path,
			"-old-files-hours=26280",
			"-config", options.Logforwarder.ConfigFile)
		if err != nil {
			return nil, err
		}
		c.logforwarder = logforwarder
	}

	//build metric redirect url -- assumes 8444 is port mapped
	metric_redirect := options.Metric.RemoteEndoint
	metric_redirect += "?controlplane_tenant_id=" + options.Service.TenantID
	metric_redirect += "&controlplane_service_id=" + options.Service.ID

	//build and serve the container metric forwarder
	forwarder, err := NewMetricForwarder(options.Metric.Address, metric_redirect)
	if err != nil {
		return c, err
	}
	c.metricForwarder = forwarder

	return c, nil
}

/*
func (c *Controller) Run() {

	serviceId := proxyCmd.Arg(0)
	command := proxyCmd.Arg(1)

	procexit := make(chan int)
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	//monitor application endpoints to mux ports
	go func() {
		for {
			func() {
				client, err := serviced.NewLBClient(proxyOptions.ServicedEndpoint)
				if err != nil {
					glog.Errorf("Could not create a client to endpoint %s: %s", proxyOptions.ServicedEndpoint, err)
					return
				}
				defer client.Close()

				var endpoints map[string][]*dao.ApplicationEndpoint
				err = client.GetServiceEndpoints(config.ServiceId, &endpoints)
				if err != nil {
					glog.Errorf("Error getting application endpoints for service %s: %s", config.ServiceId, err)
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
							uint16(config.TCPMux.Port),
							config.TCPMux.UseTLS,
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
			}()

			time.Sleep(time.Second * 10)
		}
	}()

	//setup container metric forwarder
	go func() {
		//loop until successfully identifying this container's tenant id
		var tenantID string
		for {
			client, err := serviced.NewLBClient(proxyOptions.ServicedEndpoint)
			if err == nil {
				defer client.Close()
				if err = client.GetTenantId(config.ServiceId, &tenantID); err != nil {
					glog.Errorf("Failed to get tenant id: %s", err)
				} else {
					//success
					break
				}
			} else {
				glog.Errorf("Failed to create a client to endpoint %s: %s", proxyOptions.ServicedEndpoint, err)
			}
		}

		//build metric redirect url -- assumes 8444 is port mapped
		metricRedirect := "http://localhost:8444/api/metrics/store"
		metricRedirect += "?controlplane_tenant_id=" + tenantID
		metricRedirect += "&controlplane_service_id=" + config.ServiceId

		//build and serve the container metric forwarder
		forwarder, _ := serviced.NewMetricForwarder(":22350", metricRedirect)
		forwarder.Serve()
	}()

	exitcode := <-procexit // Wait for proc goroutine to exit

	glog.Flush()
	os.Exit(exitcode)
	return nil
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
*/
