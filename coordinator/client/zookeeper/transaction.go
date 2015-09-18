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

func (t *Transaction) processCreate(op transactionOperation) (*zklib.CreateRequest, error) {
	path := join(t.conn.basePath, op.path)
	bytes, err := json.Marshal(op.node)
	if err != nil {
		return nil, client.ErrSerialization
	}
	req := &zklib.CreateRequest{
		Path:  path,
		Data:  bytes,
		Acl:   zklib.WorldACL(zklib.PermAll),
		Flags: 0,
	}
	return req, nil
}

func (t *Transaction) processSet(op transactionOperation) (*zklib.SetDataRequest, error) {
	path := join(t.conn.basePath, op.path)
	bytes, err := json.Marshal(op.node)
	if err != nil {
		return nil, client.ErrSerialization
	}
	stat := &zklib.Stat{}
	if op.node.Version() != nil {
		zstat, ok := op.node.Version().(*zklib.Stat)
		if !ok {
			return nil, client.ErrInvalidVersionObj
		}
		*stat = *zstat
	}
	req := &zklib.SetDataRequest{
		Path:    path,
		Data:    bytes,
		Version: stat.Version,
	}
	return req, nil
}

func (t *Transaction) processDelete(op transactionOperation) (*zklib.DeleteRequest, error) {
	path := join(t.conn.basePath, op.path)
	_, stat, err := t.conn.conn.Get(path)
	if err != nil {
		return nil, xlateError(err)
	}
	req := &zklib.DeleteRequest{
		Path:    path,
		Version: stat.Version,
	}
	return req, nil
}

func (t *Transaction) Commit() error {
	if t.conn == nil {
		return client.ErrConnectionClosed
	}
	zkCreate := []zklib.CreateRequest{}
	zkDelete := []zklib.DeleteRequest{}
	zkSetData := []zklib.SetDataRequest{}
	for _, op := range t.ops {
		switch op.op {
		case transactionCreate:
			req, err := t.processCreate(op)
			if err != nil {
				return err
			}
			zkCreate = append(zkCreate, *req)
		case transactionSet:
			req, err := t.processSet(op)
			if err != nil {
				return err
			}
			zkSetData = append(zkSetData, *req)
		case transactionDelete:
			req, err := t.processDelete(op)
			if err != nil {
				return err
			}
			zkDelete = append(zkDelete, *req)
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
