// Copyright 2015 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package docker

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/control-center/serviced/commons"
	dockerclient "github.com/fsouza/go-dockerclient"
	"github.com/zenoss/glog"
)

const (
	DefaultSocket   = "unix:///var/run/docker.sock"
	DefaultRegistry = "https://index.docker.io/v1/"
	Latest          = "latest"
	MaxLayers       = 127 - 2
)

// IsImageNotFound parses an err to determine whether the image is not found
func IsImageNotFound(err error) bool {
	if err != nil {
		if err == dockerclient.ErrNoSuchImage {
			return true
		}
		var checks = []*regexp.Regexp{
			regexp.MustCompile("Tag .* not found in repository .*"),
			regexp.MustCompile("Error: image .* not found"),
		}
		for _, check := range checks {
			if ok := check.MatchString(err.Error()); ok {
				return true
			}
		}
	}
	return false
}

// Docker is the docker client for the dfs
type Docker interface {
	FindImage(image string) (*dockerclient.Image, error)
	SaveImages(images []string, writer io.Writer) error
	LoadImage(reader io.Reader) error
	PushImage(image string) error
	PullImage(image string) error
	TagImage(oldImage, newImage string) error
	RemoveImage(image string) error
	FindContainer(ctr string) (*dockerclient.Container, error)
	CommitContainer(ctr, image string) (*dockerclient.Image, error)
	GetImageHash(image string) (string, error)
	GetContainerStats(containerID string, timeout time.Duration) (*dockerclient.Stats, error)
	FindImageByHash(imageHash string, checkAllLayers bool) (*dockerclient.Image, error)
}

type DockerClient struct {
	dc *dockerclient.Client
}

func NewDockerClient() (*DockerClient, error) {
	dc, err := dockerclient.NewClient(DefaultSocket)
	if err != nil {
		return nil, err
	}
	return &DockerClient{dc}, nil
}

func (d *DockerClient) FindImage(image string) (*dockerclient.Image, error) {
	return d.dc.InspectImage(image)
}

func (d *DockerClient) SaveImages(images []string, writer io.Writer) error {
	opts := dockerclient.ExportImagesOptions{
		Names:        images,
		OutputStream: writer,
	}
	glog.Infof("Exporting images %s", images)
	return d.dc.ExportImages(opts)
}

func (d *DockerClient) LoadImage(reader io.Reader) error {
	opts := dockerclient.LoadImageOptions{
		InputStream: reader,
	}
	return d.dc.LoadImage(opts)
}

func (d *DockerClient) PushImage(image string) error {
	imageID, err := commons.ParseImageID(image)
	if err != nil {
		return err
	}
	opts := dockerclient.PushImageOptions{
		Name:     imageID.BaseName(),
		Tag:      imageID.Tag,
		Registry: imageID.Registry(),
	}
	creds := d.fetchCreds(imageID.Registry())
	return d.dc.PushImage(opts, creds)
}

func (d *DockerClient) PullImage(image string) error {
	imageID, err := commons.ParseImageID(image)
	if err != nil {
		return err
	}
	opts := dockerclient.PullImageOptions{
		Repository: imageID.BaseName(),
		Registry:   imageID.Registry(),
		Tag:        imageID.Tag,
	}
	creds := d.fetchCreds(imageID.Registry())
	return d.dc.PullImage(opts, creds)
}

func (d *DockerClient) TagImage(oldImage, newImage string) error {
	newImageID, err := commons.ParseImageID(newImage)
	if err != nil {
		return err
	}
	opts := dockerclient.TagImageOptions{
		Repo:  newImageID.BaseName(),
		Tag:   newImageID.Tag,
		Force: true,
	}
	return d.dc.TagImage(oldImage, opts)
}

func (d *DockerClient) RemoveImage(image string) error {
	return d.dc.RemoveImage(image)
}

func (d *DockerClient) FindContainer(ctr string) (*dockerclient.Container, error) {
	return d.dc.InspectContainer(ctr)
}

func (d *DockerClient) CommitContainer(ctr string, image string) (*dockerclient.Image, error) {
	imageID, err := commons.ParseImageID(image)
	if err != nil {
		return nil, err
	}
	opts := dockerclient.CommitContainerOptions{
		Container:  ctr,
		Repository: imageID.BaseName(),
		Tag:        imageID.Tag,
	}
	return d.dc.CommitContainer(opts)
}

func (d *DockerClient) fetchCreds(registry string) (auth dockerclient.AuthConfiguration) {
	if registry = strings.TrimSpace(registry); registry == "" {
		registry = DefaultRegistry
	}
	auths, err := dockerclient.NewAuthConfigurationsFromDockerCfg()
	if err != nil {
		return
	}
	auth, ok := auths.Configs[registry]
	if ok {
		glog.V(1).Infof("Authorized as %s in registry %s", auth.Email, registry)
	}
	return
}

// Generates a unique hash of an image, based on the creation time and command of each layer.
// CC-1750: the hash does NOT include the layer size because during HA testing we ran into
//          an edge case where 2 copies of the same image on different machines had different
//          layer sizes.
func (d *DockerClient) GetImageHash(image string) (string, error) {
	historyList, err := d.dc.ImageHistory(image)
	if err != nil {
		return "", err
	}

	var buffer bytes.Buffer
	for _, history := range historyList {
		imageDataString := fmt.Sprintf("%d-%s\n", history.Created, history.CreatedBy)
		buffer.WriteString(imageDataString)
	}

	h := sha256.New()
	sha := base64.URLEncoding.EncodeToString(h.Sum(buffer.Bytes()))

	return sha, nil
}

func (d *DockerClient) GetContainerStats(containerID string, timeout time.Duration) (*dockerclient.Stats, error) {
	var retErr error
	statsChan := make(chan *dockerclient.Stats)
	stopChan := make(chan bool) // unused since we are NOT streaming
	finishedChan := make(chan bool)

	opts := dockerclient.StatsOptions{
		ID:      containerID,
		Stats:   statsChan,
		Stream:  false, //Pull the stats once then exit
		Done:    stopChan,
		Timeout: timeout,
	}

	// Start streaming stats
	go func() {
		retErr = d.dc.Stats(opts)
		finishedChan <- true
	}()

	// Grab the one result
	result := <-statsChan

	// Wait for the api call to exit
	_ = <-finishedChan

	// Check for errors and return
	if retErr != nil {
		return nil, retErr
	}

	return result, nil
}

// FindImageByHash searches all local images for an image with the given hash
func (d *DockerClient) FindImageByHash(imageHash string, checkAllLayers bool) (*dockerclient.Image, error) {
	opts := dockerclient.ListImagesOptions{
		All:     checkAllLayers,
		Digests: false,
	}

	allImages, err := d.dc.ListImages(opts)
	if err != nil {
		return nil, err
	}

	for _, apiImage := range allImages {
		if hash, err := d.GetImageHash(apiImage.ID); err == nil {
			if hash == imageHash {
				return d.FindImage(apiImage.ID)
			}
		} else {
			glog.Warningf("Error computing hash for %s: %s", apiImage.ID, err)
		}
	}

	return nil, dockerclient.ErrNoSuchImage
}
