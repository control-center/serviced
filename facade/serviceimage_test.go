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

// +build integration

package facade

import (
	"github.com/control-center/serviced/domain/serviceimage"
	. "gopkg.in/check.v1"

	"errors"
	"fmt"
	"strings"
	"github.com/control-center/serviced/datastore"
)

func (ft *FacadeTest) TestGetServiceImage(t *C) {
	fmt.Println(" ##### TestGetServiceImage: starting")
	image := serviceimage.ServiceImage{ImageID: "someID", UUID: "someUUID", HostID: "1", Status: serviceimage.IMGCreated}
	if err := ft.Facade.imageStore.Put(ft.CTX, image.Key(), &image); err != nil {
		t.Errorf("Setup failed. Unble to create image %s: %s", image.ImageID, err)
	}

	result, err := ft.Facade.GetServiceImage(ft.CTX, image.ImageID)

	t.Assert(result, NotNil)
	t.Assert(err, IsNil)
	t.Assert(result.ImageID, Equals, image.ImageID)
	t.Assert(result.UUID, Equals, image.UUID)
	t.Assert(result.HostID, Equals, image.HostID)
	t.Assert(result.Status, Equals, image.Status)
	fmt.Println(" ##### TestGetServiceImage: PASSED")
}

func (ft *FacadeTest) TestGetServiceImage_NotFound(t *C) {
	fmt.Println(" ##### TestGetServiceImage_NotFound: starting")
	imageID := "someImageID"

	result, err := ft.Facade.GetServiceImage(ft.CTX, imageID)

	t.Assert(result, IsNil)
	t.Assert(err, NotNil)
	t.Assert(err, ErrorMatches, "No such entity.*")
	fmt.Println(" ##### TestGetServiceImage_NotFound: PASSED")
}


func (ft *FacadeTest) TestPushServiceImage_InvalidArgument(t *C) {
	fmt.Println(" ##### TestPushServiceImage_InvalidArgument: starting")

	err := ft.Facade.PushServiceImage(ft.CTX, nil)

	t.Assert(err, NotNil)
	t.Assert(err, ErrorMatches, "nil Entity")
	fmt.Println(" ##### TestPushServiceImage_InvalidArgument: PASSED")
}

func (ft *FacadeTest) TestPushServiceImage_NewImage(t *C) {
	fmt.Println(" ##### TestPushServiceImage_NewImage: starting")
	image := serviceimage.ServiceImage{ImageID: "someID", UUID: "someUUID", HostID: "1", Status: serviceimage.IMGCreated}
	ft.mockRegistry.On("PushImage", image.ImageID).Return(nil)

	err := ft.Facade.PushServiceImage(ft.CTX, &image)

	t.Assert(err, IsNil)
	t.Assert(int(image.Status), Equals, int(serviceimage.IMGDeployed))
	t.Assert(image.CreatedAt.IsZero(), Equals, false)
	t.Assert(image.DeployedAt.IsZero(), Equals, false)

	// verify the image in the db is correct
	result := serviceimage.ServiceImage{}
	if err := ft.Facade.imageStore.Get(ft.CTX, image.Key(), &result); err != nil {
		t.Errorf("Unble to get image %s: %s", image.ImageID, err)
	}
	t.Assert(result.ImageID, Equals, image.ImageID)
	t.Assert(result.UUID, Equals, image.UUID)
	t.Assert(int(result.Status), Equals, int(serviceimage.IMGDeployed))
	t.Assert(result.CreatedAt, Equals, image.CreatedAt)
	t.Assert(result.DeployedAt, Equals, image.DeployedAt)

	fmt.Println(" ##### TestPushServiceImage_NewImage: PASSED")
}

