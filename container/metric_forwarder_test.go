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

// Package agent implements a service that runs on a serviced node. It is
// responsible for ensuring that a particular node is running the correct services
// and reporting the state and health of those services back to the master
// serviced.

package container

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

var address string
var forwarder sync.Once
var server sync.Once

// start a metric forwarder
func startForwarder() (*MetricForwarder, error) {
	metricRedirect := fmt.Sprintf("http://%s/api/metrics/store", address)
	return NewMetricForwarder(":22350", metricRedirect)
}

//echo the Request body into the response
func metricHandler(w http.ResponseWriter, r *http.Request) {
	buf := new(bytes.Buffer)
	buf.ReadFrom(r.Body)
	response := "{\"echo\":\"" + buf.String() + "\"}"
	w.Write(bytes.NewBufferString(response).Bytes())
}

// start a metric server
func startMetricServer() {
	http.HandleFunc("/api/metrics/store", metricHandler)
	server := httptest.NewServer(nil)
	address = server.Listener.Addr().String()
}

func TestMetricForwarding(t *testing.T) {
	server.Do(startMetricServer)

	f, err := startForwarder()
	if err != nil {
		t.Fatalf("Could not start forwarder: %s", err)
	}
	defer f.Close()

	request, _ := http.NewRequest("POST", "http://localhost:22350/api/metrics/store", strings.NewReader("{}"))
	request.Header.Add("Content-Type", "application/json")

	client := &http.Client{}
	response, err := client.Do(request)
	if err == nil {
		defer response.Body.Close()
		buf := new(bytes.Buffer)
		buf.ReadFrom(response.Body)
		expected := "{\"echo\":\"{}\"}"
		if buf.String() != expected {
			t.Error("Forwarding Expected ", expected, ", But Got ", buf.String())
		}
	} else {
		t.Errorf("Request failed: %s", err)
	}
}
