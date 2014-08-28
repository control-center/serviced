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

var testDockerLogsFile = "getDockerLogs.txt"
var testDockerLogsOutput = []string{
	"2014-08-28 16:22:49,100 WARN Included extra file \"/opt/zenoss/etc/supervisor/central-query_supervisor.conf\" during parsing\n",
	"2014-08-28 16:22:49,194 INFO RPC interface 'supervisor' initialized\n",
	"2014-08-28 16:22:49,194 CRIT Server 'inet_http_server' running without any HTTP authentication checking\n",
	"2014-08-28 16:22:49,195 INFO supervisord started with pid 1\n",
	"2014-08-28 16:22:50,197 INFO spawned: 'metric-consumer-app' with pid 9\n",
	"2014-08-28 16:22:50,199 INFO spawned: 'opentsdb' with pid 10\n",
	"2014-08-28 16:22:50,200 INFO spawned: 'central-query' with pid 11\n",
	"2014-08-28 16:22:55,272 INFO success: metric-consumer-app entered RUNNING state, process has stayed up for > than 5 seconds (startsecs)\n",
	"2014-08-28 16:22:55,272 INFO success: opentsdb entered RUNNING state, process has stayed up for > than 5 seconds (startsecs)\n",
	"2014-08-28 16:22:55,272 INFO success: central-query entered RUNNING state, process has stayed up for > than 5 seconds (startsecs)\n",
}

func TestGetLastDockerLogs(t *testing.T) {
	if _, err := getLastDockerLogs("should not exist", 1000); err == nil {
		t.Fatalf("expected error, file should not exist")
	}

	if _, err := getLastDockerLogs(testDockerLogsFile, 1000000000000); err != nil {
		t.Fatalf("should not blow up getting more data than exists: %s", err)
	}

	// attempt to break deserialization by seeking on a non json object boundary
	if logs, err := getLastDockerLogs(testDockerLogsFile, 2019); err != nil {
		t.Fatal("should no break when seeking to non-json object boundary: %s", err)
	} else {
		if logs == nil {
			t.Fatalf("logs should not be nil")
		}
		if len(logs) != len(testDockerLogsOutput) {
			t.Fatalf("log lengths should match: \n%s", logs)
		}
		for i, line := range logs {
			if line != testDockerLogsOutput[i] {
				t.Fatalf("Docker logs differ '%s' vs '%s'", line, testDockerLogsOutput[i])
			}
		}
	}

}

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