func (ft *FacadeTest) TestPushServiceImage_NewImagePrePushDBFail(t *C) {
	fmt.Println(" ##### TestPushServiceImage_NewImagePrePushDBFail: starting")
	invalidImage := serviceimage.ServiceImage{ImageID: "someID", UUID: "someUUID"}

	err := ft.Facade.PushServiceImage(ft.CTX, &invalidImage)

	t.Assert(err, NotNil)
	t.Assert(isValidationFailure(err), Equals, true)

	t.Assert(int(invalidImage.Status), Equals, int(serviceimage.IMGFailed))
	t.Assert(invalidImage.Error, Equals, err.Error())
	t.Assert(invalidImage.CreatedAt.IsZero(), Equals, true)
	t.Assert(invalidImage.DeployedAt.IsZero(), Equals, true)

	// verify the image is NOT in the db
	result := serviceimage.ServiceImage{}
	err = ft.Facade.imageStore.Get(ft.CTX, invalidImage.Key(), &result);
	t.Assert(err, NotNil)
	t.Assert(datastore.IsErrNoSuchEntity(err), Equals, true)

	fmt.Println(" ##### TestPushServiceImage_NewImagePrePushDBFail: PASSED")
}

func (ft *FacadeTest) TestPushServiceImage_NewImagePushFail(t *C) {
	fmt.Println(" ##### TestPushServiceImage_NewImagePushFail: starting")
	image := serviceimage.ServiceImage{ImageID: "someID", UUID: "someUUID", HostID: "1", Status: serviceimage.IMGCreated}
	errorStub := errors.New("errorStub: PushImage() failed")
	ft.mockRegistry.On("PushImage", "someID").Return(errorStub)

	err := ft.Facade.PushServiceImage(ft.CTX, &image)

	t.Assert(err, NotNil)
	t.Assert(err.Error(), Equals, errorStub.Error())

	t.Assert(int(image.Status), Equals, int(serviceimage.IMGFailed))
	t.Assert(image.Error, Equals, errorStub.Error())
	t.Assert(image.CreatedAt.IsZero(), Equals, false)
	t.Assert(image.DeployedAt.IsZero(), Equals, true)

	// verify the image in the db is correct
	result := serviceimage.ServiceImage{}
	if err := ft.Facade.imageStore.Get(ft.CTX, image.Key(), &result); err != nil {
		t.Errorf("Unble to get image %s: %s", image.ImageID, err)
	}
	t.Assert(result.ImageID, Equals, image.ImageID)
	t.Assert(result.UUID, Equals, image.UUID)
	t.Assert(int(result.Status), Equals, int(image.Status))
	t.Assert(result.Error, Equals, image.Error)
	t.Assert(result.CreatedAt, Equals, image.CreatedAt)
	t.Assert(result.DeployedAt.IsZero(), Equals, true)

	fmt.Println(" ##### TestPushServiceImage_NewImagePushFail: PASSED")
}


func (ft *FacadeTest) TestPushServiceImage_UpdateImage(t *C) {
	fmt.Println(" ##### TestPushServiceImage_UpdateImage: starting")
	image := serviceimage.ServiceImage{
		ImageID: 	"someID",
		UUID: 		"someUUID",
		HostID: 	"1",
		Status: 	serviceimage.IMGFailed,
		Error:  	"failed image",
	}
	ft.mockRegistry.On("PushImage", "someID").Return(nil)

	err := ft.Facade.PushServiceImage(ft.CTX, &image)

	t.Assert(err, IsNil)
	t.Assert(int(image.Status), Equals, int(serviceimage.IMGDeployed))
	t.Assert(image.CreatedAt.IsZero(), Equals, false)
	t.Assert(image.DeployedAt.IsZero(), Equals, false)

	// verify the image in the db is correct
	result := serviceimage.ServiceImage{}
	if err := ft.Facade.imageStore.Get(ft.CTX, image.Key(), &result); err != nil {
		t.Errorf("Unble to get image %s: %s", image.ImageID, err)
	}
	t.Assert(result.ImageID, Equals, image.ImageID)
	t.Assert(result.UUID, Equals, image.UUID)
	t.Assert(int(result.Status), Equals, int(serviceimage.IMGDeployed))
	t.Assert(result.CreatedAt, Equals, image.CreatedAt)
	t.Assert(result.DeployedAt, Equals, image.DeployedAt)

	fmt.Println(" ##### TestPushServiceImage_UpdateImage: PASSED")
}

func isValidationFailure(err error) bool {
	return strings.Contains(err.Error(), "ValidationError:")
}
