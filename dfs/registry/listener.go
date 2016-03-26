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
	dockerclient "github.com/fsouza/go-dockerclient"
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
}

// NewRegistryListener instantiates a new registry listener
func NewRegistryListener(docker docker.Docker, address, hostid string) *RegistryListener {
	return &RegistryListener{docker: docker, address: address, hostid: hostid}
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
	done := make(chan struct{})
	defer func(channel *chan struct{}) { close(*channel) }(&done)
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
		leader, err := l.conn.NewLeader(repopath)
		if err != nil {
			glog.Errorf("Could not set up leader for %s: %s", repopath, err)
			return
		}
		// Has the image been pushed?
		glog.V(1).Infof("Spawn id=%s node: %s:%s %s", id, node.Image.Repo, node.Image.Tag, node.Image.Tag)
		if node.PushedAt.Unix() == 0 {
			// Do I have the image?
			glog.Infof("Checking if push required for id=%s node: %s:%s %s (UUID=%s)", id, node.Image.Repo, node.Image.Tag, node.Image.Tag, node.Image.UUID)
			if img, err := l.FindImage(&node.Image); err == nil {
				glog.V(1).Infof("Found image %v locally, acquiring lead", node.Image)
				func() {
					// Become the leader so I can push the image
					leaderDone := make(chan struct{})
					defer close(leaderDone)
					_, err := leader.TakeLead(reponode, leaderDone)
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
						if node.PushedAt.Unix() > 0 {
							glog.V(1).Infof("Image %s already pushed, cancelling push", node.Image.String())
							return
						}
						if img.ID != node.Image.UUID {
							localHash, err := l.docker.GetImageHash(img.ID)
							if err != nil {
								glog.Warningf("Error building hash of image: %s, cancelling push: %s", img.ID, err)
								return
							} else if localHash != node.Image.Hash {
								glog.V(1).Infof("Image %s changed, cancelling push", node.Image.String())
								return
							}
						}
					}
					// Push the image and update the registry
					// If the push is unsuccessful, still update the timestamp,
					// so that the push will get retriggered the next time it
					// is needed.
					registrypath := path.Join(l.address, node.Image.String())
					glog.V(1).Infof("Updating registry image %s from path=%s", img.ID, registrypath)
					if err := l.docker.TagImage(img.ID, registrypath); err != nil {
						glog.Warningf("Could not tag %s as %s: %s", img.ID, registrypath, err)
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
		glog.V(1).Infof("Waiting for image %s to update (imagepath=%s)", node.Image, imagepath)
		select {
		case <-evt:
		case <-shutdown:
			return
		}

		close(done)
		done = make(chan struct{})
	}
}

// findImage looks for the registry image locally and in the local registry.  It firsts checks locally by UUID,
//  then by repo, tag, and hash, then it checks if the image is already in the registry, and finally it searches by hash
func (l *RegistryListener) FindImage(rImg *registry.Image) (*dockerclient.Image, error) {
	regaddr := path.Join(l.address, rImg.String())
	glog.V(1).Infof("Searching for image %s", regaddr)

	// check for UUID
	if img, err := l.docker.FindImage(rImg.UUID); err == nil {
		return img, nil
	}

	// check by repo and tag, and compare hashes
	glog.V(1).Infof("UUID %s not found locally, searching by registry address for %s", rImg.UUID, regaddr)
	if img, err := l.docker.FindImage(regaddr); err == nil {
		if localHash, err := l.docker.GetImageHash(img.ID); err != nil {
			glog.Warningf("Error building hash of image: %s: %s", img.ID, err)
		} else {
			if localHash == rImg.Hash {
				return img, nil
			}
			glog.V(1).Infof("Found %s locally, but hashes do not match", regaddr)
		}
	}

	// attempt to pull the image, then compare hashes
	glog.V(0).Infof("Image address %s not found locally, attempting pull", regaddr)
	if err := l.docker.PullImage(regaddr); err == nil {
		glog.V(1).Infof("Successfully pulled image %s from registry, checking for match", regaddr)
		if img, err := l.docker.FindImage(regaddr); err == nil {
			if img.ID == rImg.UUID {
				glog.V(1).Infof("Found image %s in registry with correct UUID", regaddr)
				return img, nil
			}
			if localHash, err := l.docker.GetImageHash(img.ID); err != nil {
				glog.Warningf("Error building hash of image: %s: %s", img.ID, err)
			} else {
				if localHash == rImg.Hash {
					glog.V(1).Infof("Found image %s in registry with correct Hash", regaddr)
					return img, nil
				}
			}
		}
	}

	// search all images for a matching hash
	// First just check top-level layers
	glog.V(0).Infof("Image %s not found in registry, searching local images by hash", regaddr)
	if img, err := l.docker.FindImageByHash(rImg.Hash, false); err == nil {
		return img, nil
	}

	// Now check all layers
	glog.V(0).Infof("Hash for Image %s not found in top-level layers, searching all layers", regaddr)
	return l.docker.FindImageByHash(rImg.Hash, true)
}
