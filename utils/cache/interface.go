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
package cache

// LRUCache defines a generic interface to an LRU cache.
type LRUCache interface {
	// Gets the value for key; returns the value and true if found, nil and false otherwise.
	Get(key string) (interface{}, bool)

	// Add a value to the cache for the specified key.
	// If the key already exists in the cache, it's value will be replaced.
	Set(key string, value interface{})

	// Returns the current number of items in the cache.
	GetCurrentSize() int

	// Returns the maximum size of the cache.
	GetMaxSize() int
}
