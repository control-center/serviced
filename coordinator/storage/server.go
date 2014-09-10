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
	"time"

	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/coordinator/client/zookeeper"
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/zzk"
	"github.com/zenoss/glog"
)

// Server manages the exporting of a file system to clients.
type Server struct {
	host    *host.Host
	zclient *client.Client
	closing chan struct{}
	driver  StorageDriver
	debug   chan string
}

// StorageDriver is an interface that storage subsystem must implement to be used
// by this packages Server implementation.
type StorageDriver interface {
	ExportPath() string
	SetClients(clients ...string)
	Sync() error
}

// NewServer returns a Server object to manage the exported file system
func NewServer(driver StorageDriver, host *host.Host, zclient *client.Client) (*Server, error) {
	if len(driver.ExportPath()) < 9 {
		return nil, fmt.Errorf("export path can not be empty")
	}

	s := &Server{
		host:    host,
		zclient: zclient,
		closing: make(chan struct{}),
		driver:  driver,
		debug:   make(chan string),
	}

	go s.loop()
	return s, nil
}

// Close informs the Server loop to shutdown.
func (s *Server) Close() {
	close(s.closing)
}

func (s *Server) loop() {
	var err error
	var leadEventC <-chan client.Event
	var e <-chan client.Event
	var children []string

	var (
		conn        client.Connection
		storageLead client.Leader
	)
	node := &Node{
		Host:       *s.host,
		ExportPath: fmt.Sprintf("%s:%s", s.host.IPAddr, s.driver.ExportPath()),
		version:    nil,
	}
	reconnect := func() error {
		conn, err = zzk.GetLocalConnection("/")
		if err != nil {
			glog.Errorf("Error in getting a connection: %v", err)
			return err
		}
		storageLead = conn.NewLeader("/storage/leader", node)
		return nil
	}

	err = reconnect()
restart:
	for {
		glog.V(2).Info("looping")
		// keep from churning if we get errors
		if err != nil {
			select {
			case <-s.closing:
				return
			case <-time.After(time.Second * 10):
				err = reconnect()
				continue restart
			}
		}

		if err = conn.CreateDir("/storage/clients"); err != nil && err != client.ErrNodeExists {
			glog.Errorf("err creating /storage/clients: %s", err)
			continue
		}

		leadEventC, err = storageLead.TakeLead()
		if err != nil && err != zookeeper.ErrDeadlock {
			glog.Errorf("err taking lead: %s", err)
			continue
		}

		children, e, err = conn.ChildrenW("/storage/clients")
		if err != nil {
			glog.Errorf("err getting childrenw: %s", err)
			continue
		}

		s.driver.SetClients(children...)
		if err = s.driver.Sync(); err != nil {
			glog.Errorf("err syncing driver: %s", err)
			continue
		}

		select {
		case <-s.closing:
			glog.Info("storage.server: received closing event")
			storageLead.ReleaseLead()
			return
		case event := <-e:
			glog.Info("storage.server: received event: %s", event)
			continue
		case event := <-leadEventC:
			glog.Info("storage.server: received event on lock: %s", event)
			storageLead.ReleaseLead()
			err = reconnect()
			continue
		}
	}
}
