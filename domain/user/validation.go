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

package user

import (
	"github.com/zenoss/glog"
	"github.com/control-center/serviced/validation"

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
