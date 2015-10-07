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

	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/domain/registry"
)

// GetRegistryImage returns the registry image from the coordinator index.
func GetRegistryImage(conn client.Connection, id string) (*registry.Image, error) {
	rimagepath := path.Join(zkregistrypath, id)
	var node RegistryImageNode
	if err := conn.Get(rimagepath, &node); err != nil {
		return nil, err
	}
	return node.Image, nil
}

// SetRegistryImage inserts a registry image into the coordinator index.
func SetRegistryImage(conn client.Connection, rImage *registry.Image) error {
	rimagepath := path.Join(zkregistrypath, rImage.ID())
	node := &RegistryImageNode{Image: rImage, PushedAt: time.Unix(0, 0)}
	err := conn.Create(rimagepath, node)
	if err == client.ErrNodeExists {
		leader := conn.NewLeader(rimagepath, &RegistryImageLeader{HostID: "master"})
		if _, err := leader.TakeLead(); err != nil {
			return err
		}
		defer leader.ReleaseLead()
		return conn.Set(rimagepath, node)
	}
	return err
}

// DeleteRegistryImage removes a registry image from the coordinator index.
func DeleteRegistryImage(conn client.Connection, id string) error {
	rimagepath := path.Join(zkregistrypath, id)
	leader := conn.NewLeader(rimagepath, &RegistryImageLeader{HostID: "master"})
	if _, err := leader.TakeLead(); err != nil {
		return err
	}
	defer leader.ReleaseLead()
	return conn.Delete(path.Join(zkregistrypath, id))
}
