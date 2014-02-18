package shell

import (
	"os/exec"
	"syscall"
)

type cmd struct {
	*exec.Cmd
}

func (c *cmd) Signal(s syscall.Signal) error {
	return c.Process.Signal(s)
}

func (c *cmd) Kill() error {
	return c.Process.Kill()
}
