// Copyright 2014 The Serviced Authors.
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

/*
PriorityQueue implementation take from golang std library container/heap documentation example
*/
package scheduler

import (
	"github.com/control-center/serviced/domain/host"
)

// PriorityQueue implements the heap.Interface and holds hostitems
type PriorityQueue []*hostitem

// hostitem is what is stored in the least commited RAM scheduler's priority queue
type hostitem struct {
	host     *host.Host
	priority uint64 // the host's available RAM
	index    int    // the index of the hostitem in the heap
}

// Len is the number of elements in the collection.
func (pq PriorityQueue) Len() int {
	return len(pq)
}

// Less reports whether the element with index i should sort before the element with index j.
func (pq PriorityQueue) Less(i, j int) bool {
	return pq[i].priority > pq[j].priority
}

// Swap swaps the elements with indexes i and j.
func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

// Push pushes the hostitem onto the heap.
func (pq *PriorityQueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*hostitem)
	item.index = n
	*pq = append(*pq, item)
}

// Pop removes the minimum element (according to Less) from the heap and returns it.
func (pq *PriorityQueue) Pop() interface{} {
	opq := *pq
	n := len(opq)
	item := opq[n-1]
	item.index = -1 // mark it as removed, just in case
	*pq = opq[0 : n-1]
	return item
}
