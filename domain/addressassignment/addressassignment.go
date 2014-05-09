// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package addressassignment

//AddressAssignment is used to track Ports that have been assigned to a Service.
type AddressAssignment struct {
	ID             string //Generated id
	AssignmentType string //Static or Virtual
	HostID         string //Host id if type is Static
	PoolID         string //Pool id if type is Virtual
	IPAddr         string //Used to associate to resource in Pool or Host
	Port           uint16 //Actual assigned port
	ServiceID      string //Service using this assignment
	EndpointName   string //Endpoint in the service using the assignment
}
