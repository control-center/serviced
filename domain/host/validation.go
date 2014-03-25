// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package host

import (
	"github.com/zenoss/glog"

	"errors"
	"fmt"
	"net"
	"strings"
)

type ValidationError struct {
	Errs []error
}

func (v *ValidationError) Error() string {
	errString := "ValidationError: "
	for idx, err := range v.Errs {
		errString = fmt.Sprintf("%v\n   %v -  %v", errString, idx, err)
	}
	return errString
}

func (h *Host) validate() error {
	glog.Info("Validating host")

	vErrors := make([]error, 0)
	id := strings.TrimSpace(h.Id)
	if id == "" {
		vErrors = append(vErrors, errors.New("Empty Host.Id not allowed"))
		//		return errors.New("empty Host.Id not allowed")
	}

	if "" == strings.TrimSpace(h.PoolId) {
		vErrors = append(vErrors, errors.New("Empty Host.PoolId not allowed"))
	}

	if nil == net.ParseIP(h.IpAddr) {
		vErrors = append(vErrors, fmt.Errorf("Invalid IPAddr %s", h.IpAddr))
	}

	//TODO: what should we be validating here? It doesn't seem to work for
	glog.Infof("Validating IPAddr %v for host %s", h.IpAddr, id)
	ipAddr, err := net.ResolveIPAddr("ip4", h.IpAddr)

	if err != nil {
		glog.Errorf("Could not resolve: %s to an ip4 address: %v", h.IpAddr, err)
		vErrors = append(vErrors, err)
	} else if ipAddr.IP.IsLoopback() {
		glog.Errorf("Can not use %s as host address because it is a loopback address", h.IpAddr)
		vErrors = append(vErrors, errors.New("Host ip can not be a loopback address"))
	}

	if len(vErrors) > 0 {
		return &ValidationError{vErrors}
	}
	//Don't like this side effect
	h.Id = id
	return nil
}
