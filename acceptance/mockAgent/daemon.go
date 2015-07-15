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
	"fmt"
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/rpc/agent"
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

func (d *daemon) run() (err error) {
	if err := d.hostConfig.setDefaults(); err != nil {
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
	listener, err := net.Listen("tcp", d.hostConfig.Listen)
	if err != nil {
		glog.Fatalf("Unable to bind to port %s. Is another instance running?", d.hostConfig.Listen)
	}

	d.rpcServer.HandleHTTP(rpc.DefaultRPCPath, rpc.DefaultDebugPath)

	glog.Infof("Listening on %s", listener.Addr().String())
	go func() {
		for {
			glog.Infof("listening:Accept()")
			conn, err := listener.Accept()
			if err != nil {
				glog.Fatalf("Error accepting connections: %s", err)
			}
			glog.Infof("listening:ServeCodec()")
			go d.rpcServer.ServeCodec(jsonrpc.NewServerCodec(conn))
			// err = d.rpcServer.ServeRequest(jsonrpc.NewServerCodec(conn))
			// if err != nil {
			// 	glog.Fatalf("Error serving request: %s", err)
			// }
		}
	}()
}

type MockServer struct {
	mockHost *host.Host
}

func (d *daemon) newMock() *MockServer {
	return &MockServer{
		mockHost: d.host,
	}
}

func (m *MockServer) BuildHost(request agent.BuildHostRequest, hostResponse *host.Host) error {
	*hostResponse = host.Host{}

	glog.Infof("Build Host Request: %s:%i, %s, %s", request.IP, request.Port, request.PoolID, request.Memory)

	if mem, err := utils.ParseEngineeringNotation(request.Memory); err == nil {
		m.mockHost.RAMCommitment = mem
	} else if mem, err := utils.ParsePercentage(request.Memory, m.mockHost.Memory); err == nil {
		m.mockHost.RAMCommitment = mem
	} else {
		return fmt.Errorf("Could not parse RAM Commitment: %v", err)
	}
	if request.PoolID != m.mockHost.PoolID {
		m.mockHost.PoolID = request.PoolID
	}
	*hostResponse = *m.mockHost
	return nil
	
}

func (d *daemon) registerAgentRPC() error {
	// FIXME: We need to register a custom server to implement our own BuildHost() RPC which can return
	//		  host definitions based on the configuration file.
	glog.Infof("agent start staticips: %v [%d]", d.hostConfig.StaticIPs, len(d.hostConfig.StaticIPs))
	if err := d.rpcServer.RegisterName("Agent", d.newMock()); err != nil {
		return fmt.Errorf("could not register Agent RPC server: %v", err)
	}
	glog.Infof("finished rpcServer.RegisterName")
	// FIXME: To control 'Active' flag in the UI need to conditionally connect to zookeeper
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
	
	d.host, err = host.Build(d.hostConfig.OutboundIP, rpcPort, d.hostConfig.PoolID, d.hostConfig.Memory)
	if err != nil {
		return fmt.Errorf("Failed to build host: %v", err)
	}

	d.host.ID = fmt.Sprintf("%d", d.hostConfig.HostID)
	d.host.Name = d.hostConfig.Name
	d.host.IPs = make([]host.HostIPResource, 0)
	fmt.Printf("mock host object = %v\n", d.host)

	// FIXME: Override other values in d.host based on corresponding values from d.hostConfig (if specified)
	return nil
}