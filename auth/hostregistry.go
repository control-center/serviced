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

// maintains a map of authenticated hosts and their
// authentication token expiration time

package auth

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

var (
	hostExpirationRegistry = &HostExpirationRegistry{registry: make(map[string]int64)}
	// ErrMissingHost indicates if a host does not
	// exist in the host expiration registry
	ErrMissingHost = errors.New("Host is not present in host expiration registry")
)

// HostExpirationRegistry is a threadsafe map of
// host id to auth expiration time. NOTE: expired hosts
// are not removed from the registry
type HostExpirationRegistry struct {
	registry map[string]int64
	sync.Mutex
}

// Set adds a host to the expiration registry by its
// id and sets its auth expiration time
func (reg *HostExpirationRegistry) Set(hostid string, expires int64) {
	reg.Lock()
	defer reg.Unlock()
	reg.registry[hostid] = expires
}

// Expired checks if a give host's auth has expired
// and returns a bool or an error if the host isn't
// in the registry
func (reg *HostExpirationRegistry) Expired(hostid string) (bool, error) {
	reg.Lock()
	defer reg.Unlock()
	expiration, ok := reg.registry[hostid]
	if !ok {
		// if it doesnt exist, I guess it's expired
		return true, ErrMissingHost
	}
	now := time.Now().Unix()
	return now >= expiration, nil
}

// SetHostExpiration adds a host and its auth expiration
// time to the host expiration registry
func SetHostExpiration(id string, expires int64) {
	fmt.Printf("settin host expiry for %s to %d", id, expires)
	hostExpirationRegistry.Set(id, expires)
}

// HostExpired checks if a host's auth has expired
func HostExpired(id string) bool {
	expired, _ := hostExpirationRegistry.Expired(id)
	fmt.Printf("host %s expired is %t", id, expired)
	return expired
}
