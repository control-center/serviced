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
