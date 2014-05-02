package api

import (
	"github.com/zenoss/serviced/container"

	"os"
	"os/signal"
	"syscall"
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

func toContainerControllerOptions(c ControllerOptions) container.ControllerOptions {
	options := container.ControllerOptions{}
	options.ServicedEndpoint = c.ServicedEndpoint
	options.Service.TenantID = c.TentantID
	options.Service.Autorestart = c.Autorestart
	options.Service.ID = c.ServiceID
	options.Service.Command = c.Command
	options.Mux.Port = c.MuxPort
	options.Mux.Enabled = c.Mux
	options.Mux.TLS = c.TLS
	options.Mux.KeyPEMFile = c.KeyPEMFile
	options.Mux.CertPEMFile = c.CertPEMFile
	options.Logforwarder.Enabled = c.Logstash
	return options
}

// Start a service proxy
func (a *api) StartProxy(cfg ControllerOptions) error {

	c, err := container.NewController(toContainerControllerOptions(cfg))
	if err != nil {
		return err
	}
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	for {
		select {
		case <-sigc:
			c.Close()
		}
	}
	return nil
}
