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

package strategy

import (
	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/dao/elasticsearch"
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/utils"
)

type Host interface {
	HostID() string
	TotalCores() int
	TotalMemory() uint64
}

type HostServiceInfo interface {
	GetRunningServicesForHost(hostID string, services *[]ServiceConfig) error
}

type ServiceConfig interface {
	GetServiceID() string
	RequestedCores() uint64
	RequestedMemory() utils.EngNotation
}

var _ ServiceConfig = &dao.RunningService{}
var _ HostServiceInfo = &elasticsearch.ControlPlaneDao{}
var _ Host = &host.Host{}

type Strategy interface {
	// The name of this strategy
	Name() string
	// Chooses the best host for svc to run on
	SelectHost(svc ServiceConfig, hosts []Host) (Host, error)
}
