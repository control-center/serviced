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

// VHost describes a vhost endpoint
type VHost struct {
	TenantID    string
	Application string
	Enabled     bool
	version     interface{}
}

// Version implements client.Node
func (node *VHost) Version() interface{} {
	return node.version
}

// SetVersion implements client.Node
func (node *VHost) SetVersion(version interface{}) {
	node.version = version
}

// VHostHandler manages the vhosts for a host
type VHostHandler interface {
	Enable(name string)
	Disable(name string)
	Set(name string, exports []ExportDetails)
}

// VHostListener listens for vhosts on a host
type VHostListener struct {
	conn    client.Connection
	hostID  string
	handler VHostHandler
}

// NewVHostListener instantiates a new vhost listener
func NewVHostListener(hostID string, handler VHostHandler) *VHostListener {
	return &VHostListener{
		hostID:  hostID,
		handler: handler,
	}
}

// SetConnection implements zzk.Listener
func (l *VHostListener) SetConnection(conn client.Connection) {
	l.conn = conn
}

// GetPath implements zzk.Listener
func (l *VHostListener) GetPath(nodes ...string) string {
	parts := append([]string{"/net/vhost", l.hostID}, nodes...)
	return path.Join(parts...)
}

// Ready implements zzk.Listener
func (l *VHostListener) Ready() error {
	return nil
}

// Done implements zzk.Listener
func (l *VHostListener) Done() {
}

// PostProcess implements zzk.Listener
func (l *VHostListener) PostProcess(p map[string]struct{}) {
}

// Spawn manages a specific vhost for a subdomain
func (l *VHostListener) Spawn(shutdown <-chan interface{}, subdomain string) {
	logger := log.WithFields(log.Fields{
		"hostid":    l.hostID,
		"subdomain": subdomain,
	})

	// keep a cache of exports that have already been
	// looked up.
	exportMap := make(map[string]ExportDetails)

	// keep track of the on/off state of the export
	isEnabled := false
	defer func() {
		if isEnabled {
			l.handler.Disable(subdomain)
			logger.Debug("Disabled virtual host")
		}
	}()

	done := make(chan struct{})
	defer func() { close(done) }()
	for {

		// set up a watch on the vhost
		pth := l.GetPath(subdomain)
		dat := &VHost{}
		evt, err := l.conn.GetW(pth, dat, done)
		if err == client.ErrNoNode {
			logger.Debug("Virtual host was deleted, exiting")
			return
		} else if err != nil {
			logger.WithError(err).Error("Could not watch subdomain")
			return
		}

		// track the exports if the vhost is enabled
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
						// we need an event, so try again
						continue
					} else if err != nil {
						exLogger.WithFields(log.Fields{
							"Error": err,
						}).Error("Could not track exports for endpoint")
						return
					}
					break
				}
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

			// only send an update if the exports have changed
			if sendUpdate {
				l.handler.Set(subdomain, exports)
			}
		}

		// do something if the state of the vhost has changed
		if isEnabled != dat.Enabled {
			if dat.Enabled {
				l.handler.Enable(subdomain)
				logger.Debug("Enabled vhost")
			} else {
				l.handler.Disable(subdomain)
				logger.Info("Disabled vhost")
			}
			isEnabled = dat.Enabled
		}

		select {
		case <-evt:
		case <-exevt:
		case <-shutdown:
			return
		}
	}
}
