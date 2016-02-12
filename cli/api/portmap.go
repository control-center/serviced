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

package api

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/control-center/serviced/domain/servicedefinition"
)

// PortMap parses remote and local port data from the command line
type PortMap map[string]servicedefinition.EndpointDefinition

// Set converts a port mapping string from the command line to a PortMap object
func (m *PortMap) Set(value string) error {
	parts := strings.Split(value, ":")
	if len(parts) != 3 {
		return fmt.Errorf("bad format: %s; must be PROTOCOL:PORTNUM:PORTNAME", value)
	}
	protocol := parts[0]
	switch protocol {
	case "tcp", "udp":
	default:
		return fmt.Errorf("unsupported protocol: %s (udp|tcp)", protocol)
	}
	portnum, err := strconv.ParseUint(parts[1], 10, 16)
	if err != nil {
		return fmt.Errorf("invalid port number: %s", parts[1])
	}
	portname := parts[2]
	if portname == "" {
		return fmt.Errorf("port name cannot be empty")
	}
	port := fmt.Sprintf("%s:%d", protocol, portnum)
	(*m)[port] = servicedefinition.EndpointDefinition{Protocol: protocol, PortNumber: uint16(portnum), Application: portname}
	return nil
}

func (m *PortMap) String() string {
	var mapping []string
	for _, v := range *m {
		mapping = append(mapping, fmt.Sprintf("%s:%d:%s", v.Protocol, v.PortNumber, v.Application))
	}
	return strings.Join(mapping, " ")
}
