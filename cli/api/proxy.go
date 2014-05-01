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
	return container.ControllerOptions{
		TentantID:        c.TentantID,
		ServiceID:        c.ServiceID,
		Command:          c.Command,
		MuxPort:          c.MuxPort,
		Mux:              c.Mux,
		TLS:              c.TLS,
		KeyPEMFile:       c.KeyPEMFile,
		CertPEMFile:      c.CertPEMFile,
		ServicedEndpoint: c.ServicedEndpoint,
		Autorestart:      c.Autorestart,
		Logstash:         c.Logstash,
	}
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
