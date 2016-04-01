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

package addressassignment

import "github.com/control-center/serviced/datastore"

//AddressAssignment is used to track Ports that have been assigned to a Service.
type AddressAssignment struct {
	ID             string //Generated id
	AssignmentType string //static or virtual
	HostID         string //Host id if type is static
	PoolID         string //Pool id if type is virtual
	IPAddr         string //Used to associate to resource in Pool or Host
	Port           uint16 //Actual assigned port
	ServiceID      string //Service using this assignment
	EndpointName   string //Endpoint in the service using the assignment
	datastore.VersionedEntity
}

// AssignmentRequest is used to couple a serviceId to an IPAddress
type AssignmentRequest struct {
	ServiceID      string
	IPAddress      string
	AutoAssignment bool
}

// EqualIP verifies the address assignment is the same by IP ONLY
func (assign AddressAssignment) EqualIP(b AddressAssignment) bool {
	if assign.PoolID != b.PoolID {
		return false
	} else if assign.IPAddr == b.IPAddr {
		return false
	} else if assign.Port != b.Port {
		return false
	} else if assign.ServiceID != b.ServiceID {
		return false
	} else if assign.EndpointName != b.EndpointName {
		return false
	}
	return true
}
