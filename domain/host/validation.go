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

func (h *Host) ValidateEntity() error {
	glog.Info("Validating host")

	violations := validation.NewValidationError()
	violations.Add(validation.NotEmpty("Host.Id", h.Id))
	violations.Add(validation.NotEmpty("Host.PoolId", h.PoolId))
	violations.Add(validation.IsIP(h.IpAddr))

	//TODO: what should we be validating here? It doesn't seem to work for
	glog.Infof("Validating IPAddr %v for host %s", h.IpAddr, h.Id)
	ipAddr, err := net.ResolveIPAddr("ip4", h.IpAddr)

	if err != nil {
		glog.Errorf("Could not resolve: %s to an ip4 address: %v", h.IpAddr, err)
		violations.Add(err)
	} else if ipAddr.IP.IsLoopback() {
		glog.Errorf("Can not use %s as host address because it is a loopback address", h.IpAddr)
		violations.Add(errors.New("Host ip can not be a loopback address"))

	}

	if len(violations.Errors) > 0 {
		return violations
	}
	//Don't like this side effect
	h.Id = strings.TrimSpace(h.Id)
	return nil
}
