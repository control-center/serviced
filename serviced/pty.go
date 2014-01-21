/* @see https://github.com/chjj/pty.js/blob/master/src/unix/pty.cc */
package main

/*
 #cgo LDFLAGS: -lutil

 #include <string.h>
 #include <stdlib.h>
 #include <unistd.h>

 #include <sys/types.h>
 #include <sys/stat.h>
 #include <sys/ioctl.h>
 #include <fcntl.h>
 #include <pty.h>
 #include <termios.h>

 #include <stdio.h>
 #include <stdint.h>

 char *GoGetproc(int fd, char *tty) {
    FILE *f;
    char *path, *buf;
    size_t len;
    int ch;
    pid_t pgrp;
    int r;

    if ((pgrp = tcgetpgrp(fd)) == -1) {
        return NULL;
    }

    r = asprintf(&path, "/proc/%11d/cmdline", (long long)pgrp);
    if (r == -1 || path == NULL) return NULL;

    if ((f = fopen(path, "r")) == NULL) {
        free(path);
        return NULL;
    }

    free(path);

    len = 0;
    buf = NULL;
    while ((ch = fgetc(f)) != EOF) {
        if (ch == '\0') break;
        buf = (char *) realloc(buf, len + 2);
        if (buf == NULL) return NULL;
        buf[len++] = ch;
    }

    if (buf != NULL) {
        buf[len] = '\0';
    }

    fclose(f);
    return buf;
 }

 int GoOpenpty(int *amaster, int *aslave, char *name,
               struct winsize *winp) {
    return openpty(amaster, aslave, name, NULL, winp);
 }

 int GoForkpty(int *amaster, char *name, struct winsize *winp) {
    return forkpty(amaster, name, NULL, winp);
 }

 int GoResize(int fd, struct winsize *winp) {
 	return ioctl(fd, TIOCSWINSZ, winp);
 }


 */
import "C"
import (
    "errors"
    "fmt"
    "os"
    "reflect"
    "strings"
    "syscall"
)

type Terminal struct {
    file, name, pty string
    fd, pid, master, slave int
    cols, rows int
    readable, writeable bool
}

func CreateTerminal(file, name, cwd string, args []string, env map[string]string, cols, rows, uid, gid int) (*Terminal, error) {
    var environ map[string]string
    for _,e := range os.Environ() {
        keyvalue := strings.Split(e, "=")
        environ[keyvalue[0]] = keyvalue[1]
    }
    if file == "" {
        file = "sh"
    }
    if cols == 0 {
        cols = 80
    }
    if rows == 0 {
        rows = 24
    }
    if len(env) == 0 {
        env = environ
    }
    if reflect.DeepEqual(environ, env) {
        // Make sure we didn't start our server from inside tmux.
        delete(env, "TMUX")
        delete(env, "TMUX_PANE")

        // Make sure we didn't start our server from inside screen.
        delete(env, "STY")
        delete(env, "WINDOW")

        // Delete some variables that might confuse our terminal.
        delete(env, "WINDOWID")
        delete(env, "TERMCAP")
        delete(env, "COLUMNS")
        delete(env, "LINES")
    }
    // Set some basic env vars if they do not exist
    // USER, SHELL, HOME, LOGNAME, WINDOWID
    if cwd == "" {
        cwd,_ = os.Getwd()
    }
    if name != "" {
        // pass
    } else if env["TERM"] != "" {
        name = env["TERM"]
    } else {
        name = "xterm"
    }

    var envdata []string
    for key, value := range env {
        envdata = append(envdata, fmt.Sprintf("%s=%s", key, value))
    }

    // fork
    term := Terminal{
        file: file,
        name: name,
        cols: cols,
        rows: rows,
        readable: true,
        writeable: true,
    }
    if err := term.fork(args, envdata, cwd, uid, gid); err != nil {
        return nil, err
    }

    return &term, nil
}

