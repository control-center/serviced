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

package elasticsearch

import (
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain/addressassignment"
)

// GetServiceAddressAssignments fills in all AddressAssignments for the specified serviced id.
func (this *ControlPlaneDao) GetServiceAddressAssignments(serviceID string, assignments *[]addressassignment.AddressAssignment) error {
	var err error
	*assignments, err = this.facade.GetAddrAssignmentsByService(datastore.Get(), serviceID)
	if assignments == nil {
		*assignments = make([]addressassignment.AddressAssignment, 0)
	}
	return err
}
