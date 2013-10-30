package main

import (
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/dao"
	sproxy "github.com/zenoss/serviced/proxy"

	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"sort"
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
	config := sproxy.Config{}
	config.TCPMux.Port = proxyOptions.muxport
	config.TCPMux.Enabled = proxyOptions.mux
	config.TCPMux.UseTLS = proxyOptions.tls
	config.ServiceId = proxyCmd.Arg(0)
	config.Command = proxyCmd.Arg(1)

	if config.TCPMux.Enabled {
		go config.TCPMux.ListenAndMux()
	}

	// continually execute subprocess
	go func(cmdString string) {
		for {
			glog.Infof("About to execute: %s", cmdString)
			cmd := exec.Command("bash", "-c", cmdString)
			stderr, err := cmd.StderrPipe()
			if err != nil {
				glog.Fatalf("Problem opening a stderr pipe to service: %s", err)
			}
			go io.Copy(os.Stderr, stderr)
			stdout, err := cmd.StdoutPipe()
			if err != nil {
				glog.Fatalf("Problem opening a stdout pipe to service: %s", err)
			}
			go io.Copy(os.Stdout, stdout)
			glog.Infof("About to execute: %s", cmdString)
			err = cmd.Run()
			if err != nil {
				glog.Errorf("Problem running service: %v", err)
				glog.Flush()
			}
			glog.Infof("service exited, sleeping...")
			time.Sleep(time.Minute)
		}
	}(config.Command)

	for {
	func() {
		client, err := sproxy.NewLBClient(proxyOptions.servicedEndpoint)
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

			var proxy *sproxy.Proxy
			var ok bool
			if proxy, ok = proxies[key]; !ok {
				// setup a new proxy
				listener, err := net.Listen("tcp4", fmt.Sprintf(":%d", endpointList[0].ContainerPort))
				if err != nil {
					glog.Errorf("Could not bind to port: %s", err)
					continue
				}
				proxy, err = sproxy.NewProxy(
					fmt.Sprintf("%v", endpointList[0]),
					uint16(config.TCPMux.Port),
					config.TCPMux.UseTLS,
					listener)
				if err != nil {
					glog.Errorf("Could not build proxy %s", err)
					continue
				}
				proxies[key] = proxy
			}
			proxy.SetNewAddresses(addresses)
		}
	}()

		time.Sleep(time.Second*10)
	}

	glog.Flush()
	os.Exit(0)
	return nil
}



var proxies map[string] *sproxy.Proxy

func init() {
	proxies = make(map[string] *sproxy.Proxy)
}
