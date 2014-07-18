// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package shell

import (
	"io"
)

type ShellReader struct {
	buffer chan byte
}

func (r ShellReader) Read(p []byte) (n int, err error) {
	for n, _ = range p {
		select {
		case b, ok := <-r.buffer:
			if ok {
				p[n] = b
			} else {
				return n, io.EOF
			}
		default:
			return n, io.EOF
		}
	}

	return n, nil
}

type ShellWriter struct {
	buffer chan byte
}

func (w ShellWriter) Write(p []byte) (n int, err error) {
	var b byte
	for n, b = range p {
		w.buffer <- b
	}

	return n, nil
}
