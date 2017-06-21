// Copyright 2017 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package service

import (
	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/domain/host"
)

// RegisteredHostHandler can be used to get hosts that are registered.  Calls
// to get the registered hosts should block until a host comes online.  The cancel
// parameter can be used to stop the call.
type RegisteredHostHandler interface {
	GetRegisteredHosts(pool string) ([]host.Host, error)
}

func NewRegisteredHostHandler(conn client.Connection) RegisteredHostHandler {
	return &DefaultRegisteredHostHandler{connection: conn}
}

type DefaultRegisteredHostHandler struct {
	connection client.Connection
}

func (h *DefaultRegisteredHostHandler) GetRegisteredHosts(pool string) ([]host.Host, error) {
	return GetRegisteredHostsForPool(h.connection, pool)
}
