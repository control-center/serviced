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
	"errors"

	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/dfs/docker"
	"github.com/control-center/serviced/domain/registry"
	"github.com/zenoss/glog"
)

var ErrPushAfterCommit = errors.New("Listener failed to perform post-commit push")

// GetRegistryImage returns the registry image from the coordinator index.
func GetRegistryImage(conn client.Connection, id string) (*registry.Image, error) {
	rimagepath := path.Join(zkregistrytags, id)
	var node RegistryImageNode
	if err := conn.Get(rimagepath, &node); err != nil {
		return nil, err
	}
	return &node.Image, nil
}

func setRegistryImage(conn client.Connection, rImage registry.Image, afterCommit bool) error {
	leaderpath := path.Join(zkregistryrepos, rImage.Library, rImage.Repo)
	leadernode := &RegistryImageLeader{HostID: "master"}
	if err := conn.CreateDir(leaderpath); err != nil && err != client.ErrNodeExists {
		glog.Errorf("Could not create repo path %s: %s", leaderpath, err)
		return err
	}
	imagepath := path.Join(zkregistrytags, rImage.ID())
	node := &RegistryImageNode{Image: rImage, PushedAt: time.Unix(0, 0)}
	if err := conn.Create(imagepath, node); err == client.ErrNodeExists {
		leader := conn.NewLeader(leaderpath, leadernode)
		leaderDone := make(chan struct{})
		defer close(leaderDone)
		_, err := leader.TakeLead(leaderDone)
		if err != nil {
			return err
		}
		defer leader.ReleaseLead()
		if err := conn.Get(imagepath, node); err != nil {
			return err
		}
		node.Image = rImage
		node.PushedAt = time.Unix(0, 0)
		node.AfterCommit = afterCommit
		node.AfterCommitPushFailed = false
		return conn.Set(imagepath, node)
	} else if err != nil {
		glog.Errorf("Could not create tag path %s: %s", imagepath, err)
		return err
	}
	return nil

}

// SetRegistryImage inserts a registry image into the coordinator index.
func SetRegistryImage(conn client.Connection, rImage registry.Image) error {
	return setRegistryImage(conn, rImage, false)
}

// Due to an issue in Docker 1.8.3 - 1.9.1, pushing an image may result in the master having a different imageID than the agents.
//  the solution is to get the new ID after the push and then pass this up to the facade for a re-push.
func SetRegistryImageAfterCommit(conn client.Connection, rImage registry.Image) (string, error) {
	
	if err:=setRegistryImage(conn, rImage, true); err != nil {
		return "", err
	}

	//wait for the image to get pushed, then return the new ID
	imagepath := path.Join(zkregistrytags, rImage.ID())
	for {
		var node RegistryImageNode
		
		evt, err := conn.GetW(imagepath, &node)
		if err != nil {
			glog.Errorf("Error getting node %s: %s", imagepath, err)
			return "", err
		}

		if(node.PushedAt.Unix() != 0) {
			return node.NewID, nil
		} else if node.AfterCommitPushFailed {
			return "", ErrPushAfterCommit
		}

		glog.Infof("Waiting for the image to get pushed")
		select {
			case <-evt:
		}
	}
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
