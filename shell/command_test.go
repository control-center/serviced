package shell

import (
	"errors"
	"io"
	"strings"
	"syscall"
	"testing"
	"time"
)

type osprocess struct{}

func (p *osprocess) Signal(s syscall.Signal) error {
	return nil
}

func (p *osprocess) Kill() error {
	return nil
}

type rwcloser struct {
	data chan byte
}

func (rw *rwcloser) Read(data []byte) (int, error) {
	var n int

	if rw.data == nil {
		return 0, errors.New("nil error on read!")
	}

	for n = 0; n < len(data); n++ {
		if m, ok := <-rw.data; !ok {
			rw.data = nil
			return n, io.EOF
		} else {
			data[n] = m
		}
	}

	return n, nil
}

func (rw *rwcloser) Write(data []byte) (int, error) {
	var n int
	for n = 0; n < len(data); n++ {
		rw.data <- data[n]
	}

	return n, nil
}

func (r *rwcloser) Close() (err error) {
	close(r.data)
	return
}

type osrunner struct {
	stdin   *rwcloser
	stdout  *rwcloser
	stderr  *rwcloser
	Process *osprocess

	start error
	wait  error
}

func (c *osrunner) StdinPipe() (io.WriteCloser, error) {
	return c.stdin, nil
}

func (c *osrunner) StdoutPipe() (io.ReadCloser, error) {
	return c.stdout, nil
}

func (c *osrunner) StderrPipe() (io.ReadCloser, error) {
	return c.stderr, nil
}

func (c *osrunner) Start() error {
	return c.start
}

func (c *osrunner) Wait() error {
	return c.wait
}

func (c *osrunner) Signal(s syscall.Signal) error {
	return nil
}

func (c *osrunner) Kill() error {
	return nil
}

func TestPipe(t *testing.T) {
	reader := &rwcloser{make(chan byte)}
	out := make(chan string)
	size := 20

	test_strings := []string{
		"test first line\ntest second line\n",
		"very very long string to read output",
		"on one string",
	}

	// Write to the buffer
	go func() {
		for _, m := range test_strings {
			reader.Write([]byte(m))
			time.Sleep(IDLE_TIMEOUT)
		}
		reader.Close()
	}()

	// Pipe
	go func() {
		if err := pipe(reader, out, size); err != nil {
			t.Fatalf("pipe received unexpected error %v", err)
		}

		if err := pipe(reader, out, size); err == nil {
			t.Fatalf("pipe expected error, but recieved nil")
		}
	}()

	// Read from the channel
	go func() {
		for _, m := range test_strings {
			for _, split := range strings.SplitAfter(m, "\n") {
				for i := 0; i < len(split); i += size {
					var expected, actual string

					if len(split) > i+size {
						expected = split[i : i+size]
					} else {
						expected = split[i:]
					}
					actual, ok := <-out

					if !ok {
						t.Fatalf("channel closed unexpectedly")
					}

					if expected != actual {
						t.Fatalf("expected: %s, actual: %s", expected, actual)
					}
				}
			}
		}

		if _, ok := <-out; ok {
			t.Fatalf("channel is still open!")
		}
	}()
}

func TestCommandReader(t *testing.T) {
	var command Command
	waiterr := errors.New("wait error")

	// receive error on stdout
	command = Command{
		cmd:        &osrunner{wait: waiterr},
		stdin:      nil,
		stdout:     &rwcloser{},
		stderr:     &rwcloser{make(chan byte)},
		stdoutChan: make(chan string),
		stderrChan: make(chan string),
	}

	command.stderr.Close()
	if err := command.Reader(10); err == nil {
		t.Fatalf("missing error on stdout!")
	} else if err == waiterr {
		t.Fatalf("unexpected error: %v", err)
	}

	// receive error on stderr
	command = Command{
		cmd:        &osrunner{wait: waiterr},
		stdin:      nil,
		stdout:     &rwcloser{make(chan byte)},
		stderr:     &rwcloser{data: nil},
		stdoutChan: make(chan string),
		stderrChan: make(chan string),
	}
	command.stdout.Close()
	if err := command.Reader(10); err == nil {
		t.Fatalf("missing error on stderr!")
	} else if err == waiterr {
		t.Fatalf("unexpected error: %v", err)
	}

	// recieve error on wait
	command = Command{
		cmd:        &osrunner{wait: waiterr},
		stdin:      nil,
		stdout:     &rwcloser{make(chan byte)},
		stderr:     &rwcloser{make(chan byte)},
		stdoutChan: make(chan string),
		stderrChan: make(chan string),
	}
	command.stdout.Close()
	command.stderr.Close()
	if err := command.Reader(10); err == nil {
		t.Fatalf("missing error on wait!")
	} else if err != waiterr {
		t.Fatalf("unexpected error: %v", err)
	}

	// success
	command = Command{
		cmd:        &osrunner{},
		stdin:      nil,
		stdout:     &rwcloser{make(chan byte)},
		stderr:     &rwcloser{make(chan byte)},
		stdoutChan: make(chan string),
		stderrChan: make(chan string),
	}
	command.stdout.Close()
	command.stderr.Close()
	if err := command.Reader(10); err != nil {
		t.Fatalf("unexpected error on reader!")
	}
}