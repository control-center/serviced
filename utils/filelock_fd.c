#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <fcntl.h>
#include <unistd.h>
#include <error.h>
#include <errno.h>

#define ERRORF(fmt, ...) \
            do { fprintf(stderr, "ERROR %s:%d] " fmt, __FILE__, __LINE__, __VA_ARGS__); } while (0)


int fd_fcntl(int fd, int cmd, short type, char* filepath)
{
    struct flock ft;
    memset(&ft, 0, sizeof(ft));
    ft.l_type = type;      /* Type of lock: F_RDLCK, F_WRLCK, F_UNLCK */
    ft.l_whence = SEEK_SET;   /* How to interpret l_start: SEEK_SET, SEEK_CUR, SEEK_END */
    ft.l_start = 0;           /* Starting offset for lock */
    ft.l_len = 0;             /* Number of bytes to lock */
    ft.l_pid = getpid();      /* PID of process blocking our lock (F_GETLK only) */

    if (fcntl(fd, cmd, &ft) == -1) {
        char errstr[128];
        memset(&errstr, 0, sizeof(errstr));
        int n = strerror_r(errno, errstr, sizeof(errstr)-0);
        ERRORF("unable to fcntl file %s: %s", filepath, errstr);
        return -1;
    }

    return 0;
}

int fd_lock(int fd, char* filepath)
{
    return fd_fcntl(fd, F_SETLKW, F_WRLCK, filepath);
}

int fd_unlock(int fd, char* filepath)
{
    return fd_fcntl(fd, F_SETLK, F_UNLCK, filepath);
}

#if defined(UNIT_TEST_FILELOCK)
/* compile unit test with: gcc -DUNIT_TEST_FILELOCK -o filelock_fd filelock_fd.c */
int main(int argc, char *argv[])
{
    if (argc != 2) {
        fprintf(stderr, "Usage: %s FILE\n", argv[0]);
        exit(-1);
    }

    char* filepath = argv[1];
    int fd = 0;
    if ((fd = open(filepath, O_RDWR|O_CREAT, 0664)) == -1) {
        perror("open");
        exit(1);
    }

    int rc = 0;

    // lock file
    fprintf(stdout, "locking:  %s\n", filepath);
    rc = fd_lock(fd, filepath);
    if (0 != rc) {
        exit(2);
    }
    fprintf(stdout, "locked:   %s\n", filepath);

    // sleep interval
    int interval = 10;
    fprintf(stdout, "sleeping %d\n", interval);
    sleep(interval);

    // unlock file
    rc = fd_unlock(fd, filepath);
    if (0 != rc) {
        exit(3);
    }
    fprintf(stdout, "unlocked: %s\n", filepath);

    return rc;
}
#endif