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

package api

import (
	"github.com/control-center/serviced/proxy"
)

// Returns a map of all mux connection info for all hosts
func (a *api) GetMuxConnectionInfo() (map[string]proxy.TCPMuxConnectionInfo, error) {
	client, err := a.connectMaster()
	if err != nil {
		return nil, err
	}

	return client.GetMuxConnectionInfo()
}

// Returns a map of mux connection info for a specific host
func (a *api) GetMuxConnectionInfoForHost(hostID string) (map[string]proxy.TCPMuxConnectionInfo, error) {
	client, err := a.connectMaster()
	if err != nil {
		return nil, err
	}

	return client.GetMuxConnectionInfoForHost(hostID)
}
