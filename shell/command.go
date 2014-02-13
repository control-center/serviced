package shell

import (
	"bytes"
	"io"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

const IDLE_TIMEOUT = 50 * time.Millisecond

func CreateCommand(file string, argv []string) (*Command, error) {
	c := new(Command)
	c.cmd = &cmd{exec.Command(file, argv...)}

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

	c.stdoutChan = make(chan string)
	c.stderrChan = make(chan string)

	// start
	if err := c.cmd.Start(); err != nil {
		return nil, err
	}

	return c, nil
}

func (c *Command) Reader(size int) (err error) {
	var stdoutErr, stderrErr error
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		stdoutErr = pipe(c.stdout, c.stdoutChan, size)
		wg.Done()
	}()

	go func() {
		stderrErr = pipe(c.stderr, c.stderrChan, size)
		wg.Done()
	}()

	wg.Wait()

	if stdoutErr != nil {
		err = stdoutErr
	} else if stderrErr != nil {
		err = stderrErr
	} else {
		err = c.cmd.Wait()
	}

	return
}

func (c *Command) Write(data []byte) (int, error) {
	return c.stdin.Write(data)
}

func (c *Command) StdoutPipe() chan string {
	return c.stdoutChan
}

func (c *Command) StderrPipe() chan string {
	return c.stderrChan
}

func (c *Command) Signal(signal syscall.Signal) error {
	return c.cmd.Signal(signal)
}

func (c *Command) Kill() error {
	return c.cmd.Kill()
}

func pipe(reader io.Reader, out chan<- string, size int) error {
	var buffer bytes.Buffer
	bchan := make(chan byte, size)
	echan := make(chan error)

	go func() {
		defer func() {
			close(bchan)
			close(echan)
		}()

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

	defer close(out)
	for bchan != nil || echan != nil {
		select {
		case b, ok := <-bchan:
			if !ok {
				bchan = nil
				continue
			}
			buffer.WriteByte(b)
			if b == '\n' || buffer.Len() >= size {
				out <- buffer.String()
				buffer.Reset()
			}
		case e, ok := <-echan:
			if !ok {
				echan = nil
				continue
			}

			if e == io.EOF {
				if buffer.Len() > 0 {
					out <- buffer.String()
					buffer.Reset()
				}
			} else {
				return e
			}
		case <-time.After(IDLE_TIMEOUT):
			if buffer.Len() > 0 {
				out <- buffer.String()
				buffer.Reset()
			}
		}

	}

	return nil
}
