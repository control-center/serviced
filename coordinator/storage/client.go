// Copyright 2014 The Serviced Authors.
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

package storage

import (
	"errors"
	"fmt"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/dfs/nfs"
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/utils"
	"github.com/control-center/serviced/zzk"
	"github.com/zenoss/glog"
)

type nfsMountT func(string, string) error

var nfsMount = nfs.Mount
var nfsUnmount = nfs.Unmount
var mkdirAll = os.MkdirAll
var storageClient *Client
var mp = utils.GetDefaultMountProc()

var ErrClientNotInitialized = errors.New("storage client not initialized")

// Client is a storage client that manges discovering and mounting filesystems
type Client struct {
	host         *host.Host
	exportedPath string
	localPath    string
	closing      chan struct{}
	mounted      chan chan<- string
	conn         client.Connection
	setLock      sync.Mutex
}

func GetClient() (*Client, error) {
	if storageClient == nil {
		return nil, ErrClientNotInitialized
	}
	return storageClient, nil
}

// NewClient returns a Client that manages remote mounts
func NewClient(host *host.Host, localPath string) (*Client, error) {
	if err := mkdirAll(localPath, 0755); err != nil {
		return nil, err
	}
	c := &Client{
		host:      host,
		localPath: localPath,
		mounted:   make(chan chan<- string),
		closing:   make(chan struct{}),
		// conn:      nil,   // commented out on purpose - no need to initialize
	}
	removeDeprecated("/serviced_var_volumes")
	go c.loop()
	return c, nil
}

func removeDeprecated(path string) {
	mounts, err := mp.ListAll()
	if err != nil {
		glog.Warningf("Could not get mounts: %s", err)
		return
	}
	for _, mount := range mounts {
		if strings.HasSuffix(mount.Device, ":"+path) {
			if err := mp.Unmount(mount.Device); err != nil {
				glog.Warningf("Could not unmount deprecated path %s (%s): %s", mount.Device, mount.MountPoint, err)
			}
		}
	}
}

// Wait will block until the client is Closed() or it has mounted the remote filesystem
func (c *Client) Wait() string {
	waitC := make(chan string, 1)
	var ch chan<- string = waitC

	select {
	case <-c.closing:
	case c.mounted <- ch:
	}

	select {
	case <-c.closing:
	case s := <-waitC:
		return s
	}

	return ""
}

// Close informs this client to shutdown its current operations.
func (c *Client) Close() {
	close(c.closing)
}

// Mount source  to local destination path. The source is relative to the exported path
func (c *Client) Mount(source, destination string) error {
	return nfsMount(&nfs.NFSDriver{}, path.Join(c.exportedPath, source), destination)
}

func (c *Client) Unmount(destination string) error {
	return nfsUnmount(&nfs.NFSDriver{}, destination)
}

func (c *Client) loop() {
	var err error
	var e <-chan client.Event
	node := &Node{
		Host:    *c.host,
		version: nil,
	}
	leaderNode := &Node{
		Host:    host.Host{},
		version: nil,
	}

	remoteShutdown := make(chan interface{})
	defer close(remoteShutdown)

	var doneC chan<- string
	var leader client.Leader
	nodePath := fmt.Sprintf("/storage/clients/%s", node.IPAddr)

	doneW := make(chan struct{})
	defer func(channel *chan struct{}) { close(*channel) }(&doneW)
	for {
		if doneC == nil {
			select {
			case doneC = <-c.mounted:
			case <-c.closing:
				return
			}
		}

		// keep from churning if we get errors
		if err != nil {
			select {
			case <-c.closing:
				return
			case <-time.After(time.Second * 10):
			}
		}
		err = nil
		if leader == nil {
			// /storage/leader needs to be at the root
			c.conn, err = zzk.GetLocalConnection("/")

			if err != nil {
				continue
			}
			leader, err = c.conn.NewLeader("/storage/leader")
			if err != nil {
				continue
			}
		}

		glog.Infof("creating %s", nodePath)
		if err = c.conn.Create(nodePath, node); err != nil && err != client.ErrNodeExists {
			glog.Errorf("could not create %s: %s", nodePath, err)
			continue
		}
		if err == client.ErrNodeExists {
			err = c.conn.Get(nodePath, node)
			if err != nil && err != client.ErrEmptyNode {
				glog.Errorf("could not get %s: %s", nodePath, err)
				continue
			}
		}
		node.Host = *c.host
		if err := c.setNode(nodePath, node, false); err != nil {
			glog.Errorf("problem updating %s: %s", nodePath, err)
			continue
		}

		e, err = c.conn.GetW(nodePath, node, doneW)
		if err != nil {
			glog.Errorf("err getting node %s: %s", nodePath, err)
			continue
		}
		if err = leader.Current(leaderNode); err != nil {
			glog.Errorf("err getting current leader: %s", err)
			continue
		}

		if leaderNode.IPAddr != c.host.IPAddr {
			glog.Infof("Check nfs supported")
			nfsd := &nfs.NFSDriver{}
			err = nfsd.Installed()
			if err != nil {
				if err == nfs.ErrNfsMountingUnsupported {
					glog.Errorf("Install the nfs-common package: %s", err)
				}
				glog.Errorf("Problem determining NFS available %s", err)
				continue
			}

		} else {
			glog.Info("skipping nfs mounting, server is localhost")
		}
		glog.Infof("At this point we know the leader is: %s", leaderNode.Host.IPAddr)
		select {
		case doneC <- leaderNode.ExportPath:
			c.exportedPath = leaderNode.ExportPath
			storageClient = c
			// notifying someone who cares
			doneC = nil
		case <-c.closing:
			remoteShutdown <- true
			return
		case evt := <-e:
			glog.Errorf("got zk event: %v", evt)
			continue
		}

		close(doneW)
		doneW = make(chan struct{})
	}
}

func (c *Client) setNode(nodePath string, node *Node, doGetBeforeSet bool) error {
	glog.V(4).Infof("waiting on lock for node %s: %+v", nodePath, node)
	c.setLock.Lock()
	defer c.setLock.Unlock()
	glog.V(4).Infof("got lock for node %s: %+v", nodePath, node)

	if doGetBeforeSet {
		err := c.conn.Get(nodePath, node)
		if err != nil && err != client.ErrEmptyNode {
			glog.Warningf("could not get %s: %s", nodePath, err)
			return err
		}
	}

	node.Host.UpdatedAt = time.Now()
	node.Host.ServiceD.Release = c.host.ServiceD.Release
	if err := c.conn.Set(nodePath, node); err != nil {
		return err
	}

	glog.V(4).Infof("updated node %s: %+v", nodePath, node)
	return nil
}
