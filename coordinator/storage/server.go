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

package storage

import (
	"fmt"
	"path"

	"strconv"
	"time"

	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/domain/host"
	"github.com/zenoss/glog"
)

const (
	storageLeaderPath  = "/storage/leader"
	storageClientsPath = "/storage/clients"
)

// Server manages the exporting of a file system to clients.
type Server struct {
	driver  StorageDriver
	monitor *Monitor
	node    *Node
	conn    client.Connection
}

// StorageDriver is an interface that storage subsystem must implement to be used
// by this packages Server implementation.
type StorageDriver interface {
	ExportPath() string
	SetClients(clients ...string)
	Sync() error
	Restart() error
}

// NewServer returns a Server object to manage the exported file system
func NewServer(driver StorageDriver, volumesPath string) (*Server, error) {
	if len(driver.ExportPath()) < 9 {
		return nil, fmt.Errorf("export path can not be empty")
	}

	monitor, err := NewMonitor(driver, getDefaultNFSMonitorMasterInterval(), volumesPath)
	if err != nil {
		return nil, fmt.Errorf("unable to create new monitor %s", err)
	}

	s := &Server{
		driver:  driver,
		monitor: monitor,
	}

	return s, nil
}

// Leader returns the manager's leader node
// Implements scheduler.Manager
func (s *Server) Leader(conn client.Connection, host *host.Host) client.Leader {
	s.conn = conn
	s.node = &Node{
		Host:       *host,
		ExportPath: fmt.Sprintf("%s:%s", host.IPAddr, s.driver.ExportPath()),
		ExportTime: strconv.FormatInt(time.Now().UnixNano(), 16),
	}

	// Create the storage leader and client nodes
	if exists, _ := conn.Exists(storageLeaderPath); !exists {
		conn.CreateDir(storageLeaderPath)
	}
	if exists, _ := conn.Exists(storageClientsPath); !exists {
		conn.CreateDir(storageClientsPath)
	}

	return conn.NewLeader(storageLeaderPath, s.node)
}

// Run runs the storage leader
// Implements scheduler.Manager
func (s *Server) Run(shutdown <-chan interface{}) error {
	glog.Infof("Starting storage manager")
	// monitor dfs; log warnings each cycle; restart dfs if needed
	go s.monitor.MonitorDFSVolume(path.Join("/exports", s.driver.ExportPath()), s.node.Host.IPAddr, s.node.ExportTime, shutdown, s.monitor.DFSVolumeMonitorPollUpdateFunc)

	for {
		clients, clientW, err := s.conn.ChildrenW(storageClientsPath)
		if err != nil {
			glog.Errorf("Could not set up watch for storage clients: %s", err)
			return err
		}

		s.monitor.SetMonitorStorageClients(s.conn, storageClientsPath)
		s.driver.SetClients(clients...)
		if err := s.driver.Sync(); err != nil {
			glog.Errorf("Error syncing driver: %s", err)
			return err
		}

		select {
		case e := <-clientW:
			glog.Info("storage.server: receieved event: %s", e)
		case <-shutdown:
			glog.Infof("Shutting down storage manager")
			return nil
		}
	}
}
