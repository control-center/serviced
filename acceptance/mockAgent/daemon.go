// Copyright 2015 The Serviced Authors.
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
package main

import (
	"crypto/tls"
	"fmt"
	"net/rpc"
	"net/rpc/jsonrpc"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/proxy"
	"github.com/control-center/serviced/utils"
	"github.com/zenoss/glog"
)

type daemon struct {
	hostConfig *HostConfig
	rpcServer  *rpc.Server
	host       *host.Host
}

func newDaemon(hostConfig *HostConfig, rpcServer *rpc.Server) (*daemon, error) {
	d := &daemon{
		hostConfig: hostConfig,
		rpcServer:  rpcServer,
	}
	return d, nil
}

func (d *daemon) run(address string) (err error) {
	if err := d.hostConfig.setDefaults(address); err != nil {
		return fmt.Errorf("unable to initialize host configuration: %v", err)
	}

	if err := d.buildHost(); err != nil {
		return fmt.Errorf("failed to build Host domain object: %v", err)
	}

	d.startRPC()
	if err := d.registerAgentRPC(); err != nil {
		glog.Fatal(err)
	}

	signalC := make(chan os.Signal, 10)
	signal.Notify(signalC, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	sig := <-signalC
	glog.Infof("Shutting down due to interrupt - %v", sig)
	return nil
}

func (d *daemon) startRPC() {
	cert, err := tls.X509KeyPair([]byte(proxy.InsecureCertPEM), []byte(proxy.InsecureKeyPEM))
	if err != nil {
		glog.Fatalf("Could not parse public/private key pair (tls.X509KeyPair): %v", err)
	}

	tlsConfig := tls.Config{
		Certificates:             []tls.Certificate{cert},
		PreferServerCipherSuites: true, CipherSuites: utils.CipherSuites(),
	}

	listener, err := tls.Listen("tcp", d.hostConfig.Listen, &tlsConfig)
	if err != nil {
		glog.Fatalf("Unable to bind to port %s. Is another instance running?", d.hostConfig.Listen)
	}

	d.rpcServer.HandleHTTP(rpc.DefaultRPCPath, rpc.DefaultDebugPath)

	glog.Infof("Listening on %s", listener.Addr().String())
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				glog.Fatalf("Error accepting connections: %s", err)
			}
			go d.rpcServer.ServeCodec(jsonrpc.NewServerCodec(conn))
		}
	}()
}

func (d *daemon) newMock() *MockAgent {
	return &MockAgent{
		mockHost: d.host,
	}
}

func (d *daemon) registerAgentRPC() error {
	glog.Infof("agent start staticips: %v [%d]", d.hostConfig.StaticIPs, len(d.hostConfig.StaticIPs))
	if err := d.rpcServer.RegisterName("Agent", d.newMock()); err != nil {
		return fmt.Errorf("could not register Agent RPC server: %v", err)
	}
	glog.Infof("finished rpcServer.RegisterName")
	return nil
}

func (d *daemon) buildHost() error {
	fmt.Printf("host configuration = %v\n", d.hostConfig)

	rpcPort := "0"
	parts := strings.Split(d.hostConfig.Listen, ":")
	if len(parts) > 1 {
		rpcPort = parts[1]
	}

	var err error
	glog.Infof("Outbound IP: %s", d.hostConfig.OutboundIP)

	d.host, err = host.Build(d.hostConfig.OutboundIP, rpcPort, d.hostConfig.PoolID, fmt.Sprintf("%d", d.hostConfig.Memory))
	if err != nil {
		return fmt.Errorf("Failed to build host: %v", err)
	}

	d.host.ID = fmt.Sprintf("%d", d.hostConfig.HostID)
	d.host.Name = d.hostConfig.Name
	d.host.IPs = make([]host.HostIPResource, 0)

	if d.hostConfig.Memory != 0 {
		d.host.Memory = d.hostConfig.Memory
	}
	if d.hostConfig.Cores != 0 {
		d.host.Cores = d.hostConfig.Cores
	}
	if d.hostConfig.KernelVersion != "" {
		d.host.KernelVersion = d.hostConfig.KernelVersion
	}
	if d.hostConfig.KernelRelease != "" {
		d.host.KernelRelease = d.hostConfig.KernelRelease
	}
	if d.hostConfig.CCRelease != "" {
		d.host.ServiceD.Release = d.hostConfig.CCRelease
	}

	return nil
}
