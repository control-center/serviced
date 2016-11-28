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

package pool

import (
	"fmt"
	"strings"

	"github.com/control-center/serviced/validation"
)

//ValidEntity validates Host fields
func (p *ResourcePool) ValidEntity() error {

	trimmedID := strings.TrimSpace(p.ID)
	violations := validation.NewValidationError()
	violations.Add(validation.NotEmpty("Pool.ID", p.ID))
	violations.Add(validation.ValidPoolId(p.ID))
	violations.Add(validation.StringsEqual(p.ID, trimmedID, "leading and trailing spaces not allowed for pool id"))

	trimmedRealm := strings.TrimSpace(p.Realm)
	violations.Add(validation.NotEmpty("Pool.Realm", p.Realm))
	violations.Add(validation.StringsEqual(p.Realm, trimmedRealm, "leading and trailing spaces not allowed for pool realm"))

	if p.ConnectionTimeout < 0 {
		violations.Add(validation.NewViolation(fmt.Sprintf("connection timeout cannot be less than 0")))
	}

	if len(violations.Errors) > 0 {
		return violations
	}
	return nil
}
