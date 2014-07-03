// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package registry

import (
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/coordinator/client"
	"github.com/zenoss/serviced/zzk/utils"
)

type registryType struct {
	getPath   func(nodes ...string) string
	ephemeral bool
}

// WatchError is called by Watch* functions when there are errors
type WatchError func(path string, err error)

// ProcessChildrenFunc is called by Watch* functions when node addition/deletion occurs
type ProcessChildrenFunc func(conn client.Connection, parentPath string, nodeIDs ...string)

//SetEphemeral sets the ephemeral flag
func (r *registryType) SetEphemeral(useEphemeral bool) {
	r.ephemeral = useEphemeral
}

//IsEphemeral gets the ephemeral flag
func (r *registryType) IsEphemeral() bool {
	return r.ephemeral
}

//EnsureKey ensures key path to the registry.  Returns the path of the key in the registry
func (r *registryType) EnsureKey(conn client.Connection, key string) (string, error) {

	path := r.getPath(key)
	glog.Infof("EnsureKey key:%s path:%s", key, path)
	exists, err := utils.PathExists(conn, path)
	if err != nil {
		return "", err
	}

	if !exists {
		if err := conn.CreateDir(path); err != nil {
			return "", err
		}
	}
	glog.Infof("EnsureKey returning path:%s", path)
	return path, nil
}

//WatchKey watches a key in the zk registry. Watches indefinitely or until cancelled, will block
func (r *registryType) WatchKey(conn client.Connection, key string, cancel <-chan bool, processChildren ProcessChildrenFunc, errorHandler WatchError) error {
	keyPath := r.getPath(key)
	return watch(conn, keyPath, cancel, processChildren, errorHandler)
}

//WatchRegistry watches the registry for new keys in the zk registry. Watches indefinitely or until cancelled, will block
func (r *registryType) WatchRegistry(conn client.Connection, cancel <-chan bool, processChildren ProcessChildrenFunc, errorHandler WatchError) error {
	path := r.getPath()
	return watch(conn, path, cancel, processChildren, errorHandler)
}

//Add node to the key in registry.  Returns the path of the node in the registry
func (r *registryType) addItem(conn client.Connection, key string, nodeID string, node client.Node) (string, error) {
	if err := r.ensureDir(conn, r.getPath(key)); err != nil {
		glog.Errorf("error with addItem.ensureDir(%s) %+v", r.getPath(key), err)
		return "", err
	}

	//TODO: make ephemeral
	path := r.getPath(key, nodeID)
	glog.V(3).Infof("Adding to %s: %#v", path, node)
	if r.ephemeral {
		var err error
		if path, err = conn.CreateEphemeral(path, node); err != nil {
			glog.Errorf("error with addItem.CreateEphemeral(%s) %+v", path, err)
			return "", err
		}
	} else {
		if err := conn.Create(path, node); err != nil {
			glog.Errorf("error with addItem.Create(%s) %+v", path, err)
			return "", err
		}
	}
	return path, nil
}

//Set node to the key in registry.  Returns the path of the node in the registry
func (r *registryType) setItem(conn client.Connection, key string, nodeID string, node client.Node) (string, error) {
	if err := r.ensureDir(conn, r.getPath(key)); err != nil {
		return "", err
	}

	//TODO: make ephemeral
	path := r.getPath(key, nodeID)

	exists, err := utils.PathExists(conn, path)
	if err != nil {
		return "", err
	}

	if exists {
		glog.V(3).Infof("Set to %s: %#v", path, node)
		epn := EndpointNode{}
		if err := conn.Get(path, &epn); err != nil {
			return "", err
		}
		node.SetVersion(epn.Version())
		if err := conn.Set(path, node); err != nil {
			return "", err
		}
	} else {
		glog.V(3).Infof("Add to %s: %#v", path, node)
		if _, err := r.addItem(conn, key, nodeID, node); err != nil {
			return "", err
		}
	}
	return path, nil
}

func (r *registryType) removeKey(conn client.Connection, key string) error {
	path := r.getPath(key)
	return removeNode(conn, path)
}

func (r *registryType) removeItem(conn client.Connection, key string, nodeID string) error {
	path := r.getPath(key, nodeID)
	return removeNode(conn, path)
}

func removeNode(conn client.Connection, path string) error {
	exists, err := utils.PathExists(conn, path)
	if err != nil {
		return err
	}

	if !exists {
		return nil
	}

	if err := conn.Delete(path); err != nil {
		glog.Errorf("Unable to delete path:%s error:%v", path, err)
		return err
	}

	return nil
}

func (r *registryType) ensureDir(conn client.Connection, path string) error {
	if exists, err := utils.PathExists(conn, path); err != nil {
		return err
	} else if !exists {
		glog.V(0).Infof("creating zk dir %s", path)
		if err := conn.CreateDir(path); err != nil {
			glog.Errorf("error with ensureDir.CreateDir(%s) %+v", path, err)
			return err
		}
	}
	return nil
}

func watch(conn client.Connection, path string, cancel <-chan bool, processChildren ProcessChildrenFunc, errorHandler WatchError) error {
	exists, err := utils.PathExists(conn, path)
	if err != nil {
		return err
	}
	if !exists {
		return client.ErrNoNode
	}
	for {
		glog.V(0).Infof("watching children at path: %s", path)
		nodeIDs, event, err := conn.ChildrenW(path)
		glog.V(0).Infof("child watch for path %s returned: %#v", path, nodeIDs)
		if err != nil {
			glog.Errorf("Could not watch %s: %s", path, err)
			defer errorHandler(path, err)
			return err
		}
		processChildren(conn, path, nodeIDs...)
		//This blocks until a change happens under the key
		select {
		case ev := <-event:
			glog.V(0).Infof("watch event %+v at path: %s", ev, path)
		case <-cancel:
			glog.V(0).Infof("watch cancel at path: %s", path)
			return nil
		}
	}
	glog.V(0).Infof("no longer watching children at path: %s", path)
	return nil
}

func (r *registryType) watchItem(conn client.Connection, path string, nodeType client.Node, cancel <-chan bool, processNode func(conn client.Connection,
	node client.Node), errorHandler WatchError) error {
	exists, err := utils.PathExists(conn, path)
	if err != nil {
		return err
	}
	if !exists {
		return client.ErrNoNode
	}
	for {
		event, err := conn.GetW(path, nodeType)
		if err != nil {
			glog.Errorf("Could not watch %s: %s", path, err)
			defer errorHandler(path, err)
			return err
		}
		processNode(conn, nodeType)
		//This blocks until a change happens under the key
		select {
		case ev := <-event:
			glog.V(2).Infof("watch event %+v at path: %s", ev, path)
		case <-cancel:
			return nil
		}

	}
	return nil
}
