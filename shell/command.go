package shell

import (
	"io"
	"os/exec"
	"syscall"
)

type Command struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	stderr io.ReadCloser
}

func CreateCommand(file string, argv []string) (*Command, error) {
	c := new(Command)
	c.cmd = exec.Command(file, argv...)

	// initialize pipes
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

	return c, nil
}

func (c *Command) Stdin() io.Writer {
	return c.stdin
}

func (c *Command) Stdout() io.Reader {
	return c.stdout
}

func (c *Command) Stderr() io.Reader {
	return c.stderr
}

func (c *Command) Resize(cols, rows *int) error {
	return nil
}

func (c *Command) Wait() error {
	return c.cmd.Wait()
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
}
