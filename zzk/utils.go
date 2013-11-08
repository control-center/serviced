package zzk

import (
	"github.com/samuel/go-zookeeper/zk"
	"github.com/zenoss/glog"

	"bytes"
	"time"
)

const SERVICE_PATH = "/services"
const SCHEDULER_PATH = "/scheduler"

func TimeoutAfter(delay time.Duration) <-chan bool {
	doneChan := make(chan bool)
	go func(t time.Duration, done chan bool) {
		time.Sleep(t)
		done <- true
	}(delay, doneChan)
	return doneChan
}

func DeleteNodebyData(path string, conn *zk.Conn, data []byte) error {

	children, _, err := conn.Children(path)
	if err != nil {
		glog.Warning("Could not list children")
		return err
	}
	for _, child := range children {
		child_path := path + "/" + child
		child_data, _, err := conn.Get(child_path)
		if err != nil {
			glog.Warning("Could not get data for %s", child_path)
			continue
		}
		if bytes.Compare(data, child_data) == 0 {
			for i := 0; i < 5; i++ {
				_, stats, _ := conn.Get(child_path)
				err = conn.Delete(child_path, stats.Version)
				if err == nil || err == zk.ErrNoNode {
					return nil
				}
			}
			glog.Error("Could not delete matched node %s", child_path)
		}
	}
	return nil
}

func CreateNode(path string, conn *zk.Conn) error {
	var err error
	for i := 0; i < 5; i++ {
		_, err = conn.Create(path, []byte{}, 0, zk.WorldACL(zk.PermAll))
		if err == zk.ErrNodeExists || err == nil {
			return nil
		}
		glog.Warningf("Could not create node:%s, %v", path, err)
	}
	return err
}
