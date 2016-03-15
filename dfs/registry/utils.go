// Copyright 2015 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package registry

import (
	"path"
	"time"

	"github.com/control-center/serviced/commons"
	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/dfs/docker"
	"github.com/control-center/serviced/domain/registry"
	"github.com/zenoss/glog"
)

// GetRegistryImage returns the registry image from the coordinator index.
func GetRegistryImage(conn client.Connection, id string) (*registry.Image, error) {
	rimagepath := path.Join(zkregistrytags, id)
	var node RegistryImageNode
	if err := conn.Get(rimagepath, &node); err != nil {
		return nil, err
	}
	return &node.Image, nil
}

// SetRegistryImage inserts a registry image into the coordinator index.
func SetRegistryImage(conn client.Connection, rImage registry.Image) error {
	leaderpath := path.Join(zkregistryrepos, rImage.Library, rImage.Repo)
	leadernode := &RegistryImageLeader{HostID: "master"}
	if err := conn.CreateDir(leaderpath); err != nil && err != client.ErrNodeExists {
		glog.Errorf("Could not create repo path %s: %s", leaderpath, err)
		return err
	}
	imagepath := path.Join(zkregistrytags, rImage.ID())
	node := &RegistryImageNode{Image: rImage, PushedAt: time.Unix(0, 0)}
	if err := conn.Create(imagepath, node); err == client.ErrNodeExists {
		leader, err := conn.NewLeader(leaderpath)
		if err != nil {
			glog.Errorf("Could not establish leader at path %s: %s", leaderpath, err)
			return err
		}
		leaderDone := make(chan struct{})
		defer close(leaderDone)
		_, err = leader.TakeLead(leadernode, leaderDone)
		if err != nil {
			return err
		}
		defer leader.ReleaseLead()
		if err := conn.Get(imagepath, node); err != nil {
			return err
		}
		node.Image = rImage
		node.PushedAt = time.Unix(0, 0)
		return conn.Set(imagepath, node)
	} else if err != nil {
		glog.Errorf("Could not create tag path %s: %s", imagepath, err)
		return err
	}
	return nil

}

// GetImageUUID gets an image UUID from the respective tags registry path,
// given an image tag, ex. "kjasd8912833hddhla/core_5.0:latest"
func GetImageUUID(conn client.Connection, tag string) (string, error) {
	imageID, err := commons.ParseImageID(tag)
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
	idpath := path.Join(zkregistrytags, rImage.ID())

	var node RegistryImageNode
	if err = conn.Get(idpath, &node); err != nil {
		return "", err
	}
	return node.Image.UUID, nil
}

// DeleteRegistryImage removes a registry image from the coordinator index.
func DeleteRegistryImage(conn client.Connection, id string) error {
	rimagepath := path.Join(zkregistrytags, id)
	var node RegistryImageNode
	if err := conn.Get(rimagepath, &node); err != nil {
		return err
	}
	if node.Image.Tag == docker.Latest {
		leaderpath := path.Join(zkregistryrepos, node.Image.Library, node.Image.Repo)
		conn.Delete(leaderpath)
	}
	return conn.Delete(rimagepath)
}

// DeleteRegistryLibrary removes all of the leader nodes in the registry
// library.
func DeleteRegistryLibrary(conn client.Connection, library string) error {
	leaderpath := path.Join(zkregistryrepos, library)
	return conn.Delete(leaderpath)
}
