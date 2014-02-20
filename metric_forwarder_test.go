// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

// Package agent implements a service that runs on a serviced node. It is
// responsible for ensuring that a particular node is running the correct services
// and reporting the state and health of those services back to the master
// serviced.

package serviced

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	//"log"
)

var address string
var forwarder sync.Once
var server sync.Once

// start a metric forwarder
func startForwarder() {
	metric_redirect := fmt.Sprintf("http://%s/api/metrics/store", address)
	forwarder, _ := NewMetricForwarder(":22350", metric_redirect)
	go forwarder.Serve()
}

//echo the Request body into the response
func metricHandler(w http.ResponseWriter, r *http.Request) {
	buf := new(bytes.Buffer)
	buf.ReadFrom(r.Body)
	response := "{\"echo\":\"" + buf.String() + "\"}"
	w.Write(bytes.NewBufferString( response).Bytes())
}

// start a metric server
func startMetricServer() {
	http.HandleFunc("/api/metrics/store", metricHandler)
	server := httptest.NewServer(nil)
	address = server.Listener.Addr().String()
}

func TestMetricForwarding(t *testing.T) {
	server.Do(startMetricServer)
	forwarder.Do(startForwarder)
	request, _ := http.NewRequest("POST", "http://localhost:22350/api/metrics/store", strings.NewReader("{}"))
	request.Header.Add("Content-Type", "application/json")

	client := &http.Client{}
	response, err := client.Do(request)
	if err == nil {
		defer response.Body.Close()
		buf := new(bytes.Buffer)
		buf.ReadFrom(response.Body)
    expected := "{\"echo\":\"{}\"}"
    if buf.String() != expected  {
			t.Error("Forwarding Expected ", expected, ", But Got ", buf.String())
		}
	} else {
		t.Error("Request failed: %s", err)
	}
}
