// Copyright 2016 The Serviced Authors.
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

package service

import (
	"fmt"
	"path"

	log "github.com/Sirupsen/logrus"
	"github.com/control-center/serviced/coordinator/client"
)

// ExportDetails describes a port binding for an endpoint as it is
// presented on the coordinator.
type ExportDetails struct {
	ExportBinding
	PrivateIP  string
	InstanceID int
	version    interface{}
}

// Version implements client.Node
func (node *ExportDetails) Version() interface{} {
	return node.version
}

// SetVersion implements client.Node
func (node *ExportDetails) SetVersion(version interface{}) {
	node.version = version
}

// RegisterExport exposes an exported endpoint
func RegisterExport(shutdown <-chan struct{}, conn client.Connection, tenantID string, export ExportDetails) {
	logger := log.WithFields(log.Fields{
		"TenantID":    tenantID,
		"Application": export.Application,
		"InstanceID":  export.InstanceID,
	})

	basepth := path.Join("/net/export", tenantID, export.Application, fmt.Sprintf("%s-%s-%d", tenantID, export.Application, export.InstanceID))
	pth := basepth
	defer func() {
		if err := conn.Delete(pth); err != nil {
			logger.WithError(err).Error("Could not remove endpoint")
		}
	}()

	done := make(chan struct{})
	defer func() { close(done) }()
	for {
		epLogger := logger.WithFields(log.Fields{
			"ExportPath": pth,
		})

		// check the export endpoint path
		ok, ev, err := conn.ExistsW(pth, done)
		if err != nil {
			epLogger.WithError(err).Error("Could not look up endpoint")
			return
		}

		// if the path doesn't exist, create it
		if !ok {
			epLogger.Debug("Registering endpoint")
			pth, err = conn.CreateEphemeral(basepth, &export)
			if err != nil {
				epLogger.WithError(err).Error("Could not create endpoint")
				return
			}
			continue
		}

		epLogger.Debug("Watching endpoint")

		select {
		case <-ev:
		case <-shutdown:
			epLogger.Debug("Listener shutting down")
			return
		}
		close(done)
		done = make(chan struct{})
	}
}

// TrackExports keeps track of changes to the list of exports for given import
func TrackExports(shutdown <-chan struct{}, conn client.Connection, tenantID, application string) <-chan []ExportDetails {
	exportsChan := make(chan []ExportDetails)
	go func() {
		defer close(exportsChan)

		// lets keep track of the binds that we have already looked up
		exportMap := make(map[string]ExportDetails)

		// get the path to the export
		pth := path.Join("/net/export", tenantID, application)

		// set up the logger
		logger := log.WithFields(log.Fields{
			"TenantID":    tenantID,
			"Application": application,
		})
		logger.Debug("Starting listener for export")

		done := make(chan struct{})
		defer func() { close(done) }()
		for {

			// check if the path exists
			ok, ev, err := conn.ExistsW(pth, done)
			if err != nil {
				logger.WithError(err).Error("Could not monitor application")
				return
			}

			// watch the path for children
			var ch []string
			if ok {
				ch, ev, err = conn.ChildrenW(pth, done)
				if err == client.ErrNoNode {
					continue
				} else if err != nil {
					logger.WithError(err).Error("Could not monitor application ports")
					return
				}
			}

			// get the data and make sure it is in sync
			exports := []ExportDetails{}
			chMap := make(map[string]ExportDetails)
			for _, name := range ch {
				export, ok := exportMap[name]

				if !ok {
					err := conn.Get(path.Join(pth, name), &export)
					if err == client.ErrNoNode {
						continue
					} else if err != nil {
						logger.WithError(err).Error("Could not look up export binding")
						return
					}
					logger.WithFields(log.Fields{
						"Name": name,
					}).Debug("New record added")
				}

				exports = append(exports, export)
				chMap[name] = export
			}
			exportMap = chMap

			// send the exports and wait for the next event
			select {
			case exportsChan <- exports:
				// exports received, wait for event
				select {
				case <-ev:
				case <-shutdown:
					return
				}
			case <-ev:
				logger.Debug("Exports updated, getting latest")
			case <-shutdown:
				return
			}
			close(done)
			done = make(chan struct{})
		}
	}()

	return exportsChan
}
