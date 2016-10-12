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

package web

import (
	"net/http"
	"sync"

	log "github.com/Sirupsen/logrus"
	"github.com/control-center/serviced/zzk/registry"
	"strings"
)

// VHostManager manages all vhosts on a host
type VHostManager struct {
	useTLS bool
	mu     *sync.RWMutex
	vhosts map[string]*VHostHandler
}

// NewVHostManager creates a new vhost manager for a host
func NewVHostManager(useTLS bool) *VHostManager {
	return &VHostManager{
		useTLS: useTLS,
		mu:     &sync.RWMutex{},
		vhosts: make(map[string]*VHostHandler),
	}
}

// Enable implements enables the vhost
func (m *VHostManager) Enable(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	h, ok := m.vhosts[name]
	if !ok {
		h = NewVHostHandler()
		m.vhosts[name] = h
	}
	h.Enable()
}

// Disable disables the vhost
func (m *VHostManager) Disable(name string) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	h, ok := m.vhosts[name]
	if ok {
		h.Disable()
	}
}

// Set updates the vhost endpoints
func (m *VHostManager) Set(name string, data []registry.ExportDetails) {
	m.mu.Lock()
	defer m.mu.Unlock()

	h, ok := m.vhosts[name]
	if ok {
		h.SetExports(data)
	} else {
		h = NewVHostHandler(data...)
		m.vhosts[name] = h
	}
}

// Handle manages a vhost request and returns true if the vhost is enabled
func (m *VHostManager) Handle(httphost string, w http.ResponseWriter, r *http.Request) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// For a request to "xyz.domain.com" See if we have a vhost handler for
	// either "xyz.domain.com" or "xyz"
	subdomain := strings.Split(httphost, ".")[0]
	return m.handle(httphost, w, r) || m.handle(subdomain, w, r)
}

func (m *VHostManager) handle(name string, w http.ResponseWriter, r *http.Request) bool {
	h, ok := m.vhosts[name]
	if ok {
		plog.WithFields(log.Fields{
			"name": name,
		}).Debug("Found VHost handler")
		return h.Handle(m.useTLS, w, r)
	}
	return false
}

// VHostHandler manages a vhost endpoint
type VHostHandler struct {
	exports Exports
	mu      *sync.RWMutex
	enabled bool
}

// NewVHostHandler instantiates a new vhost handler
func NewVHostHandler(data ...registry.ExportDetails) *VHostHandler {
	return &VHostHandler{
		exports: NewRoundRobinExports(data), // default to round-robin
		mu:      &sync.RWMutex{},
		enabled: false,
	}
}

// Enable enables a vhost endpoint
func (h *VHostHandler) Enable() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.enabled = true
}

// Disable disables a vhost endpoint
func (h *VHostHandler) Disable() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.enabled = false
}

// SetExports updates exports for a vhost endpoint
func (h *VHostHandler) SetExports(data []registry.ExportDetails) {
	h.exports.Set(data)
}

// Handle is the vhost handler, returns true if the vhost is enabled
func (h *VHostHandler) Handle(useTLS bool, w http.ResponseWriter, r *http.Request) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// do not handle if disabled
	if !h.enabled {
		return false
	}

	// get the next available export
	export := h.exports.Next()
	if export == nil {
		http.Error(w, "endpoint not available", http.StatusNotFound)
		return true
	}

	logger := plog.WithFields(log.Fields{
		"application": export.Application,
		"hostip":      export.HostIP,
		"privateip":   export.PrivateIP,
		"request":     r,
	})

	logger.Debug("Proxying endpoint")

	// get the reverse proxy for the export
	rp := GetReverseProxy(useTLS, export)

	// Set up the X-Forwarded-Proto header so that downstream servers know
	// the request originated as HTTPS.
	if _, found := r.Header["X-Forwarded-Proto"]; !found {
		r.Header.Set("X-Forwarded-Proto", "https")
	}

	rp.ServeHTTP(w, r)
	return true
}
