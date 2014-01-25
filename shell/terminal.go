package shell

import (
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"syscall"
)

type Terminal struct {
	pty
	file, name          string
	cols, rows          int
	readable, writeable bool
	stdin               io.Writer
	stdout              io.Reader
	err                 error
}

func CreateTerminal(name, file string, args []string, env map[string]string, cwd string, cols, rows, uid, gid *int) (*Terminal, error) {
	// convert environ to map
	var environ = make(map[string]string)
	for _, e := range os.Environ() {
		kv := strings.Split(e, "=")
		environ[kv[0]] = kv[1]
	}

	// set defaults
	if file == "" {
		file = "/bin/sh"
		args = []string{"sh"}
	}
	if cols == nil || *cols == 0 {
		cols = new(int)
		*cols = 80
	}
	if rows == nil || *rows == 0 {
		rows = new(int)
		*rows = 24
	}
	if uid == nil {
		uid = new(int)
		*uid = -1
	}
	if gid == nil {
		gid = new(int)
		*gid = -1
	}

	if len(env) == 0 {
		env = environ
	}
	if reflect.DeepEqual(environ, env) {
		// make sure we didn't start our server from inside tmux
		delete(env, "TMUX")
		delete(env, "TMUX_PANE")

		// make sure we didn't start our server from inside screen
		delete(env, "STY")
		delete(env, "WINDOW")

		// delete some variables that might confuse our terminal
		delete(env, "WINDOWID")
		delete(env, "TERMCAP")
		delete(env, "COLUMNS")
		delete(env, "LINES")
	}

	// set some basic variables if they do not exist
	// USER, SHELL, HOME, LOGNAME, WINDOWID
	if cwd == "" {
		cwd, _ = os.Getwd()
	}
	if name != "" {
		//pass
	} else if env["TERM"] != "" {
		name = env["TERM"]
	} else {
		env["TERM"] = "xterm"
	}

	envv := make([]string, len(env))
	i := 0
	for k, v := range env {
		envv[i] = fmt.Sprintf("%s=%s", k, v)
		i = i + 1
	}

	// fork
	term := Terminal{
		pty:       pty{},
		file:      file,
		name:      name,
		cols:      *cols,
		rows:      *rows,
		readable:  true,
		writeable: true,
	}
	term.stdin = &term
	term.stdout = &term

	if err := term.fork(file, args, envv, cwd, *cols, *rows, *uid, *gid); err != nil {
		return nil, err
	}

	go func() {
		_, err := syscall.Wait4(term.pid, nil, 0, nil)
		term.err = err
		term.readable = false
	}()

	return &term, nil
}

func OpenTerminal(cols, rows *int) (*Terminal, error) {
	// set defaults
	if cols == nil || *cols == 0 {
		cols = new(int)
		*cols = 80
	}
	if rows == nil || *rows == 0 {
		rows = new(int)
		*rows = 24
	}

	// open
	term := Terminal{
		pty:       pty{pid: -1},
		file:      os.Args[0],
		name:      os.Getenv("TERM"),
		cols:      *cols,
		rows:      *rows,
		readable:  true,
		writeable: true,
	}
	term.stdin = &term
	term.stdout = &term

	if err := term.open(*cols, *rows); err != nil {
		return nil, err
	}
	term.fd = term.master
	return &term, nil
}

func (t *Terminal) Stdin() io.Writer {
	return t.stdin
}

func (t *Terminal) Stdout() io.Reader {
	return t.stdout
}

func (t *Terminal) Stderr() io.Reader {
	return nil
}

func (t *Terminal) Read(data []byte) (int, error) {
	d := data

	for {
		n, err := syscall.Read(t.fd, d)
		if n == len(d) || err != io.EOF || !t.readable {
			return n, err
		} else {
			d = d[n:]
		}
	}
}

func (t *Terminal) Write(data []byte) (int, error) {
	return syscall.Write(t.fd, data)
}

func (t *Terminal) Resize(cols, rows *int) error {
	if cols == nil || *cols == 0 {
		cols = new(int)
		*cols = 80
	}
	if rows == nil || *rows == 0 {
		rows = new(int)
		*rows = 24
	}
	t.cols = *cols
	t.rows = *rows
	return t.resize(*cols, *rows)
}

func (t *Terminal) Wait() error {
	for t.readable {
	}
	return t.err
}

func (t *Terminal) Kill(signal *int) error {
	var s syscall.Signal
	if signal == nil {
		s = syscall.SIGHUP
	} else {
		s = syscall.Signal(*signal)
	}
	return syscall.Kill(t.pid, s)
}

func (t *Terminal) Close() {
	t.readable = false
	t.writeable = false

	if t.master > 0 && t.slave > 0 {
		syscall.Close(t.master)
		syscall.Close(t.slave)
	} else {
		syscall.Close(t.fd)
	}
}

func (t *Terminal) GetProcess() string {
	process := t.getproc()
	if process == "" {
		process = t.file
	}
	return process
}
