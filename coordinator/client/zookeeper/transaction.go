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

	zklib "github.com/control-center/go-zookeeper/zk"
	"github.com/control-center/serviced/coordinator/client"
)

const (
	transactionCreate = iota
	transactionSet    = iota
	transactionDelete = iota
)

type transactionOperation struct {
	op   int
	path string
	node client.Node
}

type Transaction struct {
	conn *Connection
	ops  []transactionOperation
}

func (t *Transaction) Create(path string, node client.Node) {
	t.ops = append(t.ops, transactionOperation{
		op:   transactionCreate,
		path: path,
		node: node,
	})
}

func (t *Transaction) Set(path string, node client.Node) {
	t.ops = append(t.ops, transactionOperation{
		op:   transactionSet,
		path: path,
		node: node,
	})
}

func (t *Transaction) Delete(path string) {
	t.ops = append(t.ops, transactionOperation{
		op:   transactionDelete,
		path: path,
	})
}

func (t *Transaction) Commit() error {
	if t.conn == nil {
		return client.ErrConnectionClosed
	}
	zkCreate := []zklib.CreateRequest{}
	zkDelete := []zklib.DeleteRequest{}
	zkSetData := []zklib.SetDataRequest{}
	for _, op := range t.ops {
		path := join(t.conn.basePath, op.path)
		switch op.op {
		case transactionCreate:
			bytes, err := json.Marshal(op.node)
			if err != nil {
				return client.ErrSerialization
			}
			zkCreate = append(zkCreate, zklib.CreateRequest{
				Path:  path,
				Data:  bytes,
				Acl:   zklib.WorldACL(zklib.PermAll),
				Flags: 0,
			})
		case transactionSet:
			bytes, err := json.Marshal(op.node)
			if err != nil {
				return client.ErrSerialization
			}
			stat := &zklib.Stat{}
			if op.node.Version() != nil {
				zstat, ok := op.node.Version().(*zklib.Stat)
				if !ok {
					return client.ErrInvalidVersionObj
				}
				*stat = *zstat
			}
			zkSetData = append(zkSetData, zklib.SetDataRequest{
				Path:    path,
				Data:    bytes,
				Version: stat.Version,
			})
		case transactionDelete:
			path := join(t.conn.basePath, op.path)
			_, stat, err := t.conn.conn.Get(path)
			if err != nil {
				return xlateError(err)
			}
			zkDelete = append(zkDelete, zklib.DeleteRequest{
				Path:    path,
				Version: stat.Version,
			})
		}
	}
	multi := zklib.MultiOps{
		Create:  zkCreate,
		SetData: zkSetData,
		Delete:  zkDelete,
	}
	if err := t.conn.conn.Multi(multi); err != nil {
		return xlateError(err)
	}
	// I honestly have no idea why we're doing this, but we were
	// doing it in the original Create function, so I replicate that
	// behavior here. -RT
	for _, op := range t.ops {
		if op.op == transactionCreate {
			op.node.SetVersion(&zklib.Stat{})
		}
	}
	return xlateError(nil)
}
