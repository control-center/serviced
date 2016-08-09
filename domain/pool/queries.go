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

package pool

// ResourcePoolsQuery is used to get resource pools from the pools store.  ResourcePoolsQuery
// is used for paging and sorting.
type ResourcePoolsQuery struct {
	Skip  int    // The number of records to skip from the beginning
	Pull  int    // The number of records to pull
	Sort  string // The field to sort by
	Order string // The order to sort by, i.e. asc or desc
}
