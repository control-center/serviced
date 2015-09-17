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

// +build unit

package serviceimage

import (
	"github.com/stretchr/testify/assert"

	"testing"
	"time"
)

func TestServiceImage_StatusToString(t *testing.T) {
	assert.Equal(t, ImageStatus(0).String(), "Created")
	assert.Equal(t, ImageStatus(1).String(), "Deployed")
	assert.Equal(t, ImageStatus(2).String(), "Failed")
	assert.Equal(t, ImageStatus(-1).String(), "unknown")
	assert.Equal(t, ImageStatus(99).String(), "unknown")
}

func TestServiceImageEquals_EmptyObjects(t *testing.T) {
	image1 := &ServiceImage{}
	image2 := &ServiceImage{}

	assert.True(t, image1.Equals(image2))
}

func TestServiceImageEquals_ValidObjects(t *testing.T) {
	image1 := buildValidNewImage()
	image2 := buildValidNewImage()

	assert.True(t, image1.Equals(image2))
}

func TestServiceImageEquals_SameObjects(t *testing.T) {
	image1 := buildValidNewImage()

	assert.True(t, image1.Equals(image1))
}

func TestServiceImageEquals_ImageIDDiffers(t *testing.T) {
	image1 := buildValidNewImage()
	image2 := buildValidNewImage()
	image2.ImageID = "somethingDifferent"

	assert.False(t, image1.Equals(image2))
}

func TestServiceImageEquals_UUIDDiffers(t *testing.T) {
	image1 := buildValidNewImage()
	image2 := buildValidNewImage()
	image2.UUID = "somethingDifferent"

	assert.False(t, image1.Equals(image2))
}

func TestServiceImageEquals_HostIDDiffers(t *testing.T) {
	image1 := buildValidNewImage()
	image2 := buildValidNewImage()
	image2.HostID = "123456"

	assert.False(t, image1.Equals(image2))
}

func TestServiceImageEquals_StatusDiffers(t *testing.T) {
	image1 := buildValidNewImage()
	image2 := buildValidNewImage()
	image2.Status = IMGDeployed

	assert.False(t, image1.Equals(image2))
}

func TestServiceImageEquals_ErrorDiffers(t *testing.T) {
	image1 := buildValidNewImage()
	image2 := buildValidNewImage()
	image2.Error = "some error"

	assert.False(t, image1.Equals(image2))
}

func TestServiceImageEquals_CreatedAtDiffers(t *testing.T) {
	image1 := buildValidNewImage()
	image2 := buildValidNewImage()
	image2.CreatedAt = time.Now()

	assert.False(t, image1.Equals(image2))
}

func TestServiceImageEquals_DeployedAtDiffers(t *testing.T) {
	image1 := buildValidNewImage()
	image2 := buildValidNewImage()
	image2.DeployedAt = time.Now()

	assert.False(t, image1.Equals(image2))
}


func TestServiceImageKey(t *testing.T) {
	image := buildValidNewImage()

	assert.Equal(t, image.Key().ID(), "someImageID")
//    assert.Equal(t, image.Key().ID(), "someImageID-someUUID")
	assert.Equal(t, image.Key().Kind(), "serviceimage")
}
