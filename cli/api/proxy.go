package api

import (
	"github.com/zenoss/serviced/container"
)

// ControllerOptions are options to be run when starting a new proxy server
type ControllerOptions struct {
	ServiceID            string   // The uuid of the service to launch
	InstanceID           string   // The service state instance id
	Command              []string // The command to launch
	MuxPort              int      // the TCP port for the remote mux
	Mux                  bool     // True if a remote mux is used
	TLS                  bool     // True if TLS should be used on the mux
	KeyPEMFile           string   // path to the KeyPEMfile
	CertPEMFile          string   // path to the CertPEMfile
	ServicedEndpoint     string
	Autorestart          bool
	MetricForwarderPort  string // port to which container processes send performance data to
	Logstash             bool
	LogstashBinary       string // path to the logstash-forwarder binary
	LogstashConfig       string // path to the logstash-forwarder config file
	VirtualAddressSubnet string // The subnet of virtual addresses, 10.3
}

func (c ControllerOptions) toContainerControllerOptions() container.ControllerOptions {
	options := container.ControllerOptions{}
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
	options.Metric.RemoteEndoint = "http://localhost:8444/api/metrics/store"
	options.VirtualAddressSubnet = c.VirtualAddressSubnet
	return options
}

// Start a service proxy
func (a *api) StartProxy(cfg ControllerOptions) error {
	c, err := container.NewController(cfg.toContainerControllerOptions())
	if err != nil {
		return err
	}
	return c.Run()
}
