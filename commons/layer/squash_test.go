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

// +build unit

package layer

import (
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/fsouza/go-dockerclient"

	. "gopkg.in/check.v1"
)

type TestLayerSuite struct{
	// add suite-specific data here such as mocks
}

// verify TestLayerSuite implements the Suite interface
var _ = Suite(&TestLayerSuite{})

// Wire gocheck into the go test runner
func TestLayer(t *testing.T) { TestingT(t) }

var errUnimplemented = fmt.Errorf("unimplemented")

type mockClientT struct {
	createContainer          docker.Container
	createContainerErr       error
	removeContainerErr       error
	exportContainerDatapaths []string
	exportContainerErr       error
	buildImageErr            error
}

func (c *mockClientT) CreateContainer(docker.CreateContainerOptions) (*docker.Container, error) {
	return &c.createContainer, c.createContainerErr
}

func (c *mockClientT) ImportImage(opts docker.ImportImageOptions) error {
	opts.OutputStream.Write([]byte("testlayer"))
	return nil
}

func (c *mockClientT) InspectImage(string) (*docker.Image, error) {
	return &docker.Image{}, nil
}

func (c *mockClientT) RemoveContainer(docker.RemoveContainerOptions) error {
	return c.removeContainerErr
}

func (c *mockClientT) ExportContainer(options docker.ExportContainerOptions) error {
	if c.exportContainerErr != nil {
		return c.exportContainerErr
	}
	if len(c.exportContainerDatapaths) <= 0 {
		return fmt.Errorf("mock: no more paths to exports")
	}
	path := c.exportContainerDatapaths[0]
	c.exportContainerDatapaths = c.exportContainerDatapaths[1:]
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(options.OutputStream, f)
	return err
}

func (c *mockClientT) BuildImage(docker.BuildImageOptions) error {
	return c.buildImageErr
}

type testCase struct {
	layer                    string
	downTo                   string
	err                      error
	shouldBeEqualTo          string
	exportContainerDatapaths []string
}

var testCases = []testCase{
	testCase{
		layer:  "7f8a29e4050bb8a80d5cb143cae6831555080cf677904c10e8729988d2ac3d6c",
		downTo: "",
		exportContainerDatapaths: []string{"data/7f8a29e4050bb8a80d5cb143cae6831555080cf677904c10e8729988d2ac3d6c.tar"},
		shouldBeEqualTo:          "testlayer",
		err:                      nil,
	},
}

// -rw-rw-r-- 1 dgarcia dgarcia 6656 Jun 20 13:02 ad892dd21d607a1458a722598a2e4d93015c4507abcd0ebfc16a43d4d1b41520.tar

func (s* TestLayerSuite) TestSquash(c *C) {
	client := &mockClientT{}
	for _, tc := range testCases {
		client.exportContainerDatapaths = tc.exportContainerDatapaths
		imageID, err := Squash(client, tc.layer, tc.downTo, "", "")

		c.Assert(err, Equals, tc.err)
		c.Assert(imageID, Equals, tc.shouldBeEqualTo)
	}
}
