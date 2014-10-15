// Copyright 2014 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package registry

import (
	"fmt"
	"path"

	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/zzk"
	"github.com/zenoss/glog"
)

// KeySynchronizer synchronizes keys in the registry
type KeySynchronizer struct {
	conn          client.Connection
	registry      *registryType
	getConnection zzk.GetConnection
}

// Allocate implements zzk.SyncHandler
func (l *KeySynchronizer) Allocate() zzk.Node { return &KeyNode{} }

// GetConnection implements zzk.SyncHandler
func (l *KeySynchronizer) GetConnection(path string) (client.Connection, error) {
	return l.getConnection("/")
}

// GetPath implements zzk.SyncHandler
func (l *KeySynchronizer) GetPath(nodes ...string) string { return l.registry.getPath(nodes...) }

// Ready implements zzk.SyncHandler
func (l *KeySynchronizer) Ready() error { return nil }

// Done implements zzk.SyncHandler
func (l *KeySynchronizer) Done() {}

// GetAll implements zzk.SyncHandler
func (l *KeySynchronizer) GetAll() ([]zzk.Node, error) {
	children, err := l.conn.Children(l.registry.getPath())
	if err != nil {
		return nil, err
	}

	var nodes []zzk.Node
	for _, nodeID := range children {
		var node KeyNode
		if err := l.conn.Get(l.registry.getPath(nodeID), &node); err != nil {
			return nil, err
		} else if node.IsRemote {
			nodes = append(nodes, &node)
		}
	}

	return nodes, nil
}

func (l *KeySynchronizer) addkey(key string) error {
	var node KeyNode
	path := l.registry.getPath(key)
	if err := l.conn.Create(l.registry.getPath(key), &node); err != nil {
		return err
	}
	node.ID = key
	node.IsRemote = true
	return l.conn.Set(path, &node)
}

func (l *KeySynchronizer) updatekey(key string) error {
	var node KeyNode
	path := l.registry.getPath(key)
	if err := l.conn.Get(path, &node); err != nil {
		return err
	}

	if !node.IsRemote {
		if children, err := l.conn.Children(path); err != nil {
			return err
		} else if count := len(children); count > 0 {
			return fmt.Errorf("cannot update %s: found %d items", key, count)
		}
	}

	node.ID = key
	node.IsRemote = true
	return l.conn.Set(path, &node)
}

// AddUpdate implements zzk.SyncHandler
func (l *KeySynchronizer) AddUpdate(_ string, node zzk.Node) (string, error) {
	exists, err := zzk.PathExists(l.conn, l.registry.getPath(node.GetID()))

	if err != nil {
		return "", err
	}

	if exists {
		err = l.updatekey(node.GetID())
	} else {
		err = l.addkey(node.GetID())
	}

	if err != nil {
		return "", err
	}

	return node.GetID(), nil
}

// Delete implements zzk.SyncHandler
func (l *KeySynchronizer) Delete(id string) error {
	return l.conn.Delete(l.registry.getPath(id))
}

// EndpointSynchronizer synchronizes the EndpointRegistry
type EndpointSynchronizer struct {
	conn     client.Connection
	registry *EndpointRegistry
	key      string
}

// NewEndpointSynchronizer instiates a new synchronizer for the EndpointRegistry
func NewEndpointSynchronizer(local client.Connection, registry *EndpointRegistry, getRemoteConnection zzk.GetConnection) *zzk.Synchronizer {
	eSync := &KeySynchronizer{local, &registry.registryType, getRemoteConnection}
	sync := zzk.NewSynchronizer(eSync)

	sync.AddListener(func(key string) zzk.Listener {
		iSync := &EndpointSynchronizer{local, registry, key}
		return zzk.NewSynchronizer(iSync)
	})

	return sync
}

// Allocate implements zzk.SyncHandler
func (l *EndpointSynchronizer) Allocate() zzk.Node { return &EndpointNode{} }

// GetConnection implements zzk.SyncHandler
func (l *EndpointSynchronizer) GetConnection(path string) (client.Connection, error) { return nil, nil }

// GetPath implements zzk.SyncHandler
func (l *EndpointSynchronizer) GetPath(nodes ...string) string {
	return l.registry.getPath(append([]string{l.key}, nodes...)...)
}

// Ready implements zzk.SyncHandler
func (l *EndpointSynchronizer) Ready() error {
	children, err := l.conn.Children(l.GetPath())
	if err != nil {
		return err
	}

	for _, id := range children {
		if err := l.Delete(id); err != nil {
			return err
		}
	}

	return nil
}

// Done implements zzk.SyncHandler
func (l *EndpointSynchronizer) Done() {}

// GetAll implements zzk.SyncHandler
func (l *EndpointSynchronizer) GetAll() ([]zzk.Node, error) { return []zzk.Node{}, nil }

// AddUpdate implements zzk.SyncHandler
func (l *EndpointSynchronizer) AddUpdate(id string, node zzk.Node) (string, error) {
	var (
		ep  string
		err error
	)

	if endpoint, ok := node.(*EndpointNode); !ok {
		glog.Errorf("Could not extract endpoint node data for %s", id)
		return "", zzk.ErrInvalidType
	} else if id == "" {
		ep, err = l.registry.SetItem(l.conn, *endpoint)
	} else {
		ep, err = l.registry.setItem(l.conn, l.key, id, endpoint)
	}

	if err != nil {
		return "", err
	}

	return path.Base(ep), nil
}

// Delete implements zzk.SyncHandler
func (l *EndpointSynchronizer) Delete(id string) error {
	return l.registry.removeItem(l.conn, l.key, id)
}
