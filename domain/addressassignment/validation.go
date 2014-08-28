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

import (
	"github.com/control-center/serviced/commons"
	"github.com/control-center/serviced/validation"

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
	case commons.STATIC:
		{
			v.Add(validation.NotEmpty("HostID", a.HostID))
		}
	case commons.VIRTUAL:
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
