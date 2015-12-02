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

package registry

import (
	"time"

	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/zzk"
	"github.com/zenoss/glog"
)

type KeyNode struct {
	ID       string
	IsRemote bool
	version  interface{}
}

func (node *KeyNode) Version() interface{}                { return node.version }
func (node *KeyNode) SetVersion(version interface{})      { node.version = version }
func (node *KeyNode) GetID() string                       { return node.ID }
func (node *KeyNode) Create(conn client.Connection) error { return nil }
func (node *KeyNode) Update(conn client.Connection) error { return nil }

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
	exists, err := zzk.PathExists(conn, path)
	if err != nil {
		return "", err
	}

	if !exists {
		key := &KeyNode{ID: key}
		if err := conn.Create(path, key); err != nil {
			return "", err
		}
	}
	glog.Infof("EnsureKey returning path:%s", path)
	return path, nil
}

//WatchKey watches a key in the zk registry. Watches indefinitely or until cancelled, will block
func (r *registryType) WatchKey(conn client.Connection, key string, cancel <-chan interface{}, processChildren ProcessChildrenFunc, errorHandler WatchError) error {
	keyPath := r.getPath(key)
	if err := zzk.Ready(cancel, conn, keyPath); err != nil {
		glog.Errorf("Could not wait for registry key at %s: %s", keyPath, err)
		return err
	}
	return watch(conn, keyPath, cancel, processChildren, errorHandler)
}

//WatchRegistry watches the registry for new keys in the zk registry. Watches indefinitely or until cancelled, will block
func (r *registryType) WatchRegistry(conn client.Connection, cancel <-chan interface{}, processChildren ProcessChildrenFunc, errorHandler WatchError) error {
	path := r.getPath()
	if err := zzk.Ready(cancel, conn, path); err != nil {
		glog.Errorf("Could not wait for registry at %s: %s", r.getPath(), err)
		return err
	}
	return watch(conn, path, cancel, processChildren, errorHandler)
}

//Add node to the key in registry.  Returns the path of the node in the registry
func (r *registryType) addItem(conn client.Connection, key string, nodeID string, node client.Node) (string, error) {
	if err := r.ensureKey(conn, key); err != nil {
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
	if err := r.ensureKey(conn, key); err != nil {
		return "", err
	}

	//TODO: make ephemeral
	path := r.getPath(key, nodeID)

	exists, err := zzk.PathExists(conn, path)
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
		if addPath, err := r.addItem(conn, key, nodeID, node); err != nil {
			return "", err
		} else {
			path = addPath
		}
		glog.V(3).Infof("Add to %s: %#v", path, node)
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
	exists, err := zzk.PathExists(conn, path)
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

func (r *registryType) ensureKey(conn client.Connection, key string) error {
	node := &KeyNode{ID: key, IsRemote: false}
	timeout := time.After(time.Second * 60)
	var err error
	path := r.getPath(key)
	for {
		err = conn.Create(path, node)
		if err == client.ErrNodeExists || err == nil {
			return nil
		}
		select {
		case <-timeout:
			break
		default:
		}
	}
}

// getChildren gets all child paths for the given nodeID
func (r *registryType) getChildren(conn client.Connection, nodeID string) ([]string, error) {
	path := r.getPath(nodeID)
	glog.V(4).Infof("Getting children for %v", path)
	names, err := conn.Children(path)
	if err != nil {
		return []string{}, err
	}
	result := []string{}
	for _, name := range names {
		result = append(result, r.getPath(nodeID, name))
	}
	return result, nil
}

func WatchChildren(conn client.Connection, path string, cancel <-chan interface{}, processChildren ProcessChildrenFunc, errorHandler WatchError) error {
	return watch(conn, path, cancel, processChildren, errorHandler)
}

func watch(conn client.Connection, path string, cancel <-chan interface{}, processChildren ProcessChildrenFunc, errorHandler WatchError) error {
	exists, err := conn.Exists(path)
	if err != nil {
		return err
	} else if !exists {
		return client.ErrNoNode
	}

	done := make(chan struct{})
	defer func(channel *chan struct{}) { close(*channel) }(&done)
	for {
		glog.V(1).Infof("watching children at path: %s", path)
		nodeIDs, event, err := conn.ChildrenW(path, done)
		glog.V(1).Infof("child watch for path %s returned: %#v", path, nodeIDs)
		if err != nil {
			glog.Errorf("Could not watch %s: %s", path, err)
			defer errorHandler(path, err)
			return err
		}
		processChildren(conn, path, nodeIDs...)
		select {
		case ev := <-event:
			glog.V(1).Infof("watch event %+v at path: %s", ev, path)
		case <-cancel:
			glog.V(1).Infof("watch cancel at path: %s", path)
			return nil
		}

		close(done)
		done = make(chan struct{})
	}
}

func (r *registryType) watchItem(conn client.Connection, path string, nodeType client.Node, cancel <-chan interface{}, processNode func(conn client.Connection,
	node client.Node), errorHandler WatchError) error {
	exists, err := conn.Exists(path)
	if err != nil {
		return err
	} else if !exists {
		return client.ErrNoNode
	}

	done := make(chan struct{})
	defer func(channel *chan struct{}) { close(*channel) }(&done)
	for {
		event, err := conn.GetW(path, nodeType, done)
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

		close(done)
		done = make(chan struct{})
	}
}
