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

const zkregistrypath = "/docker/registry"

// RegistryImageNode is the registry image as it is written into the
// coordinator.
type RegistryImageNode struct {
	Image    *registry.Image
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
	return path.Join(append([]string{zkregistrypath}, nodes...)...)
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
	leader := l.conn.NewLeader(imagepath, &RegistryImageLeader{HostID: l.hostid})
	for {
		// Get the node
		var node RegistryImageNode
		evt, err := l.conn.GetW(imagepath, &node)
		if err != nil {
			glog.Errorf("Could not look up node at %s: %s", imagepath, err)
			return
		}
		// Has the image been pushed?
		if node.Image != nil && node.PushedAt.Unix() <= 0 {
			// Do I have the image?
			if img, err := l.docker.FindImage(node.Image.UUID); err == nil {
				glog.V(1).Infof("Found image %s locally, acquiring lead", node.Image)
				func() {
					// Become the leader so I can push the image
					levt, err := leader.TakeLead()
					if err != nil {
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
							return
						}
						if node.Image == nil || img.ID != node.Image.UUID || node.PushedAt.Unix() > 0 {
							glog.V(1).Infof("Image %s changed, cancelling push", node.Image)
							return
						}
					}
					// Push the image and update the registry
					registrypath := path.Join(l.address, node.Image.String())
					glog.V(1).Infof("Updating registry image %s", registrypath)
					if err := l.docker.TagImage(node.Image.UUID, registrypath); err != nil {
						glog.Errorf("Could not tag %s as %s: %s", node.Image.UUID, registrypath, err)
						return
					} else if err := l.docker.PushImage(registrypath); err != nil {
						glog.Errorf("Could not push %s: %s", registrypath, err)
						return
					}
					// Set the value if I am still the leader
					select {
					case <-levt:
						glog.Errorf("Lost lead; cannot update docker registry")
						return
					default:
						node.PushedAt = time.Now().UTC()
						l.conn.Set(imagepath, &node)
					}
				}()
			}
		}
		glog.Infof("Waiting for image %s to update", node.Image)
		select {
		case <-evt:
		case <-shutdown:
			return
		}
	}
}
