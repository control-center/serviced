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

// ValueChangePublisher allows for subscribers to be notified upon change of
// a value.
type ValueChangePublisher struct {
	value  interface{}
	mutex  sync.RWMutex
	notify chan struct{}
}

// Returns a new ValueChangePublisher
func NewValueChangePublisher(initialValue interface{}) ValueChangePublisher {
	return ValueChangePublisher{
		value:  initialValue,
		mutex:  sync.RWMutex{},
		notify: make(chan struct{}),
	}
}

// Set closes the current channel notifying current subscribers of a change
// and stores the value.
func (v *ValueChangePublisher) Set(value interface{}) {
	v.mutex.Lock()
	defer v.mutex.Unlock()
	v.value = value
	close(v.notify)
	v.notify = make(chan struct{})
}

// Get returns the current value of the publisher and a channel that will
// be closed when the value changes.
func (v *ValueChangePublisher) Get() (interface{}, <-chan struct{}) {
	v.mutex.RLock()
	defer v.mutex.RUnlock()
	return v.value, v.notify
}
