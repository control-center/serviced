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

package web

import (
	"github.com/zenoss/glog"
	"github.com/zenoss/go-json-rest"
	"github.com/control-center/serviced/domain/pool"

	"net/url"
)

// restAddPoolVirtualIP takes a poolID, IP, netmask, and bindinterface and adds it
func restAddPoolVirtualIP(w *rest.ResponseWriter, r *rest.Request, ctx *requestContext) {
	//TODO replace virtualiprequest with model object
	var request pool.VirtualIP
	err := r.DecodeJsonPayload(&request)
	if err != nil {
		restBadRequest(w, err)
		return
	}

	glog.V(0).Infof("Add virtual ip: %+v", request)

	client, err := ctx.getMasterClient()
	if err != nil {
		restServerError(w, err)
		return
	}

	if err := client.AddVirtualIP(request); err != nil {
		glog.Errorf("Failed to add virtual IP(%+v): %v", request, err)
		restServerError(w, err)
		return
	}

	restSuccess(w)
}

// restRemovePoolVirtualIP deletes virtualip
func restRemovePoolVirtualIP(w *rest.ResponseWriter, r *rest.Request, ctx *requestContext) {
	ip, err := url.QueryUnescape(r.PathParam("ip"))
	if err != nil {
		glog.Errorf("Could not get virtual ip (%v): %v", ip, err)
		restBadRequest(w, err)
		return
	}

	poolId, err := url.QueryUnescape(r.PathParam("poolId"))
	if err != nil {
		glog.Errorf("Could not get virtual ip poolId (%v): %v", poolId, err)
		restBadRequest(w, err)
		return
	}

	glog.V(0).Infof("Remove virtual ip=%v (in pool %v)", ip, poolId)

	client, err := ctx.getMasterClient()
	if err != nil {
		restServerError(w, err)
		return
	}

	request := pool.VirtualIP{PoolID: poolId, IP: ip}
	if err := client.RemoveVirtualIP(request); err != nil {
		glog.Errorf("Failed to remove virtual IP(%+v): %v", request, err)
		restServerError(w, err)
		return
	}
	restSuccess(w)
}
