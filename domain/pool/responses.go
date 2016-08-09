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

// ResourcePoolsResponse contains the pools to return from a query of the datastore.  In the case of
// paging the resource pools in the results will be a subset of the all the pools that satisfied the
// query.
type ResourcePoolsResponse struct {
	Results []ResourcePool // The pools that satisfied the query (i.e. the page of pools)
	Total   int            // The total number pools that satisfy the query in the datastore
}
