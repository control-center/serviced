package layer

// Copyright 2014 Zenoss, Inc. All rights reserved

import (
	"github.com/zenoss/go-dockerclient"

	"fmt"
	"io"
	"os"
	"testing"
)

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
	go opts.OutputStream.Write([]byte("testimage"))
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

func TestSquash(t *testing.T) {
	client := &mockClientT{}
	for _, tc := range testCases {
		client.exportContainerDatapaths = tc.exportContainerDatapaths
		imageID, err := Squash(client, tc.layer, tc.downTo, "", "")
		if err != tc.err {
			t.Fatalf("unexpected err condition: %s, expected %+v", err, tc.err)
		}
		if imageID != tc.shouldBeEqualTo {
			t.Fatalf("imageID should be %s instead of %s", tc.shouldBeEqualTo, imageID)
		}
	}
}
