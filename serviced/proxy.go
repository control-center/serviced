package main

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
	"sort"
	"syscall"
	"time"
)

// Start a service proxy.
func (cli *ServicedCli) CmdProxy(args ...string) error {

	if err := proxyCmd.Parse(args); err != nil {
		return err
	}
	if len(proxyCmd.Args()) != 2 {
		proxyCmd.Usage()
		glog.Flush()
		os.Exit(2)
	}
	config := serviced.MuxConfig{}
	config.TCPMux.Port = proxyOptions.muxport
	config.TCPMux.Enabled = proxyOptions.mux
	config.TCPMux.UseTLS = proxyOptions.tls
	config.ServiceId = proxyCmd.Arg(0)
	config.Command = proxyCmd.Arg(1)

	if config.TCPMux.Enabled {
		go config.TCPMux.ListenAndMux()
	}

	sio := shell.NewProcessForwarderServer(proxyOptions.servicedEndpoint)
	sio.Handle("/", http.FileServer(http.Dir("/serviced/www/")))
	go http.ListenAndServe(":50000", sio)

	procexit := make(chan int)

	// continually execute subprocess
	go func(cmdString string) {
		defer func() { procexit <- 0 }()
		for {
			glog.V(0).Info("About to execute: ", cmdString)
			cmd := exec.Command("bash", "-c", cmdString)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmd.Stdin = os.Stdin
			err := cmd.Run()
			if err != nil {
				glog.Errorf("Problem running service: %v", err)
				glog.Flush()
				if exiterr, ok := err.(*exec.ExitError); ok && !proxyOptions.autorestart {
					if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
						procexit <- status.ExitStatus()
					}
				}
			}
			if !proxyOptions.autorestart {
				break
			}
			glog.V(0).Info("service exited, sleeping...")
			time.Sleep(time.Minute)
		}
	}(config.Command)

	if proxyOptions.logstash {
		go func() {
			// *********************************************************************************************
			// ***** FIX ME the following 3 variables are defined in agent.go as well! *********************
			containerLogstashForwarderDir := "/usr/local/serviced/resources/logstash"
			containerLogstashForwarderBinaryPath := containerLogstashForwarderDir + "/logstash-forwarder"
			containerLogstashForwarderConfPath := containerLogstashForwarderDir + "/logstash-forwarder.conf"
			// *********************************************************************************************
			cmdString := containerLogstashForwarderBinaryPath + " -old-files-hours=26280 -config " + containerLogstashForwarderConfPath
			glog.V(0).Info("About to execute: ", cmdString)
			myCmd := exec.Command("bash", "-c", cmdString)
			myCmd.Stdout = os.Stdout
			myCmd.Stderr = os.Stderr
			myErr := myCmd.Run()
			if myErr != nil {
				glog.Errorf("Problem running logstash-forwarder service: %v", myErr)
				glog.Flush()
			}
		}()
	}

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
				err = client.GetServiceEndpoints(config.ServiceId, &endpoints)
				if err != nil {
					glog.Errorf("Error getting application endpoints for service %s: %s", config.ServiceId, err)
					return
				}

				for key, endpointList := range endpoints {
					if len(endpointList) <= 0 {
						glog.Warningf("No endpoints found for %s", key)
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
							uint16(config.TCPMux.Port),
							config.TCPMux.UseTLS,
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

	exitcode := <-procexit // Wait for proc goroutine to exit

	glog.Flush()
	os.Exit(exitcode)
	return nil
}

var proxies map[string]*serviced.Proxy

func init() {
	proxies = make(map[string]*serviced.Proxy)
}
