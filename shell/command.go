package shell

import (
	"bytes"
	"io"
	"os/exec"
	"syscall"
	"time"
)

type Command struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	stderr io.ReadCloser

	stdoutChan chan string
	stderrChan chan string
	done       chan bool
	err        error
}

func CreateCommand(file string, argv []string) (*Command, error) {
	c := new(Command)
	c.cmd = exec.Command(file, argv...)

	// initialize pipes & channels
	if stdin, err := c.cmd.StdinPipe(); err != nil {
		return nil, err
	} else {
		c.stdin = stdin
	}
	if stdout, err := c.cmd.StdoutPipe(); err != nil {
		return nil, err
	} else {
		c.stdout = stdout
	}
	if stderr, err := c.cmd.StderrPipe(); err != nil {
		return nil, err
	} else {
		c.stderr = stderr
	}

	// start
	if err := c.cmd.Start(); err != nil {
		return nil, err
	}

	c.stdoutChan = make(chan string)
	c.stderrChan = make(chan string)
	c.done = make(chan bool)

	return c, nil
}

func (c *Command) Reader(size int) {
	var (
		eof                  bool = false
		stdoutMsg, stderrMsg chan byte
		stdoutErr, stderrErr chan error
		stdoutBuf, stderrBuf bytes.Buffer
	)
	stdoutMsg, stdoutErr = pipe(c.stdout, size)
	stderrMsg, stderrErr = pipe(c.stderr, size)

	defer func() {
		close(stdoutMsg)
		close(stdoutErr)
		close(stderrMsg)
		close(stderrErr)
	}()

	for {
		select {
		case m := <-stdoutMsg:
			stdoutBuf.WriteByte(m)
			if m == '\n' || size < stdoutBuf.Len() {
				c.stdoutChan <- stdoutBuf.String()
				stdoutBuf.Reset()
			}
		case e := <-stdoutErr:
			if e == io.EOF {
				if stdoutBuf.Len() > 0 {
					c.stdoutChan <- stdoutBuf.String()
					stdoutBuf.Reset()
				}
				if eof {
					if err := c.cmd.Wait(); err != nil {
						c.err = err
					}
					c.done <- true
					return
				}
				eof = true
			} else {
				c.err = e
				c.done <- true
				return
			}
		case m := <-stderrMsg:
			stderrBuf.WriteByte(m)
			if m == '\n' || size < stderrBuf.Len() {
				c.stderrChan <- stderrBuf.String()
				stderrBuf.Reset()
			}
		case e := <-stderrErr:
			if e == io.EOF {
				if stderrBuf.Len() > 0 {
					c.stderrChan <- stderrBuf.String()
					stderrBuf.Reset()
				}
				if eof {
					if err := c.cmd.Wait(); err != nil {
						c.err = err
					}
					c.done <- true
					return
				}
				eof = true
			} else {
				c.err = e
				c.done <- true
				return
			}
		case <-time.After(250 * time.Millisecond):
			// Hanging process; dump whatever is on the pipes
			if stdoutBuf.Len() > 0 {
				c.stdoutChan <- stdoutBuf.String()
				stdoutBuf.Reset()
			}
			if stderrBuf.Len() > 0 {
				c.stderrChan <- stderrBuf.String()
				stderrBuf.Reset()
			}
		}
	}
}

func (c *Command) Writer(data []byte) (int, error) {
	return c.stdin.Write(data)
}

func (c *Command) Stdout() chan string {
	return c.stdoutChan
}

func (c *Command) Stderr() chan string {
	return c.stderrChan
}

func (c *Command) Exited() chan bool {
	return c.done
}

func (c *Command) Error() error {
	return c.err
}

func (c *Command) Resize(cols, rows *int) error {
	return nil
}

func (c *Command) Kill(signal *int) error {
	var s syscall.Signal
	if signal == nil {
		s = syscall.SIGKILL
	} else {
		s = syscall.Signal(*signal)
	}
	return c.cmd.Process.Signal(s)
}

func (c *Command) Close() {
	c.stdin.Close()
	close(c.stdoutChan)
	close(c.stderrChan)
}

func pipe(reader io.Reader, size int) (chan byte, chan error) {
	bchan := make(chan byte, size)
	echan := make(chan error)

	go func() {
		for {
			buffer := make([]byte, 1)
			n, err := reader.Read(buffer)
			if n > 0 {
				bchan <- buffer[0]
			} else {
				echan <- err
				return
			}
		}
	}()
	return bchan, echan
}
