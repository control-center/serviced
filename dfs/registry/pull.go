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

package registry

import (
	"errors"
	"path"
	"time"

	"github.com/control-center/serviced/commons"
	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/dfs/docker"
	"github.com/control-center/serviced/domain/registry"
	dockerclient "github.com/fsouza/go-dockerclient"
	"github.com/zenoss/glog"
)

var (
	ErrOpTimeout = errors.New("operation timed out")
)

// Registry performs specific docker actions based on the registry index
type Registry interface {
	SetConnection(conn client.Connection)
	PullImage(cancel <-chan time.Time, image string) error
	ImagePath(image string) (string, error)
	FindImage(rImg *registry.Image) (*dockerclient.Image, error)
}

// ImagePath returns the proper path to the registry image
func (l *RegistryListener) ImagePath(image string) (string, error) {
	imageID, err := commons.ParseImageID(image)
	if err != nil {
		return "", err
	}
	rImage := &registry.Image{
		Library: imageID.User,
		Repo:    imageID.Repo,
		Tag:     imageID.Tag,
	}
	if imageID.IsLatest() {
		rImage.Tag = docker.Latest
	}
	return path.Join(l.address, rImage.String()), nil
}

// PullImage waits for an image to be available on the docker registry so it
// can be pulled (if it does not exist locally).
func (l *RegistryListener) PullImage(cancel <-chan time.Time, image string) error {
	imageID, err := commons.ParseImageID(image)
	if err != nil {
		return err
	}
	rImage := &registry.Image{
		Library: imageID.User,
		Repo:    imageID.Repo,
		Tag:     imageID.Tag,
	}
	if imageID.IsLatest() {
		rImage.Tag = docker.Latest
	}
	idpath := path.Join(zkregistrytags, rImage.ID())
	regaddr := path.Join(l.address, rImage.String())

	done := make(chan struct{})
	defer func(channel *chan struct{}) { close(*channel) }(&done)
	for {
		var node RegistryImageNode
		evt, err := l.conn.GetW(idpath, &node, done)
		if err != nil {
			if err == client.ErrNoNode {
				glog.Errorf("Image %s not found", regaddr)
			}
			return err
		}
		// check if the image exists locally
		glog.Infof("Looking up image %s", regaddr)
		if err := l.docker.TagImage(node.Image.UUID, regaddr); docker.IsImageNotFound(err) {
			// cannot find the image, so let's try to pull
			glog.Infof("Pulling image %s from the docker registry", regaddr)
			if err := l.docker.PullImage(regaddr); err != nil && !docker.IsImageNotFound(err) {
				glog.Errorf("Could not pull %s: %s", regaddr, err)
				return err
			}
			// was the pull successful?
			if err := l.docker.TagImage(node.Image.UUID, regaddr); docker.IsImageNotFound(err) {
				glog.Infof("Image %s not found by ID (%s), comparing hashes", regaddr, node.Image.UUID)
				//IDs may not match, so lets compare hashes
				if localHash, err := l.docker.GetImageHash(regaddr); err != nil {
					glog.Warningf("Error building hash of image: %s: %s", regaddr, err)
				} else {
					glog.V(2).Infof("For image %s, comparing local hash (%s) to master's hash (%s)", regaddr, localHash, node.Image.Hash)
					if localHash == node.Image.Hash {
						//if the match, the image we have is current, just return
						return nil
					}
				}

				if node.PushedAt.Unix() > 0 {
					// the image is definitely not in the registry, so lets
					// get that push started.
					// also, more than one client may try to update this node
					// at the same time, so there might be a version conflict
					// error; let's just ignore those here.
					node.PushedAt = time.Unix(0, 0)
					if err := l.conn.Set(idpath, &node); err != nil && err != client.ErrBadVersion {
						glog.Errorf("Image %s not found in the docker registry: %s", regaddr, err)
						return err
					}
				}
			} else if err != nil {
				glog.Errorf("Could not update tag %s for image %s: %s", regaddr, node.Image.UUID, err)
				return err
			} else {
				return nil
			}
		} else if err != nil {
			glog.Errorf("Could not update tag %s for image %s: %s", regaddr, node.Image.UUID, err)
			return err
		} else {
			return nil
		}
		glog.Infof("Waiting for image %s to be uploaded into the docker registry (idpath=%s)", regaddr, idpath)
		select {
		case e := <-evt:
			glog.Infof("Got an event: %s", e)
		case <-cancel:
			return ErrOpTimeout
		}

		close(done)
		done = make(chan struct{})
	}
}
