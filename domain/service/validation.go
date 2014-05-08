// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package service

import (
	"github.com/zenoss/serviced/commons"
	"github.com/zenoss/serviced/validation"

	"errors"
	"fmt"
)

//ValidEntity validate that Service has all required fields
func (s *Service) ValidEntity() error {

	vErr := validation.NewValidationError()
	vErr.Add(validation.NotEmpty("ID", s.Id))
	vErr.Add(validation.NotEmpty("Name", s.Name))
	vErr.Add(validation.NotEmpty("PoolID", s.PoolId))

	vErr.Add(validation.StringIn(s.Launch, commons.AUTO, commons.MANUAL))
	vErr.Add(validation.IntIn(s.DesiredState, SVC_RUN, SVC_STOP, SVN_RESTART))

	if vErr.HasError() {
		return vErr
	}
	return nil
}

//Validate used to make sure AddressAssignment is in a valid state
func (a *AddressAssignment) Validate() error {
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
