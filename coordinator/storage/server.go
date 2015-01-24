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
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/control-center/serviced/commons/proc"
	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/coordinator/client/zookeeper"
	"github.com/control-center/serviced/domain/host"
	"github.com/zenoss/glog"
)

// Server manages the exporting of a file system to clients.
type Server struct {
	host   *host.Host
	driver StorageDriver
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
func NewServer(driver StorageDriver, host *host.Host) (*Server, error) {
	if len(driver.ExportPath()) < 9 {
		return nil, fmt.Errorf("export path can not be empty")
	}

	s := &Server{
		host:   host,
		driver: driver,
	}

	return s, nil
}

func (s *Server) Run(shutdown <-chan interface{}, conn client.Connection) error {
	node := &Node{
		Host:       *s.host,
		ExportPath: fmt.Sprintf("%s:%s", s.host.IPAddr, s.driver.ExportPath()),
	}

	// Create the storage leader and client nodes
	if exists, _ := conn.Exists("/storage/leader"); !exists {
		conn.CreateDir("/storage/leader")
	}

	if exists, _ := conn.Exists("/storage/clients"); !exists {
		conn.CreateDir("/storage/clients")
	}

	leader := conn.NewLeader("/storage/leader", node)
	leaderW, err := leader.TakeLead()
	if err != zookeeper.ErrDeadlock && err != nil {
		glog.Errorf("Could not take storage lead: %s", err)
		return err
	}

	processExportedVolumeChangeFunc := func(mountpoint string, isExported bool) {
		if !isExported {
			glog.Warningf("DFS NFS volume %s may be unexported - further action may be needed i.e: restart nfs", mountpoint)
			/*
				// TODO: restart nfs when active num remotes > 0
				// race condition with restarting nfs on master and mounting on
				// remote is seen if nfs is prematurely started before any remotes
				// check in
				if err := s.driver.Restart(); err != nil {
					glog.Errorf("Error restarting driver: %s", err)
				}
			*/
		}
	}
	go proc.MonitorExportedVolume(path.Join("/exports", s.driver.ExportPath()), getDefaultNFSMonitorInterval(), shutdown, processExportedVolumeChangeFunc)

	defer leader.ReleaseLead()

	for {
		clients, clientW, err := conn.ChildrenW("/storage/clients")
		if err != nil {
			glog.Errorf("Could not set up watch for storage clients: %s", err)
			return err
		}

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

func getDefaultNFSMonitorInterval() time.Duration {
	var minMonitorInterval int32 = 60 // in seconds
	var monitorInterval int32 = minMonitorInterval
	monitorIntervalString := os.Getenv("SERVICED_NFS_MONITOR_INTERVAL")
	if len(strings.TrimSpace(monitorIntervalString)) == 0 {
		// ignore unset SERVICED_NFS_MONITOR_INTERVAL
	} else if intVal, intErr := strconv.ParseInt(monitorIntervalString, 0, 32); intErr != nil {
		glog.Warningf("ignoring invalid SERVICED_NFS_MONITOR_INTERVAL of '%s': %s", monitorIntervalString, intErr)
	} else if int32(intVal) < minMonitorInterval {
		glog.Warningf("ignoring invalid SERVICED_NFS_MONITOR_INTERVAL of '%s' < minMonitorInterval:%v seconds", monitorIntervalString, minMonitorInterval)
	} else {
		monitorInterval = int32(intVal)
	}

	return time.Duration(monitorInterval) * time.Second
}
