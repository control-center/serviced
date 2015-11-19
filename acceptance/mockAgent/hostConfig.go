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

	"github.com/control-center/serviced/utils"
	"github.com/control-center/serviced/validation"
)

type HostConfig struct {
	Name          string   `json:"name"`
	RPCPort       int      `json:"rpcPort"`
	PoolID        string   `json:"pool,omitempty"`
	Memory        uint64   `json:"memory,omitempty"`
	HostID        uint16   `json:"hostID,omitempty"`
	Cores         int      `json:"cores,omitempty"`
	KernelVersion string   `json:"kernelVersion,omitempty"`
	KernelRelease string   `json:"kernelRelease,omitempty"`
	CCRelease     string   `json:"ccRelease,omitempty"`
	OutboundIP    string   `json:"outboundIP,omitempty"`
	StaticIPs     []string `json:"staticIPs,omitempty"`
	Listen        string   `json:"-"`
}

func (hostConfig *HostConfig) setDefaults(address string) error {
	if hostConfig.Listen == "" {
		hostConfig.Listen = fmt.Sprintf(":%d", hostConfig.RPCPort)
	}

	if err := validation.ValidHostID(fmt.Sprintf("%d", hostConfig.HostID)); err != nil {
		return fmt.Errorf("invalid hostid: %d", hostConfig.HostID)
	}

	if hostConfig.OutboundIP == "" || hostConfig.OutboundIP == "%{local_ip}" || hostConfig.OutboundIP == "%{target_host}" {
		var err error
		if address == "" {
			hostConfig.OutboundIP, err = utils.GetIPAddress()
			if err != nil {
				return fmt.Errorf("Failed to acquire ip address: %s", err)
			}
		} else {
			hostConfig.OutboundIP = address
		}
	}

	return nil
}
