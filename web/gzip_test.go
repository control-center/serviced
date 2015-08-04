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

package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zenoss/go-json-rest"
)

func TestGzipAccept(t *testing.T) {
	httpRequest, _ := http.NewRequest("GET", "/foo/bar", nil)
	httpRequest.Header.Set("Accept-Encoding", "gzip")
	restRequest := rest.Request{httpRequest, map[string]string{}}
	handler := func(w *rest.ResponseWriter, r *rest.Request) {
		expected := "gzip"
		encoding := w.Header().Get("Content-Encoding")
		if encoding != expected {
			t.Error(expected + " was the expected content encoding, but instead got " + encoding)
		}
	}
	w := httptest.NewRecorder()
	restResponseWriter := rest.NewResponseWriter(w, false)
	gzipHandler(handler)(&restResponseWriter, &restRequest)
}

func TestGzipNoAccept(t *testing.T) {
	httpRequest, _ := http.NewRequest("GET", "/foo/bar", nil)
	restRequest := rest.Request{httpRequest, map[string]string{}}
	handler := func(w *rest.ResponseWriter, r *rest.Request) {
		if strings.Contains(w.Header().Get("Content-Encoding"), "gzip") {
			t.Error("Content encoding was gzip when it was not set to be accepted.")
		}
	}
	w := httptest.NewRecorder()
	restResponseWriter := rest.NewResponseWriter(w, false)
	gzipHandler(handler)(&restResponseWriter, &restRequest)
}
