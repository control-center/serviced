package web

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"

	"github.com/zenoss/go-json-rest"
)

type gzipResponseWriter struct {
	http.ResponseWriter
	gzipWriter io.Writer
}

func (self gzipResponseWriter) Write(b []byte) (int, error) {
	return self.gzipWriter.Write(b)
}

func gzipHandler(h handlerFunc) handlerFunc {
	return func(w *rest.ResponseWriter, r *rest.Request) {
		if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			gzw := gzip.NewWriter(w)
			defer gzw.Close()
			grw := gzipResponseWriter{w, gzw}
			grw.Header().Add("Vary", "Accept-Encoding")
			grw.Header().Set("Content-Encoding", "gzip")
			writer := rest.NewResponseWriter(grw, false)
			h(&writer, r)
		} else {
			h(w, r)
		}
	}
}
