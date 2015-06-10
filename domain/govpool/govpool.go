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

package govpool

import "github.com/control-center/serviced/datastore"

// A GovernedPool maps to a ResourcePool and provides upstream information about
// the RemotePool that it correlates.
type GovernedPool struct {
	PoolID        string
	RemotePoolID  string
	RemoteAddress string
	datastore.VersionedEntity
}

// New creates a new GovernedPool object
func New(poolID, remotePoolID, remoteAddress string) *GovernedPool {
	return &GovernedPool{
		PoolID:        poolID,
		RemotePoolID:  remotePoolID,
		RemoteAddress: remoteAddress,
	}
}