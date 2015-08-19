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
// limitations under the License.

package service

import (
	"encoding/json"
	"github.com/control-center/serviced/domain/servicedefinition"
	"github.com/zenoss/glog"
)

// private for dealing with unmarshal recursion
type serviceEndpoint ServiceEndpoint

// UnmarshalJSON implements the encoding/TextUnmarshaler interface
func (e *ServiceEndpoint) UnmarshalJSON(b []byte) error {
	se := serviceEndpoint{}
	if err := json.Unmarshal(b, &se); err == nil {
		*e = ServiceEndpoint(se)
	} else {
		return err
	}
	glog.V(4).Infof("ServiceEndpoint UnmarshalJSON %#v", e)
	if len(e.VHostList) > 0 {
		//VHostList is defined, keep it and unset deprecated field if set
		e.VHosts = nil
		return nil
	}
	if len(e.VHosts) > 0 {
		// no VHostsList but vhosts is defined. Convert to VHostsList
		glog.Warning("ServiceEndpoint VHosts field is deprecated, see VHostList")
		glog.V(0).Infof("VHosts is %#v", e.VHosts)
		for _, vhost := range e.VHosts {
			e.VHostList = append(e.VHostList, servicedefinition.VHost{Name: vhost, Enabled: true})
		}
		glog.V(0).Infof("VHostList %#v converted from VHosts %#v", e.VHostList, e.VHosts)
		e.VHosts = nil
	}
	return nil
}
