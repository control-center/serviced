// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package service

import (
	"github.com/zenoss/serviced/commons"
	"github.com/zenoss/serviced/validation"
)

//ValidEntity validate that Service has all required fields
func (s *Service) ValidEntity() error {

	vErr := validation.NewValidationError()
	vErr.Add(validation.NotEmpty("ID", s.Id))
	vErr.Add(validation.NotEmpty("Name", s.Name))
	vErr.Add(validation.NotEmpty("PoolID", s.PoolId))

	vErr.Add(validation.StringIn(s.Launch, commons.AUTO, commons.MANUAL))
	vErr.Add(validation.IntIn(s.DesiredState, SVCRun, SVCStop, SVCRestart))

	if vErr.HasError() {
		return vErr
	}
	return nil
}
