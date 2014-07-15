// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package addressassignment

import (
	"github.com/zenoss/serviced/validation"

	"fmt"
)

//ValidEntity used to make sure AddressAssignment is in a valid state
func (a *AddressAssignment) ValidEntity() error {
	v := validation.NewValidationError()
	v.Add(validation.NotEmpty("ServiceID", a.ServiceID))
	v.Add(validation.NotEmpty("EndpointName", a.EndpointName))
	v.Add(validation.IsIP(a.IPAddr))
	v.Add(validation.ValidPort(int(a.Port)))
	switch a.AssignmentType {
	case "static":
		{
			v.Add(validation.NotEmpty("HostID", a.HostID))
		}
	case "virtual":
		{
			v.Add(validation.NotEmpty("PoolID", a.PoolID))
		}
	default:
		return fmt.Errorf("assignment type must be static of virtual, found %v", a.AssignmentType)
	}

	if v.HasError() {

		return v
	}
	return nil
}
