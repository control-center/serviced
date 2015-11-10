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

package pool

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/control-center/serviced/commons/queue"
)

// Item wrapper type for items stored in pool
type Item struct {
	Item         interface{}
	checkedOut   bool
	checkoutTime time.Time
	id           uint64
}

// ErrItemUnavailable Error when an item cannot be allocated fromt the pool
var ErrItemUnavailable = errors.New("Pool Item not avaialable")

// Pool interface for pooling arbitrary objects
type Pool interface {
	// Borrow item from pool, returns ErrItemUnavailable if none available
	Borrow() (*Item, error)

	// Borrow item from pool, wait timeout. If timeout < 0 wait indefinitely. Returns ErrItemUnavailable if object not available
	BorrowWait(timeout time.Duration) (*Item, error)

	// Return item back to pool
	Return(item *Item) error

	// Remove removes the object from the pool
	Remove(item *Item) error

	//Returns the current number of items borrowed
	Borrowed() int

	//Returns the current number of items in the pool (not borrowed)
	Idle() int
}

// ItemFactory function to create an item for the pool
type ItemFactory func() (interface{}, error)

// NewPool create a pool with a capacity and factory for creating items.
func NewPool(capacity int, itemFactory ItemFactory) (Pool, error) {

	q, err := queue.NewChannelQueue(capacity)
	if err != nil {
		return nil, err
	}

	itemMap := make(map[uint64]*Item)
	pool := itemPool{itemMap: itemMap, itemQ: q, capacity: capacity, itemFactory: itemFactory}
	return &pool, nil
}

type itemPool struct {
	itemMap map[uint64]*Item
	itemQ   queue.Queue

	idCounter   uint64
	capacity    int
	itemFactory ItemFactory
	poolLock    sync.RWMutex
}

func (p *itemPool) BorrowWait(timeout time.Duration) (*Item, error) {
	var (
		item, newItem *Item
		err           error
		itemOrError   = true
	)

	// function for locking purposes
	func() {
		p.poolLock.Lock()
		defer p.poolLock.Unlock()
		qItem, found := p.itemQ.Poll()
		if found {
			item = p.checkout(qItem)
			return
		}
		if !found {
			newItem, err = p.newItem()
			if err != nil && err != ErrItemUnavailable {
				return
			} else if err == nil {
				item = p.checkout(newItem)
				return
			}
		}
		itemOrError = false
	}()

	if itemOrError {
		return item, err
	}
	var qItem interface{}
	itemChan, timeoutChan := p.itemQ.TakeChan(timeout)
	select {
	case qItem = <-itemChan:
		return p.checkout(qItem), nil
	case <-timeoutChan:
		return nil, ErrItemUnavailable
	}
}

func (p *itemPool) Borrow() (*Item, error) {
	return p.BorrowWait(0)
}

func (p *itemPool) Return(item *Item) error {
	err := func() error {
		p.poolLock.RLock()
		defer p.poolLock.RUnlock()
		if pooledItem, found := p.itemMap[item.id]; !found {
			return fmt.Errorf("Pool Return error, item not found")
		} else if pooledItem != item { //check same object (pointer compare)
			return fmt.Errorf("Pool Return error, item not in pool")
		} else if !item.checkedOut {
			return fmt.Errorf("Pool Return error, item not checked out")
		}
		item.checkedOut = false
		return nil
	}()
	if err != nil {
		return err
	}

	if !p.itemQ.Offer(item) {
		return p.Remove(item)
	}
	return nil
}

func (p *itemPool) Remove(item *Item) error {
	p.poolLock.Lock()
	defer p.poolLock.Unlock()

	if pooledItem, found := p.itemMap[item.id]; !found {
		return fmt.Errorf("Pool Remove error, item not found")
	} else if pooledItem != item { //check same object (pointer compare)
		return fmt.Errorf("Pool Remove error, item not in pool")
	} else if !item.checkedOut {
		return fmt.Errorf("Pool Remove error, item not checked out")
	}

	delete(p.itemMap, item.id)
	item, err := p.newItem()
	if err == nil {
		p.itemQ.Offer(item)
	} else {
		fmt.Printf("Remove error %v\n", err)
	}
	return nil
}

//Returns the current number of items borrowed
func (p *itemPool) Borrowed() int {
	p.poolLock.RLock()
	defer p.poolLock.RUnlock()
	count := 0
	for _, item := range p.itemMap {
		if item.checkedOut {
			count++
		}
	}
	return count
}

// Idle Returns the current number of items in the pool (not borrowed)
func (p *itemPool) Idle() int {
	p.poolLock.RLock()
	defer p.poolLock.RUnlock()
	count := 0
	for _, item := range p.itemMap {
		if !item.checkedOut {
			count++
		}
	}
	return count
}

func (p *itemPool) checkout(item interface{}) *Item {
	poolItem := item.(*Item)
	poolItem.checkedOut = true
	poolItem.checkoutTime = time.Now()
	return poolItem
}

// creates a new Item if it can
func (p *itemPool) newItem() (*Item, error) {
	if len(p.itemMap) >= p.capacity {
		return nil, ErrItemUnavailable
	}
	i, err := p.itemFactory()
	if err != nil {
		return nil, err
	}

	pItem := &Item{id: p.nextID(), Item: i}
	p.itemMap[pItem.id] = pItem
	return pItem, nil
}

func (p *itemPool) nextID() uint64 {
	return atomic.AddUint64(&p.idCounter, 1)
}
