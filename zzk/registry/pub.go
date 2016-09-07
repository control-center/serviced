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

package registry

import (
	"path"

	log "github.com/Sirupsen/logrus"
	"github.com/control-center/serviced/coordinator/client"
)

// PublicPort describes a public endpoint
type PublicPort struct {
	TenantID    string
	Application string
	ServiceID   string // TODO: search by tenant and application
	Enabled     bool
	Protocol    string
	UseTLS      bool
	version     interface{}
}

// Version implements client.Node
func (node *PublicPort) Version() interface{} {
	return node.version
}

// SetVersion implements client.Node
func (node *PublicPort) SetVersion(version interface{}) {
	node.version = version
}

// PublicPortHandler manages a public port and its exports
type PublicPortHandler interface {
	Enable(port string, protocol string, useTLS bool)
	Disable(port string)
	Set(port string, exports []ExportDetails)
}

// PublicPortListener listens to ports for a provided ip
type PublicPortListener struct {
	conn    client.Connection
	hostID  string
	handler PublicPortHandler
}

// NewPublicPortListener instantiates a new public port listener
// for a provided hostID (or master)
func NewPublicPortListener(hostID string, handler PublicPortHandler) *PublicPortListener {
	return &PublicPortListener{
		hostID:  hostID,
		handler: handler,
	}
}

// SetConnection implements zzk.Listener
func (l *PublicPortListener) SetConnection(conn client.Connection) {
	l.conn = conn
}

// GetPath implements zzk.Listener
func (l *PublicPortListener) GetPath(nodes ...string) string {
	parts := append([]string{"/net/pub", l.hostID}, nodes...)
	return path.Join(parts...)
}

// Ready implements zzk.Listener
func (l *PublicPortListener) Ready() error {
	return nil
}

// Done implements zzk.Listener
func (l *PublicPortListener) Done() {
}

// PostProcess implements zzk.Listener
func (l *PublicPortListener) PostProcess(p map[string]struct{}) {
}

// Spawn monitors the public port and its exports
func (l *PublicPortListener) Spawn(shutdown <-chan interface{}, portAddr string) {
	logger := plog.WithFields(log.Fields{
		"hostid":      l.hostID,
		"portaddress": portAddr,
	})

	pth := l.GetPath(portAddr)

	// keep a cache of exports that have already been
	// looked up.
	exportMap := make(map[string]ExportDetails)

	// keep track of the on/off state of the export
	isEnabled := false
	defer func() {
		if isEnabled {
			l.handler.Disable(portAddr)
			logger.Debug("Disabled port")
		}
	}()

	done := make(chan struct{})
	defer func() { close(done) }()
	for {

		// set up a watch on the port address (e.g. 127.0.0.1:1234, :1234, :::1:1234)
		dat := &PublicPort{}
		evt, err := l.conn.GetW(pth, dat, done)
		if err == client.ErrNoNode {
			logger.Debug("Public port was deleted, exiting")
			return
		} else if err != nil {
			logger.WithError(err).Error("Could not watch public port")
			return
		}

		// track the exports if the port is enabled
		var exevt <-chan client.Event
		if dat.Enabled {
			exLogger := logger.WithFields(log.Fields{
				"tenantid":    dat.TenantID,
				"application": dat.Application,
			})

			var ch []string

			expth := path.Join("/net/export", dat.TenantID, dat.Application)

			// keep checking until we have an event or an error
			for {
				var ok bool
				var err error

				ok, exevt, err = l.conn.ExistsW(expth, done)
				if err != nil {
					exLogger.WithError(err).Error("Could not check exports for endpoint")
					return
				}

				if ok {
					ch, exevt, err = l.conn.ChildrenW(expth, done)
					if err == client.ErrNoNode {
						logger.Debug("Public port was suddenly deleted, retrying")
						close(done)
						done = make(chan struct{})

						// we need an event, so try again
						continue
					} else if err != nil {
						exLogger.WithError(err).Error("Could not track exports for endpoint")
						return
					}
				}
				break
			}

			exports := []ExportDetails{}

			// get the exports and update the cache
			sendUpdate := len(ch) != len(exportMap)
			chMap := make(map[string]ExportDetails)
			for _, name := range ch {
				export, ok := exportMap[name]
				if !ok {
					sendUpdate = true
					if err := l.conn.Get(path.Join(expth, name), &export); err == client.ErrNoNode {
						continue
					} else if err != nil {
						exLogger.WithField("exportkey", name).WithError(err).Error("Could not look up export")
						return
					}
				}
				chMap[name] = export
				exports = append(exports, export)
			}

			exportMap = chMap

			// only set new values if the exports have changed
			if sendUpdate {
				l.handler.Set(portAddr, exports)
				exLogger.Debug("Set new endpoints for export")
			}
		}

		// do something if the state of the port has changed
		if isEnabled != dat.Enabled {
			if dat.Enabled {
				l.handler.Enable(portAddr, dat.Protocol, dat.UseTLS)
				logger.Debug("Enabled port")
			} else {
				l.handler.Disable(portAddr)
				logger.Info("Disabled port")
			}
			isEnabled = dat.Enabled
		}

		select {
		case <-evt:
		case <-exevt:
		case <-shutdown:
			return
		}

		close(done)
		done = make(chan struct{})
	}
}
