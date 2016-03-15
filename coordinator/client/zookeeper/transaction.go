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

package zookeeper

import (
	"encoding/json"
	"path"

	zklib "github.com/control-center/go-zookeeper/zk"
	"github.com/control-center/serviced/coordinator/client"
	"github.com/zenoss/glog"
)

const (
	multiCreate int = iota
	multiSet
	multiDelete
)

type multiReq struct {
	Type int
	Path string
	Node client.Node
}

type Transaction struct {
	conn *Connection
	ops  []multiReq
}

func (t *Transaction) Create(path string, node client.Node) client.Transaction {
	t.ops = append(t.ops, multiReq{multiCreate, path, node})
	return t
}

func (t *Transaction) Set(path string, node client.Node) client.Transaction {
	t.ops = append(t.ops, multiReq{multiSet, path, node})
	return t
}

func (t *Transaction) Delete(path string) client.Transaction {
	t.ops = append(t.ops, multiReq{multiDelete, path, nil})
	return t
}

func (t *Transaction) Commit() error {
	t.conn.RLock()
	defer t.conn.RUnlock()
	if err := t.conn.isClosed(); err != nil {
		return err
	}
	var ops []interface{}
	for _, op := range t.ops {
		path := path.Join(t.conn.basePath, op.Path)
		data, err := json.Marshal(op.Node)
		if err != nil {
			glog.Errorf("Could not serialize node at path %s (%+v): %s", path, op.Node, err)
			return client.ErrSerialization
		}
		switch op.Type {
		case multiCreate:
			ops = append(ops, &zklib.CreateRequest{
				Path:  path,
				Data:  data,
				Acl:   zklib.WorldACL(zklib.PermAll),
				Flags: 0,
			})
			op.Node.SetVersion(&zklib.Stat{})
		case multiSet:
			stat := zklib.Stat{}
			if vers := op.Node.Version(); vers != nil {
				if zstat, ok := vers.(*zklib.Stat); !ok {
					glog.Errorf("Could not parse version of node at path %s (%+v): %s", path, op.Node, err)
					return client.ErrInvalidVersionObj
				} else {
					stat = *zstat
				}
			}
			ops = append(ops, &zklib.SetDataRequest{
				Path:    path,
				Data:    data,
				Version: stat.Version,
			})
		case multiDelete:
			_, stat, err := t.conn.conn.Get(path)
			if err != nil {
				glog.Errorf("Could not find path %s for delete: %s", path, err)
				return xlateError(err)
			}
			ops = append(ops, &zklib.DeleteRequest{
				Path:    path,
				Version: stat.Version,
			})
		}
	}
	_, err := t.conn.conn.Multi(ops...)
	return xlateError(err)
}
