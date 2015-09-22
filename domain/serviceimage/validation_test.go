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
	"github.com/control-center/serviced/validation"
	"github.com/stretchr/testify/assert"

	"testing"
)


func TestServiceImageValidate_ValidNewImage(t *testing.T) {
	image := buildValidNewImage()

	err := image.ValidEntity()

	assert.Nil(t, err, "no errors should be returned")
}

func TestServiceImageValidate_ValidFailedImage(t *testing.T) {
	image := buildValidNewImage()
	image.Status = IMGFailed
	image.Error = "some failure message"

	err := image.ValidEntity()

	assert.Nil(t, err, "no errors should be returned")
}

func TestServiceImageValidate_EmptyImageID(t *testing.T) {
	image := buildValidNewImage()
	image.ImageID = ""

	err := image.ValidEntity()

	assertViolations(t, err, []string{"empty string for ServiceImage.ImageID"})
}

func TestServiceImageValidate_BlankImageID(t *testing.T) {
	image := buildValidNewImage()
	image.ImageID = " "

	err := image.ValidEntity()

	assertViolations(t, err,
		[]string{"empty string for ServiceImage.ImageID",
				"leading and trailing spaces not allowed for ServiceImage.ImageID"})
}

func TestServiceImageValidate_UntrimmedImageID(t *testing.T) {
	image := buildValidNewImage()
	image.ImageID = " abc "

	err := image.ValidEntity()

	assertViolations(t, err, []string{"leading and trailing spaces not allowed for ServiceImage.ImageID"})
}

func TestServiceImageValidate_EmptyUUID(t *testing.T) {
	image := buildValidNewImage()
	image.UUID = ""

	err := image.ValidEntity()

	assertViolations(t, err, []string{"empty string for ServiceImage.UUID"})
}

func TestServiceImageValidate_BlankUUID(t *testing.T) {
	image := buildValidNewImage()
	image.UUID = " "

	err := image.ValidEntity()

	assertViolations(t, err,
		[]string{"empty string for ServiceImage.UUID",
				 "leading and trailing spaces not allowed for ServiceImage.UUID"})
}

func TestServiceImageValidate_UntrimmedUUID(t *testing.T) {
	image := buildValidNewImage()
	image.UUID = " abc "

	err := image.ValidEntity()

	assertViolations(t, err, []string{"leading and trailing spaces not allowed for ServiceImage.UUID"})
}

func TestServiceImageValidate_EmptyHostID(t *testing.T) {
	image := buildValidNewImage()
	image.HostID = ""

	err := image.ValidEntity()

	assertViolations(t, err, []string{"empty string for ServiceImage.HostID"})
}

func TestServiceImageValidate_BlankHostID(t *testing.T) {
	image := buildValidNewImage()
	image.HostID = " "

	err := image.ValidEntity()

	assertViolations(t, err,
		[]string{"empty string for ServiceImage.HostID",
				"leading and trailing spaces not allowed for ServiceImage.HostID"})
}

func TestServiceImageValidate_UntrimmedHostID(t *testing.T) {
	image := buildValidNewImage()
	image.HostID = " 1 "

	err := image.ValidEntity()

	assertViolations(t, err, []string{"leading and trailing spaces not allowed for ServiceImage.HostID"})
}

func TestServiceImageValidate_InvalidHostID(t *testing.T) {
	image := buildValidNewImage()
	image.HostID = "invalidHostID"

	err := image.ValidEntity()

	assertViolations(t, err, []string{"unable to convert hostid: invalidHostID to uint"})
}

func TestServiceImageValidate_InvalidStatus(t *testing.T) {
	image := buildValidNewImage()
	image.Status = 99

	err := image.ValidEntity()

	assertViolations(t, err, []string{"int 99 not in [0 1 2]"})
}

func TestServiceImageValidate_InvalidErrorOnNewImage(t *testing.T) {
	image := buildValidNewImage()
	image.Error = "some error"

	err := image.ValidEntity()

	assertViolations(t, err, []string{"ServiceImage.Error must be blank for Created or Deployed images"})
}

func TestServiceImageValidate_InvalidErrorOnDeployedImage(t *testing.T) {
	image := buildValidNewImage()
	image.Status = IMGDeployed
	image.Error = "some error"

	err := image.ValidEntity()

	assertViolations(t, err, []string{"ServiceImage.Error must be blank for Created or Deployed images"})
}

func TestServiceImageValidate_MissingError(t *testing.T) {
	image := buildValidNewImage()
	image.Status = IMGFailed
	image.Error = ""

	err := image.ValidEntity()

	assertViolations(t, err, []string{"empty string for ServiceImage.Error"})
}

func assertViolations(t* testing.T, result error, expectedErrors []string) {
	assert.NotNil(t, result)

	violations := result.(*validation.ValidationError)
	assert.Equal(t, len(violations.Errors), len(expectedErrors))
	for i, expected := range(expectedErrors) {
		assert.Equal(t, violations.Errors[i].Error(), expected)
	}
}
