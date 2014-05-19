// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package user

import (
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/validation"

	"strings"
)

//ValidEntity validates Host fields
func (u *User) ValidEntity() error {
	glog.V(4).Info("Validating User")

	trimmed := strings.TrimSpace(u.Name)
	violations := validation.NewValidationError()
	violations.Add(validation.NotEmpty("User.Name", u.Name))
	violations.Add(validation.StringsEqual(u.Name, trimmed, "leading and trailing spaces not allowed for user name"))

	violations.Add(validation.NotEmpty("User.Password", u.Password))

	if len(violations.Errors) > 0 {
		return violations
	}
	return nil
}
