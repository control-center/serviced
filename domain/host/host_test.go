// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package host

import (
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced"

	"fmt"
	"strings"
	"testing"
)

func containsAll(ve *ValidationError, errStrings ...string) bool {
	for _, err := range errStrings {
		if !contains(ve, err) {
			return false
		}
	}
	return true
}
func contains(ve *ValidationError, errString string) bool {

	for _, err := range ve.Errs {
		if err.Error() == errString {
			return true
		}
	}
	return false
}

type validateCase struct {
	id             string
	poolid         string
	ip             string
	expectedErrors []string
}

var validatetable []validateCase

func init() {

	ip, err := serviced.GetIPAddress()
	if err != nil {
		glog.Errorf("Could not determine ip %v", err)
	}

	validatetable = []validateCase{
		validateCase{"", "", "", []string{"Empty Host.Id not allowed", "Empty Host.PoolId not allowed", "Invalid IPAddr "}},
		validateCase{"hostid", "", "", []string{"Empty Host.PoolId not allowed", "Invalid IPAddr "}},
		validateCase{"", "poolid", "", []string{"Empty Host.Id not allowed", "Invalid IPAddr "}},
		validateCase{"hostid", "poolid", "", []string{"Invalid IPAddr "}},
		validateCase{"hostid", "poolid", "blam", []string{"Invalid IPAddr blam"}},
		validateCase{"hostid", "poolid", "127.0.0.1", []string{"Host ip can not be a loopback address"}},
		validateCase{"hostid", "poolid", ip, []string{}},
	}

}

func Test_ValidateTable(t *testing.T) {
	for idx, test := range validatetable {
		glog.Infof("test for  %v", test)
		h := New()
		h.Id = test.id
		h.PoolId = test.poolid
		h.IpAddr = test.ip

		err := h.validate()
		if len(test.expectedErrors) > 0 {
			if verr, isVErr := err.(*ValidationError); !isVErr {
				t.Errorf("expected ValidationError, got %v", err)
			} else if !containsAll(verr, test.expectedErrors...) {
				t.Errorf("Did not find expected errors for case %v, %v", idx, verr)
			}
		} else if err != nil {
			t.Errorf("Unexpected error testig case %v: %v", test, err)
		}

	}

}

func Test_BuildInvalid(t *testing.T) {

	empty := make([]string, 0)
	var host Host
	_, err := Build("", "", empty...)
	glog.Infof("error %v", host)
	if err == nil {
		t.Errorf("expected error")
	}

	_, err = Build("1234", "", empty...)
	glog.Infof("build  error %v", err)
	if err == nil {
		t.Errorf("expected error")
	}

	_, err = Build("", "", empty...)
	glog.Infof("build  error %v", err)
	if err == nil {
		t.Errorf("expected error")
	}

	_, err = Build("127.0.0.1", "poolid", empty...)
	if err == nil || err.Error() != "Loopback address 127.0.0.1 cannot be used to register a host" {
		t.Errorf("Unexpected error %v", err)
	}

	_, err = Build("", "poolid", "127.0.0.1")
	glog.Infof("last build  error %v", err)

	if err == nil || err.Error() != "Loopback address 127.0.0.1 cannot be used as an IP Resource" {
		t.Errorf("Unexpected error %v", err)
	}

	_, err = Build("", "poolid", "")
	if err == nil {
		t.Errorf("Expected error %v", err)
	}

}

func Test_Build(t *testing.T) {
	ip, err := serviced.GetIPAddress()
	if err != nil {
		t.Fatalf("Unexpected error %v", err)
	}

	empty := make([]string, 0)
	host, err := Build("", "test_pool", empty...)
	glog.Infof("build  error %v", err)
	if err != nil {
		t.Errorf("Unexpected error %v", err)
	}
	if err = host.validate(); err != nil {
		t.Errorf("Validation failed %v", err)
	}

	if len(host.IPs) != 1 {
		t.Errorf("Unexpected result %v", host.IPs)
	}

	if host.IpAddr != ip {
		t.Errorf("Expected ip %v, got %v", ip, host.IPs)
	}

	if host.IPs[0].IPAddress != ip {
		t.Errorf("Expected ip %v, got %v", ip, host.IPs)
	}

}

func Test_getIPResources(t *testing.T) {

	ips, err := getIPResources("dummy_hostId", "123")
	if err == nil || err.Error() != "IP address 123 not valid for this host" {
		t.Errorf("Unexpected error %v", err)
	}
	if len(ips) != 0 {
		t.Errorf("Unexpected result %v", ips)
	}

	ips, err = getIPResources("dummy_hostId", "127.0.0.1")
	if err == nil || err.Error() != "Loopback address 127.0.0.1 cannot be used as an IP Resource" {
		t.Errorf("Unexpected error %v", err)
	}
	if len(ips) != 0 {
		t.Errorf("Unexpected result %v", ips)
	}

	ip, err := serviced.GetIPAddress()
	if err != nil {
		t.Fatalf("Unexpected error %v", err)
	}

	validIPs := []string{ip, strings.ToLower(ip), strings.ToUpper(ip), fmt.Sprintf("   %v   ", ip)}
	for _, validIP := range validIPs {
		ips, err = getIPResources("dummy_hostId", validIP)
		if err != nil {
			if err != nil {
				t.Errorf("Unexpected error %v", err)
			}
			if len(ips) != 1 {
				t.Errorf("Unexpected result %v", ips)
			}
		}
	}

}