func OpenTerminal(cols, rows int) (*Terminal, error){
    if cols == 0 {
        cols = 80
    }
    if rows == 0 {
        rows = 24
    }

    term := Terminal{
        cols:       cols,
        rows:       rows,
        pid:        -1,
        readable:   true,
        writeable:  true,
    }
    if err := term.open(); err != nil {
        return nil, err
    }

    var environ map[string]string
    for _,e := range os.Environ() {
        keyvalue := strings.Split(e,"=")
        environ[keyvalue[0]] = keyvalue[1]
    }
    term.fd = term.master
    term.file = term.getproc()
    term.name = environ["TERM"]

    return &term, nil
}

func (t *Terminal) Write(data []byte) (n int, err error) {
    return syscall.Write(t.fd, data)
}

func (t *Terminal) Read(data []byte) (n int, err error) {
    return syscall.Read(t.fd, data)
}

func (t *Terminal) Resize(cols, rows int) error {
    if cols == 0 {
        cols = 80
    }
    if rows == 0 {
        rows = 24
    }

    t.cols = cols
    t.rows = rows
    return t.resize(cols, rows)
}

func (t *Terminal) Kill(signal int) error {
    var s syscall.Signal
    if signal == 0 {
        s = syscall.SIGHUP
    } else {
        s = syscall.Signal(signal)
    }

    return syscall.Kill(t.pid, s)
}

func (t *Terminal) Process() string {
    p := t.getproc()
    if p == "" {
        p = t.file
    }
    return p
}

func (t *Terminal) Close() {
    t.writeable = false
    t.readable = false
}

func (t *Terminal) fork(args, env []string, cwd string, uid int, gid int) error {
    var winp = new(C.struct_winsize)
    winp.ws_col = C.ushort(t.cols)
    winp.ws_row = C.ushort(t.rows)
    winp.ws_xpixel = 0
    winp.ws_ypixel = 0

    //fork the pty
    var master C.int = -1
    var name []C.char = make([]C.char, 40)
    var pid C.int = C.GoForkpty(&master, &name[0], winp)

    t.fd = int(master)
    t.pid = int(pid)
    t.pty = C.GoString(&name[0])

    switch t.pid {
    case -1:
        return errors.New("forkpty(3) failed")
    case  0:
        if cwd != "" {
            if err := syscall.Chdir(cwd); err != nil {
                panic("chdir failed")
            }
        }

        if uid != -1 && gid != -1 {
            if err := syscall.Setgid(gid); err != nil {
                panic("setgid failed")
            }

            if err := syscall.Setuid(uid); err != nil {
                panic("setuid failed")
            }
        }

        syscall.Exec(t.file, args, env)
        panic("exec failed")
    default:
        if err := syscall.SetNonblock(t.fd, true); err != nil {
            return err
        }
    }

    return nil
}

func (t *Terminal) open() error {
    var winp = new(C.struct_winsize)
    winp.ws_col = C.ushort(t.cols)
    winp.ws_row = C.ushort(t.rows)
    winp.ws_xpixel = 0
    winp.ws_ypixel = 0

    var master, slave C.int
    var name = make([]C.char, 40)

    if ret := int(C.GoOpenpty(&master, &slave, &name[0], winp)); ret == -1 {
        return errors.New("openpty(3) failed")
    }

    t.master = int(master)
    t.slave = int(slave)
    t.pty = C.GoString(&name[0])

    if err := syscall.SetNonblock(t.fd, true); err != nil {
        return err
    }

    if err := syscall.SetNonblock(t.pid, true); err != nil {
        return err
    }
    return nil
}

func (t *Terminal) resize(cols int, rows int) error {
    var winp = new(C.struct_winsize)
    winp.ws_col = C.ushort(cols)
    winp.ws_row = C.ushort(rows)
    winp.ws_xpixel = 0
    winp.ws_ypixel = 0

    if ret := int(C.GoResize(C.int(t.fd), winp)); ret == -1 {
        return errors.New("ioctl(2) failed")
    }

    return nil
}

func (t *Terminal) getproc() string {
    fd   := C.int(t.fd)
    tty  := C.CString(t.pty)
    name := C.GoString(C.GoGetproc(fd, tty))

    return name
}
