// Copyright 2016 The Serviced Authors.
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

package web

import (
	"math/rand"
	"sync"
	"time"

	"github.com/control-center/serviced/zzk/registry2"
)

func init() {
	rand.Seed(time.Now().Unix())
}

// Exports manage a list of available exports
type Exports interface {
	Set(data []registry.ExportDetails)
	Next() *registry.ExportDetails
}

// RoundRobinExports returns the next export in a round-robin manner
type RoundRobinExports struct {
	mu   *sync.Mutex
	xid  int
	data []registry.ExportDetails
}

// NewRoundRobinExports creates a new round robin list of exports
func NewRoundRobinExports(data []registry.ExportDetails) *RoundRobinExports {
	e := &RoundRobinExports{
		mu: &sync.Mutex{},
	}
	e.set(data)
	return e
}

// Set updates the list of exports.
func (e *RoundRobinExports) Set(data []registry.ExportDetails) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.set(data)
}

// set updates the export list, but first randomizes the order and resets the
// counter.
func (e *RoundRobinExports) set(data []registry.ExportDetails) {

	// reset the counter
	e.xid = 0

	// randomize the exports
	e.data = make([]registry.ExportDetails, len(data))
	for i, j := range rand.Perm(len(data)) {
		e.data[i] = data[j]
	}
}

// Next returns the next available export
func (e *RoundRobinExports) Next() *registry.ExportDetails {
	e.mu.Lock()
	defer e.mu.Unlock()

	// make sure there is data to submit
	if size := len(e.data); size > 0 {
		dat := e.data[e.xid]
		e.xid = (e.xid + 1) % size
		return &dat
	}
	return nil
}
