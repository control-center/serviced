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

package utils

import "sync"

// ChannelCond provides functionality similar to sync.Cond, but is backed by
// channels instead of a mutex, allowing it to support timeouts and
// cancellation.
type ChannelCond struct {
	sync.RWMutex
	c chan struct{}
}

// Broadcast notifies all waiting goroutines that this condition has been
// satisfied
func (c *ChannelCond) Broadcast() {
	var old chan struct{}
	c.Lock()
	defer c.Unlock()
	c.c, old = make(chan struct{}), c.c
	close(old)
}

// Wait returns a channel that will close when the condition is satisfied.
func (c *ChannelCond) Wait() <-chan struct{} {
	c.RLock()
	defer c.RUnlock()
	return c.c
}

// NewChannelCond returns a new ChannelCond
func NewChannelCond() *ChannelCond {
	return &ChannelCond{c: make(chan struct{})}
}
