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
	proxymux "github.com/control-center/serviced/proxy"
	"github.com/control-center/serviced/utils"
	"github.com/zenoss/glog"
	"github.com/zenoss/go-json-rest"

	"sort"
	"time"
)

//restGetMuxConnections gets mux connection info from all hosts. Response is map[host-id][src-dst]MuxConnectionInfo
func restGetMuxConnections(w *rest.ResponseWriter, r *rest.Request, ctx *requestContext) {
	response := make(map[string]map[string]proxymux.TCPMuxConnectionInfo)
	client, err := ctx.getMasterClient()
	if err != nil {
		restServerError(w, err)
		return
	}

	hosts, err := client.GetHosts()
	if err != nil {
		glog.Errorf("Could not get hosts: %v", err)
		restServerError(w, err)
		return
	}

	activeHostIDs, err := client.GetActiveHostIDs()
	if err != nil {
		glog.Errorf("Could not get active hosts: %v", err)
		restServerError(w, err)
		return
	}

	myHostID, err := utils.HostID()
	if err != nil {
		glog.Errorf("Could not get hostid: %v", err)
		restServerError(w, err)
		return
	}

	activeHostIDsMap := make(map[string]string)
	for _, hostid := range activeHostIDs {
		activeHostIDsMap[hostid] = hostid
	}

	glog.V(3).Infof("Returning mux connections for %d active hosts", len(activeHostIDs))
	for i, host := range hosts {
		if _, ok := activeHostIDsMap[host.ID]; !ok {
			continue
		}

		if myHostID != host.ID {
			glog.V(3).Infof("TODO: get mux connections for remote hosts: %+v", &hosts[i])
			continue
		}

		response[host.ID] = proxymux.GetMuxConnectionInfo()
		glog.Infof("mux connections (count:%d) for host: %s %s", len(response[host.ID]), host.ID, host.IPAddr)

		sortedkeys := make([]string, len(response[host.ID]))
		i := 0
		for k, _ := range response[host.ID] {
			sortedkeys[i] = k
			i++
		}
		sort.Strings(sortedkeys)
		for _, k := range sortedkeys {
			cxn := response[host.ID][k]
			glog.Infof("  %s  (%s/%d <-- %s/%s) (%s <-- %s) %s", k,
				cxn.ApplicationEndpoint.Application, cxn.ApplicationEndpoint.InstanceID,
				cxn.Src.ServiceName, cxn.Src.InstanceID,
				cxn.ApplicationEndpoint.ServiceID, cxn.Src.ServiceID, time.Since(cxn.CreatedAt))
		}
	}

	w.WriteJson(&response)
}
