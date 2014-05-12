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
