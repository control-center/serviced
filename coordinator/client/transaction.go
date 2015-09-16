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

package client

const (
	TransactionCreate = iota
	TransactionSet    = iota
	TransactionDelete = iota
)

type TransactionOperation struct {
	Op   int
	Path string
	Node Node
}

type Transaction struct {
	Conn Connection
	Ops  []TransactionOperation
}

func (t *Transaction) Create(path string, node Node) {
	t.Ops = append(t.Ops, TransactionOperation{
		Op:   TransactionCreate,
		Path: path,
		Node: node,
	})
}

func (t *Transaction) Set(path string, node Node) {
	t.Ops = append(t.Ops, TransactionOperation{
		Op:   TransactionSet,
		Path: path,
		Node: node,
	})
}

func (t *Transaction) Delete(path string) {
	t.Ops = append(t.Ops, TransactionOperation{
		Op:   TransactionDelete,
		Path: path,
	})
}

func (t *Transaction) Commit() error {
	return t.Conn.Transact(*t)
}
