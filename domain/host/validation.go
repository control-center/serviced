// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package host

import (
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/validation"

	"errors"
	"net"
	"strings"
)

//ValidEntity validates Host fields
func (h *Host) ValidEntity() error {
	glog.Info("Validating host")

	trimmedID := strings.TrimSpace(h.ID)
	violations := validation.NewValidationError()
	violations.Add(validation.NotEmpty("Host.ID", h.ID))
	violations.Add(validation.StringsEqual(h.ID, trimmedID, "leading and trailing spaces not allowed for host id"))
	violations.Add(validation.NotEmpty("Host.PoolID", h.PoolID))
	violations.Add(validation.IsIP(h.IPAddr))

	//TODO: what should we be validating here? It doesn't seem to work for
	glog.Infof("Validating IPAddr %v for host %s", h.IPAddr, h.ID)
	ipAddr, err := net.ResolveIPAddr("ip4", h.IPAddr)

	if err != nil {
		glog.Errorf("Could not resolve: %s to an ip4 address: %v", h.IPAddr, err)
		violations.Add(err)
	} else if ipAddr.IP.IsLoopback() {
		glog.Errorf("Can not use %s as host address because it is a loopback address", h.IPAddr)
		violations.Add(errors.New("host ip can not be a loopback address"))

	}

	if len(violations.Errors) > 0 {
		return violations
	}
	return nil
}
