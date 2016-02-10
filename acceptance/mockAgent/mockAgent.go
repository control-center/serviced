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
package main

import (
	"fmt"

	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/rpc/agent"
	"github.com/control-center/serviced/utils"
	"github.com/zenoss/glog"
)

type MockAgent struct {
	mockHost *host.Host
}

func (m *MockAgent) BuildHost(request agent.BuildHostRequest, hostResponse *host.Host) error {
	*hostResponse = host.Host{}

	glog.Infof("Build Host Request: %s:%d, %s, %s", request.IP, request.Port, request.PoolID, request.Memory)

	if _, err := utils.ParseEngineeringNotation(request.Memory); err == nil {
		m.mockHost.RAMLimit = request.Memory
	} else if _, err := utils.ParsePercentage(request.Memory, m.mockHost.Memory); err == nil {
		m.mockHost.RAMLimit = request.Memory
	} else {
		return fmt.Errorf("Could not parse RAM limit: %v", err)
	}
	if request.PoolID != m.mockHost.PoolID {
		m.mockHost.PoolID = request.PoolID
	}
	*hostResponse = *m.mockHost
	return nil
}
