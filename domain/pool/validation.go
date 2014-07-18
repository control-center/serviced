// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package pool

import (
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/validation"

	"strings"
)

//ValidEntity validates Host fields
func (p *ResourcePool) ValidEntity() error {
	glog.V(4).Info("Validating ResourcePool")

	trimmedID := strings.TrimSpace(p.ID)
	violations := validation.NewValidationError()
	violations.Add(validation.NotEmpty("Pool.ID", p.ID))
	violations.Add(validation.StringsEqual(p.ID, trimmedID, "leading and trailing spaces not allowed for pool id"))

	if len(violations.Errors) > 0 {
		return violations
	}
	return nil
}
