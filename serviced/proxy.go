package main

import (
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/dao"
	sproxy "github.com/zenoss/serviced/proxy"

	"bytes"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"sort"
	"time"
)

func handler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	serviced := "./serviced/serviced"

	var op string
	if opData := r.Form["op"]; len(opData) == 1 {
		op = opData[0]
	} else {
		fmt.Fprintf(w, "Missing required param 'op'")
		return
	}

	var serviceId string
	if serviceIdData := r.Form["serviceId"]; len(serviceIdData) == 1 {
		serviceId = serviceIdData[0]
	} else {
		fmt.Fprintf(w, "Missing required param 'serviceId'")
		return
	}

	var imageId string
	if imageIdData := r.Form["imageId"]; len(imageIdData) == 1 {
		imageId = imageIdData[0]
	} else {
		imageId = ""
	}

	files := r.Form["files"]

	var args string
	switch op {
	case "show":
		args = fmt.Sprintf("%s", serviceId)
		break
	case "rollback":
	case "commit":
		if imageId == "" {
			fmt.Fprintf(w, "Missing required param 'imageId'")
			return
		}
		args = fmt.Sprintf("%s %s", serviceId, imageId)
		break
	case "get":
		if len(files) == 0 {
			fmt.Fprintf(w, "Missing required param 'files'")
			return
		}
		args = fmt.Sprintf("-snapshot=%s %s %s", imageId, serviceId, files[0])
		break
	case "recv":
		if len(files) == 0 {
			fmt.Fprintf(w, "Missing required param 'files'")
			return
		}

		fileArgs := files[0]
		for i := 1; i < len(files); i++ {
			fileArgs = fileArgs + " " + files[i]
		}
		args = fmt.Sprintf("-snapshot=%s %s %s", imageId, serviceId, fileArgs)
		break
	default:
		fmt.Fprintf(w, "Unknown operation: %s", op)
		return
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	fmt.Fprintf(w, ">> serviced %s %s\n", op, args)
	cmd := exec.Command("bash", "-c", serviced+" "+op+" "+args)
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(w, "Received %s\n", err)
		fmt.Fprintf(w, "%s", stderr.Bytes())
		return
	}
	fmt.Fprintf(w, "%s", stdout.Bytes())
}

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

	http.HandleFunc("/exec", handler)
	http.ListenAndServe(":50000", nil)

	procexit := make(chan int)

	// continually execute subprocess
	go func(cmdString string) {
		defer func() { procexit <- 1 }()
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
			}
			if !proxyOptions.autorestart {
				break
			}
			glog.V(0).Info("service exited, sleeping...")
			time.Sleep(time.Minute)
		}
	}(config.Command)

	go func() {
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

			time.Sleep(time.Second * 10)
		}
	}()

	<-procexit // Wait for proc goroutine to exit

	glog.Flush()
	os.Exit(0)
	return nil
}

var proxies map[string]*sproxy.Proxy

func init() {
	proxies = make(map[string]*sproxy.Proxy)
}
