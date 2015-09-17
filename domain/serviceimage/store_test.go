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

package serviceimage

import (
    "github.com/control-center/serviced/datastore"
    "github.com/control-center/serviced/datastore/elastic"
    . "gopkg.in/check.v1"

    "fmt"
    "reflect"
    "testing"
)


// This plumbs gocheck into testing
func TestServiceImageStore(t *testing.T) { TestingT(t) }

type ServiceImageSuite struct {
    elastic.ElasticTest
    ctx datastore.Context
    store  *ServiceImageStore
}

var _ = Suite(&ServiceImageSuite{
    ElasticTest: elastic.ElasticTest{
        Index:    "controlplane",
        Mappings: []elastic.Mapping{MAPPING},
    }})

func (suite *ServiceImageSuite) SetUpTest(c *C) {
    suite.ElasticTest.SetUpTest(c)
    datastore.Register(suite.Driver())
    suite.ctx = datastore.Get()
    suite.store = NewStore()
}

func (suite *ServiceImageSuite) TestServiceImageStore_Key(c *C) {
    key := ImageKey("imageId")

    c.Assert(key, NotNil)
    c.Assert(key.Kind(), Equals, "serviceimage")
    c.Assert(key.ID(), Equals, "imageId")
}

func (suite *ServiceImageSuite) TestServiceImageStore_TrimmedKey(c *C) {
    key := ImageKey("  imageId  ")

    c.Assert(key, NotNil)
    c.Assert(key.Kind(), Equals, "serviceimage")
    c.Assert(key.ID(), Equals, "imageId")
}

func (suite *ServiceImageSuite) TestGetImagesByImageID_EmptyTag(c *C) {
    results, err := suite.store.GetImagesByImageID(suite.ctx, "")

    c.Assert(results, IsNil)
    c.Assert(err, NotNil)
    c.Assert(err, ErrorMatches, "empty imageID not allowed")
}

func (suite *ServiceImageSuite) TestGetImagesByImageID_BlankTag(c *C) {
    results, err := suite.store.GetImagesByImageID(suite.ctx, "  ")

    c.Assert(results, IsNil)
    c.Assert(err, NotNil)
    c.Assert(err, ErrorMatches, "empty imageID not allowed")
}

func (suite *ServiceImageSuite) TestGetImagesByImageID_EmptyDB(c *C) {
    results, err := suite.store.GetImagesByImageID(suite.ctx, "someTag")

    c.Assert(results, IsNil)
    c.Assert(err, IsNil)
}

func (suite *ServiceImageSuite) TestGetImagesByImageID_NotFound(c *C) {
    image := suite.buildAndStoreImage(c, "someImageID")
    defer suite.store.Delete(suite.ctx, image.Key())

    results, err := suite.store.GetImagesByImageID(suite.ctx, "someUndefinedTag")

    c.Assert(results, IsNil)
    c.Assert(err, IsNil)
}

func (suite *ServiceImageSuite) TestGetImagesByImageID_FoundOne(c *C) {
    // Load some other records first so we're looking in a set with >1 item
    filler1 := suite.buildAndStoreImage(c, "fillerImage1")
    defer suite.store.Delete(suite.ctx, filler1.Key())
    filler2 := suite.buildAndStoreImage(c, "fillerImage2")
    defer suite.store.Delete(suite.ctx, filler2.Key())

    // Add the image under test
    targetID := "targetImageID"
    image := suite.buildAndStoreImage(c, targetID)
    defer suite.store.Delete(suite.ctx, image.Key())

    results, err := suite.store.GetImagesByImageID(suite.ctx, targetID)

    c.Assert(results, NotNil)
    c.Assert(err, IsNil)
    c.Assert(len(results), Equals, 1)
    c.Assert(results[0], ImageEquals, image)
}
//
//func (suite *ServiceImageSuite) TestGetImagesByImageID_FoundMoreThanOne(c *C) {
//    // Load some other records first so we're looking in a set with >1 item
//    filler1 := suite.buildAndStoreImage(c, "fillerImage1")
//    defer suite.store.Delete(suite.ctx, filler1.Key())
//    filler2 := suite.buildAndStoreImage(c, "fillerImage2")
//    defer suite.store.Delete(suite.ctx, filler2.Key())
//
//    // Add the images under test
//    targetID := "someImageID"
//    image1 := buildValidNamedImage(targetID)
//    suite.storeImage(c, image1)
//    defer suite.store.Delete(suite.ctx, image1.Key())
//
//    image2 := buildValidNamedImage(targetID)
//    image2.UUID = "someDifferentUUID"
//    suite.storeImage(c, image2)
//    defer suite.store.Delete(suite.ctx, image2.Key())
//
//    results, err := suite.store.GetImagesByImageID(suite.ctx, targetID)
//
//    c.Assert(results, NotNil)
//    c.Assert(err, IsNil)
//    c.Assert(len(results), Equals, 2)
//    c.Assert(results[0], ImageEquals, image1)
//    c.Assert(results[1], ImageEquals, image2)
//}

