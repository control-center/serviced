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
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"time"

	"github.com/control-center/serviced/zzk/registry"
)

// rpcache keeps track of all used reverse proxies
var rpcache = &ReverseProxyCache{
	mu:   &sync.RWMutex{},
	data: make(map[ReverseProxyKey]*httputil.ReverseProxy),
}

// ReverseProxyKey is the hash key to identify an instantiated reverse proxy
type ReverseProxyKey struct {
	HostAddress string
	PrivateAddress string
	UseTLS  bool
}

// ReverseProxyCache keeps track of all available reverse proxies
type ReverseProxyCache struct {
	mu   *sync.RWMutex
	data map[ReverseProxyKey]*httputil.ReverseProxy
}

// Get retrieves a reverse proxy from the cache
func (cache *ReverseProxyCache) Get(hostAddress, privateAddress string, useTLS bool) (*httputil.ReverseProxy, bool) {
	cache.mu.RLock()
	defer cache.mu.RUnlock()
	key := ReverseProxyKey{
		HostAddress: hostAddress,
		PrivateAddress: privateAddress,
		UseTLS:  useTLS,
	}
	rp, ok := cache.data[key]
	return rp, ok
}

// Set sets an instantiated reverse proxy
func (cache *ReverseProxyCache) Set(hostAddress, privateAddress string, useTLS bool, rp *httputil.ReverseProxy) {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	key := ReverseProxyKey{
		HostAddress: hostAddress,
		PrivateAddress: privateAddress,
		UseTLS:  useTLS,
	}
	cache.data[key] = rp
}

// GetReverseProxy acquires a reverse proxy from the cache if it exists or
// creates it if it is not found.
func GetReverseProxy(useTLS bool, export *registry.ExportDetails) *httputil.ReverseProxy {
	remoteAddress := ""
	hostAddress := fmt.Sprintf("%s:%d", export.HostIP, export.MuxPort)
	privateAddress := fmt.Sprintf("%s:%d", export.PrivateIP, export.PortNumber)

	// Set the remote address based on whether the container is running on this
	// host.
	if IsLocalAddress(export.HostIP) {
		useTLS = false
		remoteAddress = privateAddress
	} else {
		remoteAddress = hostAddress
	}

	// Look up the reverse proxy in the cache and return it if it exists.
	rp, ok := rpcache.Get(hostAddress, privateAddress, useTLS)
	if ok {
		return rp
	}

	// Set up the reverse proxy and add it to the cache
	rpurl := url.URL{Scheme: "http", Host: remoteAddress}
	transport := &http.Transport{Proxy: http.ProxyFromEnvironment}
	transport.Dial = func(network, addr string) (net.Conn, error) {
		return GetRemoteConnection(useTLS, export)
	}
	rp = httputil.NewSingleHostReverseProxy(&rpurl)
	rp.Transport = transport
	rp.FlushInterval = time.Millisecond * 10
	rpcache.Set(hostAddress, privateAddress, useTLS, rp)
	return rp
}
