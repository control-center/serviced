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
	"time"

	"github.com/control-center/serviced/container"
)

// ControllerOptions are options to be run when starting a new proxy server
type ControllerOptions struct {
	ServiceID               string   // The uuid of the service to launch
	InstanceID              string   // The service state instance id
	Command                 []string // The command to launch
	MuxPort                 int      // the TCP port for the remote mux
	Mux                     bool     // True if a remote mux is used
	TLS                     bool     // True if TLS should be used on the mux
	KeyPEMFile              string   // path to the KeyPEMfile
	CertPEMFile             string   // path to the CertPEMfile
	ServicedEndpoint        string
	Autorestart             bool
	MetricForwarderPort     string // port to which container processes send performance data to
	Logstash                bool
	LogstashBinary          string // path to the logstash-forwarder binary
	LogstashConfig          string // path to the logstash-forwarder config file
	LogstashIdleFlushTime   string // how often should log stash flush its logs
	LogstashSettleTime      string // how long to wait for log stash to flush its logs before exiting
	LogstashURL             string // logstash endpoint
	VirtualAddressSubnet    string // The subnet of virtual addresses, 10.3
	MetricForwardingEnabled bool   // Enable metric forwarding from the container
}

func (c ControllerOptions) toContainerControllerOptions() (options container.ControllerOptions, err error) {
	options.ServicedEndpoint = c.ServicedEndpoint
	options.Service.Autorestart = c.Autorestart
	options.Service.InstanceID = c.InstanceID
	options.Service.ID = c.ServiceID
	options.Service.Command = c.Command
	options.Mux.Port = c.MuxPort
	options.Mux.Enabled = c.Mux
	options.Mux.TLS = c.TLS
	options.Mux.KeyPEMFile = c.KeyPEMFile
	options.Mux.CertPEMFile = c.CertPEMFile
	options.Logforwarder.Enabled = c.Logstash
	options.Logforwarder.Path = c.LogstashBinary
	options.Logforwarder.ConfigFile = c.LogstashConfig
	options.Metric.Address = c.MetricForwarderPort
	options.MetricForwarding = c.MetricForwardingEnabled
	options.Metric.RemoteEndoint = "http://localhost:8444/api/metrics/store"
	options.VirtualAddressSubnet = c.VirtualAddressSubnet
	options.Logforwarder.SettleTime, err = time.ParseDuration(c.LogstashSettleTime)
	if err != nil {
		return options, err
	}
	options.Logforwarder.IdleFlushTime, err = time.ParseDuration(c.LogstashIdleFlushTime)
	if err != nil {
		return options, err
	}

	return options, nil
}
