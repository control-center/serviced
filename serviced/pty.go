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

 int GoGetproc(int fd, char *tty) {
    FILE *f;
    char *path, *buf;
    size_t len;
    int ch;
    pid_t pgrp;
    int r;

    if ((pgrp == tcgetgrp(fd)) == -1 {
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
    "syscall"
)

type Terminal struct {
    fd int
    pid int
    pty string
}

func (t *Terminal) fork(filename string, args []string, env []string, cwd string, cols int, rows int, uid int, gid int) error {

    var winp = new(C.struct_winsize)
    winp.ws_col = C.int(cols)
    winp.ws_row = C.int(rows)
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
        return PtyError("forkpty(3) failed")
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

        syscall.Exec(filename, argv, env)
        panic("exec failed")
    default:
        if err := syscall.SetNonblock(t.fd, true); err != nil {
            return err
        }
    }

    return nil
}

func (t *Terminal) open(cols int, rows int) error {
    var winp = new(C.struct_winsize)
    winp.ws_col = C.int(cols)
    winp.ws_row = C.int(rows)
    winp.ws_xpixel = 0
    winp.ws_ypixel = 0

    var master, slave C.int
    var name = make([]C.char, 40)

    if ret := int(C.GoOpenPty(&master, &slave, &name[0], &winp)); ret == -1 {
        return PtyError("openpty(3) failed")
    }

    t.fd = int(master)
    t.pid = int(slave)
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
    winp.ws_col = C.int(cols)
    winp.ws_row = C.int(rows)
    winp.ws_xpixel = 0
    winp.ws_ypixel = 0

    if ret := int(C.GoResize(t.fd, &winp); ret == -1 {
        return PtyError("ioctl(2) failed")
    }

    return nil
}

func (t *Terminal) getproc() string {
    fd   := C.int(t.fd)
    tty  := C.CString(t.pty)
    name := C.GoString(C.GoGetproc(fd, tty))

    return name
}
