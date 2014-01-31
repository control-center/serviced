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

	stdoutChan chan string
	stderrChan chan string
	done       chan bool
	err        error
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
		pty:        pty{},
		file:       file,
		name:       name,
		cols:       *cols,
		rows:       *rows,
		readable:   true,
		writeable:  true,
		stdoutChan: make(chan string),
		stderrChan: make(chan string),
		done:       make(chan bool),
	}
	if err := term.fork(file, args, envv, cwd, *cols, *rows, *uid, *gid); err != nil {
		return nil, err
	}
	return &term, nil
}

func (t *Terminal) Reader(size int) {
	for {
		data := make([]byte, size)
		n, e := syscall.Read(t.fd, data)
		switch e {
		case nil, io.EOF:
			if n > 0 {
				t.stdoutChan <- string(data[:n])
			}
		case syscall.EIO:
			t.done <- true
			break
		default:
			t.err = e
			t.done <- true
			break
		}
	}
}

func (t *Terminal) Write(data []byte) (int, error) {
	return syscall.Write(t.fd, data)
}

func (t *Terminal) StdoutPipe() chan string {
	return t.stdoutChan
}

func (t *Terminal) StderrPipe() chan string {
	return t.stderrChan
}

func (t *Terminal) ExitedPipe() chan bool {
	return t.done
}

func (t *Terminal) Error() error {
	return t.err
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

func (t *Terminal) Signal(sig int) error {
    s := syscall.Signal(sig)
    return syscall.Kill(t.pid, s)
}

func (t *Terminal) Kill() error {
    return syscall.Kill(t.pid, syscall.SIGHUP)
}

func (t *Terminal) Close() {
	t.readable = false
	t.writeable = false
	close(t.stdoutChan)
	close(t.stderrChan)
	syscall.Close(t.fd)
}

func (t *Terminal) GetProcess() string {
	process := t.getproc()
	if process == "" {
		process = t.file
	}
	return process
}
