package storage

import (
	"fmt"
	"time"

	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/coordinator/client"
	"github.com/zenoss/serviced/domain/host"
)

type Client struct {
	host    *host.Host
	conn    client.Connection
	closing chan struct{}
}

func NewClient(host *host.Host, conn client.Connection) *Client {
	c := &Client{
		host:    host,
		conn:    conn,
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

		glog.Infof("creating %s", nodePath)
		if err = c.conn.Create(nodePath, node); err != nil && err != client.ErrNodeExists {
			continue
		}
		if err == client.ErrNodeExists {
			node2 := Node{}
			err = c.conn.Get(nodePath, &node2)
			if err != nil {
				continue
			}
			node.SetVersion(node2.Version())
			err = c.conn.Set(nodePath, node)
			if err != nil {
				continue
			}
		}
		e, err = c.conn.GetW(nodePath, node)
		if err != nil {
			continue
		}
		select {
		case <-c.closing:
			return
		case <-e:
			continue
		}
	}
}
