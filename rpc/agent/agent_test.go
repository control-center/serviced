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

package agent

import (
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/utils"

	"strings"
	"testing"
)

func TestGetInfo(t *testing.T) {

	ip, err := utils.GetIPAddress()
	if err != nil {
		t.Fatalf("Unexpected error %v", err)
	}

	staticIPs := []string{ip}
	agent := NewServer(staticIPs)
	h := host.New()
	request := BuildHostRequest{IP: "", PoolID: "testpool"}

	err = agent.BuildHost(request, h)
	if err != nil && !strings.Contains(err.Error(), "not valid for this host") {
		t.Fatalf("Unexpected error %v", err)
	}
	if len(h.IPs) != 0 {
		t.Fatalf("Unexpected result %v (%d)", h.IPs, len(h.IPs))
	}

	request = BuildHostRequest{IP: "127.0.0.1", PoolID: "testpool"}

	err = agent.BuildHost(request, h)
	if err == nil || err.Error() != "loopback address 127.0.0.1 cannot be used to register a host" {
		t.Fatalf("Unexpected error %v", err)
	}

	request = BuildHostRequest{IP: "", PoolID: "testpool"}

	err = agent.BuildHost(request, h)
	if err == nil || !strings.Contains(err.Error(), "not valid for this host") {
		t.Errorf("Unexpected error %v", err)
	}

}
