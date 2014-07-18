// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package storage

import (
	"fmt"
	"os"
	"time"

	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/coordinator/client"
	"github.com/zenoss/serviced/dfs/nfs"
	"github.com/zenoss/serviced/domain/host"
	"github.com/zenoss/serviced/zzk"
)

type nfsMountT func(string, string) error

var nfsMount = nfs.Mount
var mkdirAll = os.MkdirAll

// Client is a storage client that manges discovering and mounting filesystems
type Client struct {
	host      *host.Host
	localPath string
	closing   chan struct{}
	mounted   chan string
}

// NewClient returns a Client that manages remote mounts
func NewClient(host *host.Host, localPath string) (*Client, error) {
	if err := mkdirAll(localPath, 0755); err != nil {
		return nil, err
	}
	c := &Client{
		host:      host,
		localPath: localPath,
		mounted:   make(chan string, 1),
		closing:   make(chan struct{}),
	}
	go c.loop()
	return c, nil
}

// Wait will block until the client is Closed() or it has mounted the remote filesystem
func (c *Client) Wait() string {
	select {
	case <-c.closing:
	case s := <-c.mounted:
		return s
	}
	return ""
}

// Close informs this client to shutdown its current operations.
func (c *Client) Close() {
	close(c.closing)
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
	var leader client.Leader
	var conn client.Connection
	nodePath := fmt.Sprintf("/storage/clients/%s", node.IPAddr)
	for {
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
			conn, err = zzk.GetBasePathConnection(zzk.GeneratePoolPath(c.host.PoolID))
			if err != nil {
				continue
			}
			leader = conn.NewLeader("/storage/leader", leaderNode)
		}

		glog.Infof("creating %s", nodePath)
		if err = conn.Create(nodePath, node); err != nil && err != client.ErrNodeExists {
			glog.Errorf("could not create %s: %s", nodePath, err)
			continue
		}
		if err == client.ErrNodeExists {
			err = conn.Get(nodePath, node)
			if err != nil && err != client.ErrEmptyNode {
				glog.Errorf("could not get %s: %s", nodePath, err)
				continue
			}
		}
		node.Host = *c.host
		if err := conn.Set(nodePath, node); err != nil {
			glog.Errorf("problem updating %s: %s", nodePath, err)
			continue
		}

		e, err = conn.GetW(nodePath, node)
		if err != nil {
			glog.Errorf("err getting node %s: %s", nodePath, err)
			continue
		}
		if err = leader.Current(leaderNode); err != nil {
			glog.Errorf("err getting current leader: %s", err)
			continue
		}

		if leaderNode.IPAddr != c.host.IPAddr {
			err = nfsMount(leaderNode.ExportPath, c.localPath)
			if err != nil {
				if err == nfs.ErrNfsMountingUnsupported {
					glog.Errorf("install the nfs-common package: %s", err)
				}
				glog.Errorf("problem mouting %s: %s", leaderNode.ExportPath, err)
				continue
			}
		} else {
			glog.Info("skipping nfs mounting, server is localhost")
		}
		glog.Infof("At this point we know the leader is: %s", leaderNode.Host.IPAddr)
		select {
		case c.mounted <- leaderNode.ExportPath:
			// notifying someone who cares
		case <-c.closing:
			return
		case evt := <-e:
			glog.Errorf("got zk event: %s", evt)
			continue
		}
	}
}
