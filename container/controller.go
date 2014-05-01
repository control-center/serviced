package container

import (
	//"github.com/zenoss/glog"
	"github.com/zenoss/serviced/commons/subprocess"

	//"net"
	"time"
)

// ControllerOptions are options to be run when starting a new proxy server
type ControllerOptions struct {
	TentantID        string   // The top level service id
	ServiceID        string   // The uuid of the service to launch
	Command          []string // The command to launch
	MuxPort          int      // the TCP port for the remote mux
	Mux              bool     // True if a remote mux is used
	TLS              bool     // True if TLS should be used on the mux
	KeyPEMFile       string   // path to the KeyPEMfile
	CertPEMFile      string   // path to the CertPEMfile
	ServicedEndpoint string
	Autorestart      bool
	Logstash         bool
}

// Controller is a object to manage the operations withing a container. For example,
// it creates the managed service instance, logstash forwarding, port forwarding, etc.
type Controller struct {
	options         ControllerOptions
	service         *subprocess.Instance
	metricForwarder *MetricForwarder
	logforwarder    *subprocess.Instance
}

func (c *Controller) Close() {
	return
}

// NewController
func NewController(options ControllerOptions) (*Controller, error) {
	c := &Controller{}

	if options.Logstash {
		// make sure we pick up any logfile that was modified within the
		// last three years
		// TODO: Either expose the 3 years a configurable or get rid of it
		logforwarder, err := subprocess.New(time.Millisecond, time.Second,
			"/usr/local/serviced/resources/logstash/logstash-forwarder",
			"-old-files-hours=26280",
			"-config", "/etc/logstash-forwarder.conf")
		if err != nil {
			return nil, err
		}
		c.logforwarder = logforwarder
	}

	//build metric redirect url -- assumes 8444 is port mapped
	metric_redirect := "http://localhost:8444/api/metrics/store"
	metric_redirect += "?controlplane_tenant_id=" + options.TentantID
	metric_redirect += "&controlplane_service_id=" + options.ServiceID

	//build and serve the container metric forwarder
	forwarder, err := NewMetricForwarder(":22350", metric_redirect)
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

	// continually execute subprocess
	go func(cmdString string) {
		defer func() { procexit <- 0 }()
		for {
			glog.V(0).Info("About to execute: ", cmdString)
			cmd := exec.Command("bash", "-c", cmdString)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmd.Stdin = os.Stdin
			serviceExit := make(chan error)
			go func() {
				serviceExit <- cmd.Run()
			}()
			select {
			case sig := <-sigc:
				glog.V(1).Infof("Caught signal %d", sig)
				cmd.Process.Signal(sig)
				cmd.Wait()
				exitCode := 0
				if cmd.ProcessState != nil {
					exitCode, _ = cmd.ProcessState.Sys().(int)
				}
				procexit <- exitCode
			case cmderr := <-serviceExit:
				if cmderr != nil {
					client, err := serviced.NewLBClient(proxyOptions.servicedEndpoint)
					message := fmt.Sprintf("Service returned a non-zero exit code: %v. Command: \"%v\" Message: %v", serviceId, command, err)
					if err == nil {
						defer client.Close()
						glog.Errorf(message)

						// send the log message to the master
						client.SendLogMessage(serviced.ServiceLogInfo{serviceId, message}, nil)
					} else {
						glog.Errorf("Failed to create a client to endpoint %s: %s", proxyOptions.servicedEndpoint, err)
					}

					glog.Infof("%s", err)
					glog.Flush()
					if exiterr, ok := cmderr.(*exec.ExitError); ok && !proxyOptions.autorestart {
						if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
							procexit <- status.ExitStatus()
						}
					}
				}
			}

			if !proxyOptions.autorestart {
				break
			}
			glog.V(0).Info("service exited, sleeping...")
			time.Sleep(time.Minute)
		}
	}(command)


	//monitor application endpoints to mux ports
	go func() {
		for {
			func() {
				client, err := serviced.NewLBClient(proxyOptions.servicedEndpoint)
				if err != nil {
					glog.Errorf("Could not create a client to endpoint %s: %s", proxyOptions.servicedEndpoint, err)
					return
				}
				defer client.Close()

				var endpoints map[string][]*dao.ApplicationEndpoint
				err = client.GetServiceEndpoints(serviceId, &endpoints)
				if err != nil {
					glog.Errorf("Error getting application endpoints for service %s: %s", serviceId, err)
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

					var proxy *serviced.Proxy
					var ok bool
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
							uint16(proxyOptions.muxport),
							proxyOptions.tls,
							listener)
						if err != nil {
							glog.Errorf("Could not build proxy %s", err)
							continue
						}

						glog.Infof("Success binding port: %s -> %+v", key, proxy)
						proxies[key] = proxy
					}
					proxy.SetNewAddresses(addresses)
				}
			}()

			time.Sleep(time.Second * 10)
		}
	}()

	//setup container metric forwarder

	exitcode := <-procexit // Wait for proc goroutine to exit

	glog.Flush()
	os.Exit(exitcode)
	return nil
}

var proxies map[string]*serviced.Proxy

func init() {
	proxies = make(map[string]*serviced.Proxy)
}
*/
