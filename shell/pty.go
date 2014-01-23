/* @see https://github.com/chjj/pty.js/blob/master/src/unix/pty.cc */
package shell

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
	"syscall"
)

type pty struct {
	fd, pid, master, slave int
	pty                    string
}

func (t *pty) fork(file string, args, env []string, cwd string, cols, rows, uid, gid int) error {
	var winp = new(C.struct_winsize)
	winp.ws_col = C.ushort(cols)
	winp.ws_row = C.ushort(rows)
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
	case 0:
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

		syscall.Exec(file, args, env)
		panic("exec failed")
	default:
		if err := syscall.SetNonblock(t.fd, true); err != nil {
			return err
		}
	}

	return nil
}

func (t *pty) open(cols, rows int) error {
	var winp = new(C.struct_winsize)
	winp.ws_col = C.ushort(cols)
	winp.ws_row = C.ushort(rows)
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

func (t *pty) resize(cols, rows int) error {
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

func (t *pty) getproc() string {
	fd := C.int(t.fd)
	tty := C.CString(t.pty)
	name := C.GoString(C.GoGetproc(fd, tty))

	return name
}
