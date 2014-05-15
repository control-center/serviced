package storage

import (
	"fmt"
	"time"

	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/coordinator/client"
	"github.com/zenoss/serviced/dfs/nfs"
	"github.com/zenoss/serviced/domain/host"
)

type nfsMountT func(string, string) error

var nfsMount nfsMountT = nfs.Mount

type Client struct {
	host    *host.Host
	zclient    *client.Client
	closing chan struct{}
}

func NewClient(host *host.Host, zclient *client.Client) *Client {
	c := &Client{
		host:    host,
		zclient: zclient,
		closing: make(chan struct{}),
	}
	go c.loop()
	return c
}

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
			conn, err = c.zclient.GetConnection()
			if err != nil {
				continue
			}
			leader = conn.NewLeader("/storage/leader", leaderNode)
		}

		glog.Infof("creating %s", nodePath)
		if err = conn.Create(nodePath, node); err != nil && err != client.ErrNodeExists {
			continue
		}
		if err == client.ErrNodeExists {
			err = conn.Get(nodePath, node)
			if err != nil {
				continue
			}
			node.Host = *c.host
			err = conn.Set(nodePath, node)
			if err != nil {
				continue
			}
		}
		e, err = conn.GetW(nodePath, node)
		if err != nil {
			continue
		}
		if err = leader.Current(leaderNode); err != nil {
			continue
		}

		err = nfsMount(leaderNode.ExportPath, "/opt/serviced/var")
		if err != nil {
			continue
		}
		glog.Infof("At this point we know the leader is: %s", leaderNode.Host.IPAddr)
		select {
		case <-c.closing:
			return
		case <-e:
			continue
		}
	}
}
