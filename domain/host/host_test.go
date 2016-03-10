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

// +build unit

package host

import (
	"github.com/control-center/serviced/utils"
	"github.com/control-center/serviced/validation"
	"github.com/zenoss/glog"

	"strings"
	"testing"
)

func containsAll(ve *validation.ValidationError, errStrings ...string) bool {
	for _, err := range errStrings {
		if !contains(ve, err) {
			return false
		}
	}
	return true
}
func contains(ve *validation.ValidationError, errString string) bool {

	for _, err := range ve.Errors {
		if err.Error() == errString {
			return true
		}
	}
	return false
}

type validateCase struct {
	id             string
	rpcport        int
	poolid         string
	ip             string
	expectedErrors []string
}

var validatetable []validateCase

func init() {

	ip, err := utils.GetIPAddress()
	if err != nil {
		glog.Errorf("Could not determine ip %v", err)
	}

	validatetable = []validateCase{
		validateCase{"", 65535, "", "", []string{"empty string for Host.ID", "empty string for Host.PoolID", "invalid IP Address "}},
		validateCase{"hostid", 65535, "", "", []string{"empty string for Host.PoolID", "invalid IP Address "}},
		validateCase{"", 65535, "poolid", "", []string{"empty string for Host.ID", "invalid IP Address "}},
		validateCase{"hostid", 65535, "poolid", "", []string{"invalid IP Address "}},
		validateCase{"hostid", 65535, "poolid", "blam", []string{"invalid IP Address blam"}},
		validateCase{"hostid", 65535, "poolid", "127.0.0.1", []string{"host ip can not be a loopback address"}},
		validateCase{"hostid", -1, "poolid", ip, []string{"not in valid port range: -1"}},
		validateCase{"hostid", 65536, "poolid", ip, []string{"not in valid port range: 65536"}},
		validateCase{"deadb30f", 65535, "poolid", ip, []string{}},
	}

}

func Test_ValidateTable(t *testing.T) {
	for idx, test := range validatetable {
		h := New()
		h.ID = test.id
		h.RPCPort = test.rpcport
		h.PoolID = test.poolid
		h.IPAddr = test.ip

		err := h.ValidEntity()
		if len(test.expectedErrors) > 0 {
			if verr, isVErr := err.(*validation.ValidationError); !isVErr {
				t.Errorf("expected ValidationError for case %v, got %v", idx, err)
			} else if !containsAll(verr, test.expectedErrors...) {
				t.Errorf("Did not find expected errors for case %v, %v", idx, verr)
			}
		} else if err != nil {
			t.Errorf("Unexpected error testig case %v %v: %v", idx, test, err)
		}

	}

}

func Test_BuildInvalid(t *testing.T) {

	empty := make([]string, 0)
	_, err := Build("", "65535", "", "", empty...)
	if err == nil {
		t.Errorf("expected error")
	}

	_, err = Build("1234", "65535", "", "", empty...)
	if err == nil {
		t.Errorf("expected error")
	}

	_, err = Build("", "65535", "", "", empty...)
	if err == nil {
		t.Errorf("expected error")
	}

	_, err = Build("127.0.0.1", "65535", "poolid", "", empty...)
	if _, ok := err.(IsLoopbackError); !ok {
		t.Errorf("Unexpected error %v", err)
	}

	_, err = Build("", "65535", "poolid", "", "127.0.0.1")
	if _, ok := err.(IsLoopbackError); !ok {
		t.Errorf("Unexpected error %v", err)
	}

	_, err = Build("", "65535", "poolid", "", "")
	if err == nil {
		t.Errorf("Expected error %v", err)
	}

}

func Test_Build(t *testing.T) {
	ip, err := utils.GetIPAddress()
	if err != nil {
		t.Fatalf("Unexpected error %v", err)
	}

	empty := make([]string, 0)
	host, err := Build("", "65535", "test_pool", "", empty...)
	glog.Infof("build  error %v", err)
	if err != nil {
		t.Errorf("Unexpected error %v", err)
	}

	if err = host.ValidEntity(); err != nil {
		t.Errorf("Validation failed %v", err)
	}

	if len(host.IPs) != 1 {
		t.Errorf("Unexpected result %v", host.IPs)
	}

	if host.IPAddr != ip {
		t.Errorf("Expected ip %v, got %v", ip, host.IPs)
	}

	if host.IPs[0].IPAddress != ip {
		t.Errorf("Expected ip %v, got %v", ip, host.IPs)
	}

}

func Test_getIPResources(t *testing.T) {

	ips, err := getIPResources("dummy_hostId", "123")
	if _, ok := err.(InvalidIPAddress); !ok {
		t.Errorf("Unexpected error %v", err)
	}
	if len(ips) != 0 {
		t.Errorf("Unexpected result %v", ips)
	}

	ips, err = getIPResources("dummy_hostId", "127.0.0.1")
	if _, ok := err.(IsLoopbackError); !ok {
		t.Errorf("Unexpected error %v", err)
	}
	if len(ips) != 0 {
		t.Errorf("Unexpected result %v", ips)
	}

	ip, err := utils.GetIPAddress()
	if err != nil {
		t.Fatalf("Unexpected error %v", err)
	}

	validIPs := []string{ip, strings.ToLower(ip), strings.ToUpper(ip)}
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

func Test_getOSKernelData(t *testing.T) {
	kernelVersion, kernelRelease, err := getOSKernelData()

	if err != nil {
		t.Errorf("Unexpected error %v", err)
	}

	if kernelVersion == "There was an error retrieving kernel data" || kernelRelease == "There was an error retrieving kernel data" {
		t.Errorf("Unexpected error %v", err)
	}

	glog.Infof("Kernel Version:  %v Kernel Release: %v", kernelVersion, kernelRelease)
}
