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
