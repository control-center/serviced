package api

import (
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced"
	"github.com/zenoss/serviced/dao"
	"github.com/zenoss/serviced/shell"

	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var proxyOptions ProxyOptions

// ProxyOptions are options to be run when starting a new proxy server
type ProxyOptions struct {
	MuxPort          int
	Mux              bool
	TLS              bool
	KeyPEMFile       string
	CertPEMFile      string
	ServicedEndpoint string
	Autorestart      bool
	Logstash         bool
}

// LoadProxyOptions loads the proxy option information
func LoadProxyOptions(ops ProxyOptions) {
	proxyOptions = ops
}

// ProxyConfig is the config object for starting a proxy server
type ProxyConfig struct {
	ServiceID string
	Command   []string
}

func getNextIP() string {
	s := fmt.Sprintf("10.3.0.%d", nextip)
	nextip += 1
	return s
}

// Start a service proxy
func (a *api) StartProxy(cfg ProxyConfig) error {
	config := serviced.MuxConfig{}
	config.TCPMux.Port = proxyOptions.MuxPort
	config.TCPMux.Enabled = proxyOptions.Mux
	config.TCPMux.UseTLS = proxyOptions.TLS
	config.ServiceId = cfg.ServiceID
	config.Command = strings.Join(cfg.Command, " ")

	if config.TCPMux.Enabled {
		go config.TCPMux.ListenAndMux()
	}

	sio := shell.NewProcessForwarderServer(proxyOptions.ServicedEndpoint)
	sio.Handle("/", http.FileServer(http.Dir("/serviced/www/")))
	go http.ListenAndServe(":50000", sio)

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
					client, err := serviced.NewLBClient(proxyOptions.ServicedEndpoint)
					message := fmt.Sprintf("Service returned a non-zero exit code: %v. Command: \"%v\" Message: %v", config.ServiceId, config.Command, err)
					if err == nil {
						defer client.Close()
						glog.Errorf(message)

						// send the log message to the master
						client.SendLogMessage(serviced.ServiceLogInfo{config.ServiceId, message}, nil)
					} else {
						glog.Errorf("Failed to create a client to endpoint %s: %s", proxyOptions.ServicedEndpoint, err)
					}

					glog.Infof("%s", err)
					glog.Flush()
					if exiterr, ok := cmderr.(*exec.ExitError); ok && !proxyOptions.Autorestart {
						if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
							procexit <- status.ExitStatus()
						}
					}
				}
			}

			if !proxyOptions.Autorestart {
				break
			}
			glog.V(0).Info("service exited, sleeping...")
			time.Sleep(time.Minute)
		}
	}(config.Command)

	if proxyOptions.Logstash {
		go func() {
			// make sure we pick up any logfile that was modified within the
			// last three years
			// TODO: Either expose the 3 years a configurable or get rid of it
			logstashCmd := serviced.LOGSTASH_CONTAINER_DIRECTORY + "/logstash-forwarder"
			args := []string{"-old-files-hours=26280", "-config", serviced.LOGSTASH_CONTAINER_CONFIG}
			glog.V(0).Info("About to execute: %s %s", logstashCmd, args)

			myCmd := exec.Command(logstashCmd, args...)
			myCmd.Stdout = os.Stdout
			myCmd.Stderr = os.Stderr
			myErr := myCmd.Run()
			if myErr != nil {
				glog.Errorf("Problem running service: %v", myErr)
				glog.Flush()
			}
		}()
	}

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
