// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package host

import (
	"github.com/zenoss/glog"

	"errors"
	"net"
	"strings"
)

func (h *Host) validate() error {

	id := strings.TrimSpace(h.Id)
	if id == "" {
		return errors.New("empty Host.Id not allowed")
	}

	ipAddr, err := net.ResolveIPAddr("ip4", h.IpAddr)
	if err != nil {
		glog.Errorf("Could not resolve: %s to an ip4 address: %s", h.IpAddr, err)
		return err
	}
	if ipAddr.IP.IsLoopback() {
		glog.Errorf("Can not use %s as host address because it is a loopback address", h.IpAddr)
		return errors.New("host ip can not be a loopback address")
	}

	h.Id = id
	return nil
}
