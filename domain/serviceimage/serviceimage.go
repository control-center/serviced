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
	"github.com/control-center/serviced/datastore"

	"time"
)

// The status of the image.
type ImageStatus int

const (
	IMGCreated  = ImageStatus(0)
	IMGDeployed = ImageStatus(1)
	IMGFailed   = ImageStatus(2)
)

func (status ImageStatus) String() string {
	switch status {
	case IMGCreated:
		return "Created"
	case IMGDeployed:
		return "Deployed"
	case IMGFailed:
		return "Failed"
	default:
		return "unknown"
	}
}


// Represents a service image that should be in the local docker registry
type ServiceImage struct {
	ImageID		   string 		// the image id; aka a repo tag or repository in docker-speaker
	UUID           string 		// the docker image UUID.
	HostID         string 		// the hostID of the host that created the image.
	Status 		   ImageStatus	// the status of the image.
	Error          string		// contains the error reason if Status==IMGFailed;
								// 		blank for any other value of Status
	CreatedAt      time.Time	// timestamp when the image was created
	DeployedAt     time.Time	// timestamp when the image push completed
	datastore.VersionedEntity
}

// Equals verifies whether two image objects are equal
func (a *ServiceImage) Equals(b *ServiceImage) bool {
	if a.ImageID != b.ImageID {
		return false
	}
	if a.UUID != b.UUID {
		return false
	}
	if a.HostID != b.HostID {
		return false
	}
	if a.Status != b.Status {
		return false
	}
	if a.Error != b.Error {
		return false
	}
	if a.CreatedAt.Unix() != b.CreatedAt.Unix() {
		return false
	}
	if a.DeployedAt.Unix() != b.DeployedAt.Unix() {
		return false
	}

	return true
}

// New creates a new empty image
func New() *ServiceImage {
	image := &ServiceImage{}
	return image
}

func (image *ServiceImage) Key() datastore.Key {
	return ImageKey(image.ImageID)
}
