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
	"io"
	"regexp"
	"strings"

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
	PushImageAfterCommit(image string) (string, error)
	PullImage(image string) error
	TagImage(oldImage, newImage string) error
	RemoveImage(image string) error
	FindContainer(ctr string) (*dockerclient.Container, error)
	CommitContainer(ctr, image string) (*dockerclient.Image, error)
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

//Due to an issue in docker 1.8.3 - 1.9.1, after a commit, the push will result in the master having a different imageID than the registry
//  We work aroudn this by deleting the master's image by ID, then re-pulling it.
func (d *DockerClient) PushImageAfterCommit(image string) (string, error) {

	if err := d.PushImage(image); err != nil {
		return "", err
	}

	//now remove the image and re-pull it
	//first find it so we can remove by ID
	img, err := d.FindImage(image)
	if err != nil {
		glog.Errorf("Error finding image %s: %s", image, err)
		return "", err
	}

	if err = d.RemoveImage(img.ID); err != nil {
		glog.Errorf("Error removing image %s: %s", img.ID, err)
		return "", err
	} 
	
	// Re-pull the image
	if err = d.PullImage(image); err != nil {
		glog.Errorf("Error re-pulling the image %s: %s", image, err)
		return "", err
	}

	//now find it again so we can return the new ID
	img, err = d.FindImage(image)
	if err != nil {
		glog.Errorf("Error re-finding image %s: %s", image, err)
		return "", err
	}

	return img.ID, nil
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
