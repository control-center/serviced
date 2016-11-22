// Copyright 2014 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package api

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime/pprof"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/control-center/serviced/auth"
	"github.com/control-center/serviced/config"
	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/dao/client"
	"github.com/control-center/serviced/rpc/agent"
	"github.com/control-center/serviced/rpc/master"
	"github.com/control-center/serviced/utils"
	"github.com/control-center/serviced/validation"

	dockerclient "github.com/fsouza/go-dockerclient"
	"github.com/zenoss/glog"
)

type api struct {
	master master.ClientInterface
	agent  *agent.Client
	docker *dockerclient.Client
	dao    dao.ControlPlane // Deprecated
}

var hostAuthenticated bool

func New() API {
	// let lazy init populate each interface as necessary
	return NewAPI(nil, nil, nil, nil)
}

// New creates a new API type
func NewAPI(master master.ClientInterface, agent *agent.Client, docker *dockerclient.Client, dao dao.ControlPlane) API {
	return &api{master: master, agent: agent, docker: docker, dao: dao}
}

// Starts the agent or master services on this host
func (a *api) StartServer() error {
	options := config.GetOptions()
	configureLoggingForLogstash(options.LogstashURL)
	log := log.WithFields(logrus.Fields{
		"staticips": options.StaticIPs,
	})
	log.Debug("Starting server")

	if err := utils.SetCiphers("http", options.TLSCiphers); err != nil {
		return fmt.Errorf("unable to set HTTP TLS Ciphers %v", err)
	}
	log.WithFields(logrus.Fields{
		"ciphers": strings.Join(options.TLSCiphers, ","),
	}).Debug("Set supported TLS ciphers for HTTP")

	if err := utils.SetMinTLS("http", options.TLSMinVersion); err != nil {
		return fmt.Errorf("unable to set minimum HTTP TLS version %v", err)
	}
	log.WithFields(logrus.Fields{
		"minversion": options.TLSMinVersion,
	}).Debug("Set minimum TLS version for HTTP")

	if err := utils.SetCiphers("mux", options.MUXTLSCiphers); err != nil {
		return fmt.Errorf("unable to set MUX TLS Ciphers %v", err)
	}
	log.WithFields(logrus.Fields{
		"ciphers": strings.Join(options.MUXTLSCiphers, ","),
	}).Debug("Set supported TLS ciphers for the mux")

	if err := utils.SetMinTLS("mux", options.MUXTLSMinVersion); err != nil {
		return fmt.Errorf("unable to set minimum MUX TLS version %v", err)
	}
	log.WithFields(logrus.Fields{
		"minversion": options.MUXTLSMinVersion,
	}).Debug("Set minimum TLS version for the mux")

	if err := utils.SetCiphers("rpc", options.RPCTLSCiphers); err != nil {
		return fmt.Errorf("unable to set RPC TLS Ciphers %v", err)
	}
	log.WithFields(logrus.Fields{
		"ciphers": strings.Join(options.RPCTLSCiphers, ","),
	}).Debug("Set supported TLS ciphers for RPC")

	if err := utils.SetMinTLS("rpc", options.RPCTLSMinVersion); err != nil {
		return fmt.Errorf("unable to set minimum RPC TLS version %v", err)
	}
	log.WithFields(logrus.Fields{
		"minversion": options.RPCTLSMinVersion,
	}).Debug("Set minimum TLS version for RPC")

	if len(options.CPUProfile) > 0 {
		f, err := os.Create(options.CPUProfile)
		if err != nil {
			log.WithFields(logrus.Fields{
				"file": options.CPUProfile,
			}).WithError(err).Fatal("Unable to create CPU profile file")
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	d, err := newDaemon(options.Endpoint, options.StaticIPs, options.MasterPoolID, time.Duration(options.TokenExpiration)*time.Second)
	if err != nil {
		return err
	}
	return d.run()
}

func configureLoggingForLogstash(logstashURL string) {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}
	glog.SetLogstashType("serviced-" + hostname)
	glog.SetLogstashURL(logstashURL)
}

// Opens a connection to the master if not already connected
func (a *api) connectMaster() (master.ClientInterface, error) {
	if a.master == nil {
		var err error
		a.master, err = master.NewClient(config.GetOptions().Endpoint)
		if err != nil {
			return nil, fmt.Errorf("could not create a client to the master: %s", err)
		}
		a.authenticateHost()
	}
	return a.master, nil
}

// Opens a connection to the agent if not already connected
func (a *api) connectAgent(address string) (*agent.Client, error) {
	if a.agent == nil {
		var err error
		a.agent, err = agent.NewClient(address)
		if err != nil {
			return nil, fmt.Errorf("could not create a client to the agent: %s", err)
		}
		a.authenticateHost()
	}
	return a.agent, nil
}

// Opens a connection to docker if not already connected
func (a *api) connectDocker() (*dockerclient.Client, error) {
	if a.docker == nil {
		const DockerEndpoint string = "unix:///var/run/docker.sock"
		var err error
		if a.docker, err = dockerclient.NewClient(DockerEndpoint); err != nil {
			return nil, fmt.Errorf("could not create a client to docker: %s", err)
		}
	}
	return a.docker, nil
}

// DEPRECATED: Opens a connection to the DAO if not already connected
func (a *api) connectDAO() (dao.ControlPlane, error) {
	if a.dao == nil {
		var err error
		a.dao, err = client.NewControlClient(config.GetOptions().Endpoint)
		if err != nil {
			return nil, fmt.Errorf("could not create a client to the agent: %s", err)
		}
		a.authenticateHost()
	}
	return a.dao, nil
}


// This will authenticate the host once to get a valid token for any CLI commands
//  that require it.
func (a *api) authenticateHost() error {
	if hostAuthenticated {
		return nil
	}

	options := config.GetOptions()

	// Try to load the master keys, fail silently if they don't exist
	masterKeyFile := filepath.Join(options.IsvcsPath, auth.MasterKeyFileName)
	if err := auth.LoadMasterKeyFile(masterKeyFile); err != nil {
		log.WithError(err).Debug("Unable to load master keys")
	}

	// Load the delegate keys
	delegateKeyFile := filepath.Join(options.EtcPath, auth.DelegateKeyFileName)
	if err := auth.LoadDelegateKeysFromFile(delegateKeyFile); err != nil {
		return err
	}

	// Get our host ID
	myHostID, err := utils.HostID()
	if err != nil {
		return err
	} else if err := validation.ValidHostID(myHostID); err != nil {
		return err
	}

	// Load an auth token once
	tokenFile := filepath.Join(options.EtcPath, auth.TokenFileName)
	getToken := func() (string, int64, error) {
		return a.AuthenticateHost(myHostID)
	}

	if _, err := auth.RefreshToken(getToken, tokenFile); err != nil {
		return err
	}

	hostAuthenticated = true
	return nil
}
