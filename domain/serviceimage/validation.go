// Copyright 2015 The Serviced Authors.
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

package serviceimage

import (
	"github.com/control-center/serviced/validation"
	"github.com/zenoss/glog"

	"errors"
	"strings"
)

// ValidEntity validates ServiceImage fields
func (i *ServiceImage) ValidEntity() error {
	glog.V(4).Info("Validating ServiceImage")
	violations := validation.NewValidationError()

	trimmedImageID := strings.TrimSpace(i.ImageID)
	violations.Add(validation.NotEmpty("ServiceImage.ImageID", i.ImageID))
	violations.Add(validation.StringsEqual(i.ImageID, trimmedImageID, "leading and trailing spaces not allowed for ServiceImage.ImageID"))

	trimmedUUID := strings.TrimSpace(i.UUID)
	violations.Add(validation.NotEmpty("ServiceImage.UUID", i.UUID))
	violations.Add(validation.StringsEqual(i.UUID, trimmedUUID, "leading and trailing spaces not allowed for ServiceImage.UUID"))

	trimmedHostID := strings.TrimSpace(i.HostID)
	violations.Add(validation.NotEmpty("ServiceImage.HostID", i.HostID))
	violations.Add(validation.StringsEqual(i.HostID, trimmedHostID, "leading and trailing spaces not allowed for ServiceImage.HostID"))
	if len(trimmedHostID) > 0 && trimmedHostID == i.HostID {
		violations.Add(validation.ValidHostID(i.HostID))
	}

	violations.Add(validation.IntIn(int(i.Status), int(IMGCreated), int(IMGDeployed), int(IMGFailed)))

	if i.Status == IMGFailed {
		violations.Add(validation.NotEmpty("ServiceImage.Error", i.Error))
	} else if len(i.Error) > 0 {
		violations.Add(errors.New("ServiceImage.Error must be blank for Created or Deployed images"))
	}

	if len(violations.Errors) > 0 {
		return violations
	}
	return nil
}
