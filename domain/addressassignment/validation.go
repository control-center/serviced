// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package addressassignment

import (
	"errors"
	"fmt"
)

//ValidEntity used to make sure AddressAssignment is in a valid state
func (a *AddressAssignment) ValidEntity() error {
	if a.ServiceID == "" {
		return errors.New("ServiceId must be set")
	}
	if a.EndpointName == "" {
		return errors.New("EndpointName must be set")
	}
	if a.IPAddr == "" {
		return errors.New("IPAddr must be set")
	}
	if a.Port == 0 {
		return errors.New("Port must be set")
	}
	switch a.AssignmentType {
	case "static":
		{
			if a.HostID == "" {
				return errors.New("HostId must be set for static assignments")
			}
		}
	case "virtual":
		{
			if a.PoolID == "" {
				return errors.New("PoolId must be set for virtual assignments")
			}

		}
	default:
		return fmt.Errorf("Assignment type must be static of virtual, found %v", a.AssignmentType)
	}

	return nil
}
