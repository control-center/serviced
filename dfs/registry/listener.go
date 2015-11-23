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
	"path"
	"time"

	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/dfs/docker"
	"github.com/control-center/serviced/domain/registry"
	"github.com/zenoss/glog"
)

const (
	zkregistryrepos = "/docker/registry/repos"
	zkregistrytags  = "/docker/registry/tags"
)

// RegistryImageNode is the registry image as it is written into the
// coordinator.
type RegistryImageNode struct {
	Image    registry.Image
	PushedAt time.Time
	version  interface{}
}

// Version implements client.Node
func (node *RegistryImageNode) Version() interface{} {
	return node.version
}

// SetVersion implements client.Node
func (node *RegistryImageNode) SetVersion(version interface{}) {
	node.version = version
}

// RegistryImageLeader is the client that can modify the registry index on the
// coordinator.
type RegistryImageLeader struct {
	HostID  string
	version interface{}
}

// Version implements client.Node
func (node *RegistryImageLeader) Version() interface{} {
	return node.version
}

// SetVersion implements client.Node
func (node *RegistryImageLeader) SetVersion(version interface{}) {
	node.version = version
}

// RegistryListener is the push listener for the docker registry index.
type RegistryListener struct {
	// connection to the coordinator
	conn client.Connection
	// docker client
	docker docker.Docker
	// address to the registry (e.g. localhost:5000)
	address string
	// id of the host as recognized by cc
	hostid string
	// timeout to cancel long-running pulls
	pulltimeout time.Duration
}

// NewRegistryListener instantiates a new registry listener
func NewRegistryListener(docker docker.Docker, address, hostid string, timeout time.Duration) *RegistryListener {
	return &RegistryListener{docker: docker, address: address, hostid: hostid, pulltimeout: timeout}
}

// SetConnection implements zzk.Listener
func (l *RegistryListener) SetConnection(conn client.Connection) {
	l.conn = conn
}

// GetPath implements zzk.Listener
func (l *RegistryListener) GetPath(nodes ...string) string {
	return path.Join(append([]string{zkregistrytags}, nodes...)...)
}

// Ready implements zzk.Listener
func (l *RegistryListener) Ready() (err error) { return }

// Done implements zzk.Listener
func (l *RegistryListener) Done() {}

// PostProcess implements zzk.Listener
func (l *RegistryListener) PostProcess(_ map[string]struct{}) {}

// Spawn watches a registry image and will try to push any images that are not
// saved in the registry.
func (l *RegistryListener) Spawn(shutdown <-chan interface{}, id string) {
	imagepath := l.GetPath(id)
	done := make(chan bool)
	defer func(channel *chan bool) { close(*channel) }(&done)
	for {
		// Get the node
		var node RegistryImageNode
		evt, err := l.conn.GetW(imagepath, &node, done)
		if err != nil {
			glog.Errorf("Could not look up node at %s: %s", imagepath, err)
			return
		}
		repopath := path.Join(zkregistryrepos, node.Image.Library, node.Image.Repo)
		reponode := &RegistryImageLeader{HostID: l.hostid}
		leader := l.conn.NewLeader(repopath, reponode)
		// Has the image been pushed?
		glog.V(1).Infof("Spawn id=%s node: %s:%s %s", id, node.Image.Repo, node.Image.Tag, node.Image.Tag)
		if node.PushedAt.Unix() == 0 {
			// Do I have the image?
			if img, err := l.docker.FindImage(node.Image.UUID); err == nil {
				glog.V(1).Infof("Found image %s locally, acquiring lead", node.Image)
				func() {
					// Become the leader so I can push the image
					leaderDone := make(chan bool)
					defer close(leaderDone)
					_, err := leader.TakeLead(leaderDone)
					if err != nil {
						glog.Errorf("Could not take lead %s: %s", imagepath, err)
						return
					}
					defer leader.ReleaseLead()
					// Did I shutdown before I got the lead?
					select {
					case <-shutdown:
						return
					default:
						// Did the image change (or get pushed) before I got the lead?
						if err := l.conn.Get(imagepath, &node); err != nil {
							glog.Errorf("Could not get %s: %s", imagepath, err)
							return
						}
						if img.ID != node.Image.UUID || node.PushedAt.Unix() > 0 {
							glog.V(1).Infof("Image %s changed, cancelling push", node.Image)
							return
						}
					}
					// Push the image and update the registry
					// If the push is unsuccessful, still update the timestamp,
					// so that the push will get retriggered the next time it
					// is needed.
					registrypath := path.Join(l.address, node.Image.String())
					glog.V(1).Infof("Updating registry image %s from path=%s", node.Image.UUID, registrypath)
					if err := l.docker.TagImage(node.Image.UUID, registrypath); err != nil {
						glog.Warningf("Could not tag %s as %s: %s", node.Image.UUID, registrypath, err)
						node.PushedAt = time.Unix(0, 0)
					} else if err := l.docker.PushImage(registrypath); err != nil {
						glog.Warningf("Could not push %s: %s", registrypath, err)
						node.PushedAt = time.Unix(0, 0)
					} else {
						node.PushedAt = time.Now().UTC()
					}
					// The point here is to make sure the node triggers an
					// event regardless of whether the push was successful.
					l.conn.Set(imagepath, &node)
				}()
			} else {
				glog.Errorf("Could not find image %s: %s", node.Image.UUID, err)
			}
		}
		glog.V(1).Infof("Waiting for image %s to update", node.Image)
		select {
		case <-evt:
		case <-shutdown:
			return
		}

		close(done)
		done = make(chan bool)
	}
}
