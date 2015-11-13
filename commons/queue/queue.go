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
// limitations under the License.package rpcutils

package queue

import (
	"fmt"
	"sync/atomic"
	"time"
)

// Queue is a bounded blocking queue
type Queue interface {
	// Take pops an item off the head of the queue, waits indefinitely
	Take() interface{}

	// TakeChan pops an item off the head of the queue, waits time.Duration. If duration <0 waits indefinitely
	// Return chan interface for item in quey, and error chan for timeout
	TakeChan(time.Duration) (<-chan interface{}, <-chan error)

	// Poll removes item off the head of the queue, returns immediately. Returns item, bool is true if item was in
	// deque, false otherwise
	Poll() (interface{}, bool)

	// Put puts item in deque, waits indefinitely
	Put(item interface{})

	// Offer puts the item in queue. If successful return true, if queue is full return false
	Offer(item interface{}) bool

	//Returns the current number of items in the queue
	Size() int32

	//Returns the current number of items the queue can hold
	Capacity() int32
}

// NewChannelQueue creates a queue with a given capacity. If capacity <=0, error is returned
func NewChannelQueue(capacity int) (Queue, error) {
	cap := int32(capacity)
	if capacity <= 0 {
		return nil, fmt.Errorf("Invalid size for queue: %d", capacity)
	}
	qChan := make(chan interface{}, cap)

	return &chanQueue{capacity: cap, qChan: qChan}, nil
}

type chanQueue struct {
	qChan    chan interface{}
	capacity int32
	size     int32
}

func (q *chanQueue) TakeChan(timeout time.Duration) (<-chan interface{}, <-chan error) {
	timeoutChan := make(chan error, 1)
	resultChan := make(chan interface{}, 1)
	go func() {
		if timeout < 0 {
			item := <-q.qChan
			atomic.AddInt32(&q.size, -1)
			resultChan <- item
		} else {
			select {
			case item := <-q.qChan:
				atomic.AddInt32(&q.size, -1)
				resultChan <- item
			case <-time.After(timeout):
				timeoutChan <- fmt.Errorf("Timeout waiting on queue item: %s", timeout)
			}
		}
	}()
	return resultChan, timeoutChan
}

func (q *chanQueue) Take() interface{} {
	itemChan, _ := q.TakeChan(0)
	select {
	case item := <-itemChan:
		return item
	}
}
func (q *chanQueue) Poll() (interface{}, bool) {
	select {
	case item := <-q.qChan:
		atomic.AddInt32(&q.size, -1)
		return item, true
	default:
		return nil, false
	}
}

func (q *chanQueue) Put(item interface{}) {
	q.qChan <- item
	atomic.AddInt32(&q.size, 1)
}
func (q *chanQueue) Offer(item interface{}) bool {
	select {
	case q.qChan <- item:
		atomic.AddInt32(&q.size, 1)
		return true
	default:
		return false
	}
}
func (q *chanQueue) Capacity() int32 {
	return q.capacity
}

func (q *chanQueue) Size() int32 {
	return q.size
}
