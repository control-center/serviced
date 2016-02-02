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

	"github.com/control-center/serviced/domain"
	"github.com/control-center/serviced/domain/servicedefinition"
	"github.com/zenoss/glog"
)

type service Service

func (s *Service) UnmarshalJSON(b []byte) error {
	svc := service{}
	if err := json.Unmarshal(b, &svc); err == nil {
		*s = Service(svc)
	} else {
		return err
	}
	if len(s.Commands) > 0 {
		s.Runs = nil
		return nil
	}
	if len(s.Runs) > 0 {
		s.Commands = make(map[string]domain.Command)
		for k, v := range s.Runs {
			s.Commands[k] = domain.Command{
				Command:         v,
				CommitOnSuccess: true,
			}
		}
		s.Runs = nil
	}
	return nil
}

// private for dealing with unmarshal recursion
type serviceEndpoint ServiceEndpoint

// UnmarshalJSON implements the encoding/json/Unmarshaler interface used to convert deprecated vhosts list to VHostList
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
		if glog.V(2) {
			glog.Warningf("ServiceEndpoint VHosts field is deprecated, see VHostList: %#v", e.VHosts)
		}
		for _, vhost := range e.VHosts {
			e.VHostList = append(e.VHostList, servicedefinition.VHost{Name: vhost, Enabled: true})
		}
		glog.V(2).Infof("VHostList %#v converted from VHosts %#v", e.VHostList, e.VHosts)
		e.VHosts = nil
	}
	return nil
}
