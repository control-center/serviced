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
	"github.com/zenoss/go-json-rest"
)

// getPools returns the list of pools requested.  This call supports paging.
func getPools(w *rest.ResponseWriter, r *rest.Request, ctx *requestContext) {
	facade := ctx.getFacade()
	dataCtx := ctx.getDatastoreContext()

	pools, err := facade.GetReadPools(dataCtx)
	if err != nil {
		restServerError(w, err)
		return
	}

	w.WriteJson(pools)
}
