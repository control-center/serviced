// Copyright 2014 The Serviced Authors.
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
	"github.com/control-center/serviced/coordinator/client/zookeeper"
	"github.com/control-center/serviced/domain/host"
	"github.com/zenoss/glog"
)

// Server manages the exporting of a file system to clients.
type Server struct {
	host    *host.Host
	driver  StorageDriver
	monitor *Monitor
}

// StorageDriver is an interface that storage subsystem must implement to be used
// by this packages Server implementation.
type StorageDriver interface {
	ExportPath() string
	SetClients(clients ...string)
	Sync() error
	Restart() error
	Stop() error
}

// NewServer returns a Server object to manage the exported file system
func NewServer(driver StorageDriver, host *host.Host, volumesPath string) (*Server, error) {
	if len(driver.ExportPath()) < 9 {
		return nil, fmt.Errorf("export path can not be empty")
	}

	monitor, err := NewMonitor(driver, getDefaultNFSMonitorMasterInterval(), volumesPath)
	if err != nil {
		return nil, fmt.Errorf("unable to create new monitor %s", err)
	}

	s := &Server{
		host:    host,
		driver:  driver,
		monitor: monitor,
	}

	return s, nil
}

func (s *Server) Run(shutdown <-chan interface{}, conn client.Connection) error {
	node := &Node{
		Host:       *s.host,
		ExportPath: fmt.Sprintf("%s:%s", s.host.IPAddr, s.driver.ExportPath()),
		ExportTime: strconv.FormatInt(time.Now().UnixNano(), 16),
	}

	// Create the storage leader and client nodes
	if exists, _ := conn.Exists("/storage/leader"); !exists {
		conn.CreateDir("/storage/leader")
	}

	storageClientsPath := "/storage/clients"

	if exists, _ := conn.Exists(storageClientsPath); !exists {
		conn.CreateDir(storageClientsPath)
	}

	leader := conn.NewLeader("/storage/leader", node)
	leaderW, err := leader.TakeLead()
	if err != zookeeper.ErrDeadlock && err != nil {
		glog.Errorf("Could not take storage lead: %s", err)
		return err
	}

	// monitor dfs; log warnings each cycle; restart dfs if needed
	go s.monitor.MonitorDFSVolume(path.Join("/exports", s.driver.ExportPath()), s.host.IPAddr, node.ExportTime, shutdown, s.monitor.DFSVolumeMonitorPollUpdateFunc)

	// loop until shutdown event
	defer leader.ReleaseLead()

	for {
		clients, clientW, err := conn.ChildrenW(storageClientsPath)
		if err != nil {
			glog.Errorf("Could not set up watch for storage clients: %s", err)
			return err
		}

		s.monitor.SetMonitorStorageClients(conn, storageClientsPath)
		s.driver.SetClients(clients...)
		if err := s.driver.Sync(); err != nil {
			glog.Errorf("Error syncing driver: %s", err)
			return err
		}

		select {
		case e := <-clientW:
			glog.Info("storage.server: receieved event: %s", e)
		case <-leaderW:
			err := fmt.Errorf("storage.server: lost lead")
			return err
		case <-shutdown:
			return nil
		}
	}
}
