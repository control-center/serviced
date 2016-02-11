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

package host

import (
	"fmt"

	"github.com/control-center/serviced/validation"
	"github.com/zenoss/glog"

	"errors"
	"net"
	"strings"
)

//ValidEntity validates Host fields
func (h *Host) ValidEntity() error {
	glog.V(4).Info("Validating host")

	//if err := validation.ValidHostID(entity.ID); err != nil {
	//	return fmt.Errorf("invalid hostid:'%s' for host Name:'%s' IP:%s", entity.ID, entity.Name, entity.IPAddr)
	//}

	trimmedID := strings.TrimSpace(h.ID)
	violations := validation.NewValidationError()
	violations.Add(validation.NotEmpty("Host.ID", h.ID))
	violations.Add(validation.ValidHostID(h.ID))
	violations.Add(validation.StringsEqual(h.ID, trimmedID, "leading and trailing spaces not allowed for host id"))
	violations.Add(validation.ValidPort(h.RPCPort))
	violations.Add(validation.NotEmpty("Host.PoolID", h.PoolID))
	violations.Add(validation.IsIP(h.IPAddr))

	//TODO: what should we be validating here? It doesn't seem to work for
	glog.V(4).Infof("Validating IPAddr %v for host %s", h.IPAddr, h.ID)
	ipAddr, err := net.ResolveIPAddr("ip4", h.IPAddr)

	if err != nil {
		glog.Errorf("Could not resolve: %s to an ip4 address: %v", h.IPAddr, err)
		violations.Add(err)
	} else if ipAddr.IP.IsLoopback() {
		glog.Errorf("Can not use %s as host address because it is a loopback address", h.IPAddr)
		violations.Add(errors.New("host ip can not be a loopback address"))

	}
	if _, err := GetRAMLimit(h.RAMLimit, h.Memory); err == ErrSizeTooBig {
		h.RAMLimit = fmt.Sprintf("%d", h.Memory)
	} else if err != nil {
		glog.Errorf("Can not parse RAM Limit: %s", err)
		violations.Add(err)
	}
	if len(violations.Errors) > 0 {
		return violations
	}
	return nil
}