func (suite *ServiceImageSuite) TestGetImagesByStatus_InvalidStatus(c *C) {
    results, err := suite.store.GetImagesByStatus(suite.ctx, -1)

    c.Assert(results, IsNil)
    c.Assert(err, ErrorMatches, "status -1 is invalid")

    results, err = suite.store.GetImagesByStatus(suite.ctx, 99)

    c.Assert(results, IsNil)
    c.Assert(err, ErrorMatches, "status 99 is invalid")
}

func (suite *ServiceImageSuite) TestGetImagesByStatus_EmptyDB(c *C) {
    results, err := suite.store.GetImagesByStatus(suite.ctx, IMGCreated)

    c.Assert(results, IsNil)
    c.Assert(err, IsNil)
}

func (suite *ServiceImageSuite) TestGetImagesByStatus_NotFound(c *C) {
    image := suite.buildAndStoreImage(c, "someImageID")
    defer suite.store.Delete(suite.ctx, image.Key())

    results, err := suite.store.GetImagesByStatus(suite.ctx, IMGDeployed)

    c.Assert(results, IsNil)
    c.Assert(err, IsNil)
}

func (suite *ServiceImageSuite) TestGetImagesByStatus_FoundOne(c *C) {
    // Load some other records first so we're looking in a set with >1 item
    filler1 := buildValidNamedImage("fillerImage1")
    filler1.Status = IMGDeployed
    suite.storeImage(c, filler1)
    defer suite.store.Delete(suite.ctx, filler1.Key())

    filler2 := buildValidNamedImage("fillerImage2")
    filler2.Status = IMGFailed
    filler2.Error = "bogus error message"
    suite.storeImage(c, filler2)
    defer suite.store.Delete(suite.ctx, filler2.Key())

    // Add the image under test
    targetID := "targetImageID"
    image := suite.buildAndStoreImage(c, targetID)
    defer suite.store.Delete(suite.ctx, image.Key())

    results, err := suite.store.GetImagesByStatus(suite.ctx, IMGCreated)

    c.Assert(results, NotNil)
    c.Assert(err, IsNil)
    c.Assert(len(results), Equals, 1)
    c.Assert(results[0], ImageEquals, image)
}

func (suite *ServiceImageSuite) TestGetImagesByStatus_FoundMoreThanOne(c *C) {
    // Load some other records first so we're looking in a set with >1 item
    filler1 := buildValidNamedImage("fillerImage1")
    filler1.Status = IMGDeployed
    suite.storeImage(c, filler1)
    defer suite.store.Delete(suite.ctx, filler1.Key())

    filler2 := buildValidNamedImage("fillerImage2")
    filler2.Status = IMGFailed
    filler2.Error = "bogus error message"
    suite.storeImage(c, filler2)
    defer suite.store.Delete(suite.ctx, filler2.Key())

    // Add the images under test
    image1 := suite.buildAndStoreImage(c, "image1")
    defer suite.store.Delete(suite.ctx, image1.Key())

    image2 := suite.buildAndStoreImage(c, "image2")
    defer suite.store.Delete(suite.ctx, image2.Key())

    results, err := suite.store.GetImagesByStatus(suite.ctx, IMGCreated)

    c.Assert(results, NotNil)
    c.Assert(err, IsNil)
    c.Assert(len(results), Equals, 2)
    c.Assert(results[0], ImageEquals, image1)
    c.Assert(results[1], ImageEquals, image2)
}

func (suite *ServiceImageSuite) buildAndStoreImage(c *C, imageID string) *ServiceImage {
    image := buildValidNamedImage(imageID)
    return suite.storeImage(c, image)
}

func (suite *ServiceImageSuite) storeImage(c *C, image *ServiceImage) *ServiceImage {
    if err := suite.store.Put(suite.ctx, image.Key(), image); err != nil {
        c.Errorf("Unexpected error: %v", err)
    }
    return image
}


// The ImageEquals Checker verifies that the actual ServiceImage is equal to
// the expected value, according to ServiceImage.Equals.
//
// For example:
//
//     c.Assert(actual, ImageEquals, expected)
//
var ImageEquals Checker = &imageEqualsChecker{
    &CheckerInfo{Name: "ImageEquals", Params: []string{"obtained", "expected"}},
}

type imageEqualsChecker struct {
    *CheckerInfo
}

func (checker *imageEqualsChecker) Check(params []interface{}, names []string) (result bool, error string) {
    defer func() {
        if v := recover(); v != nil {
            result = false
            error = fmt.Sprint(v)
        }
    }()

    actual := getImageParam(params[0])
    expected := getImageParam(params[1])
    return actual.Equals(expected), ""
}

// Returns a pointer to ServiceImage even if the value of param was an actual
// instance of ServiceImage instead of a pointer
func getImageParam(param interface{}) *ServiceImage {
    var image *ServiceImage

    if reflect.TypeOf(param).String() == "serviceimage.ServiceImage" {
        imageObj := param.(ServiceImage)
        image = &imageObj
    } else {
        image = param.(*ServiceImage)
    }
    return image
}
