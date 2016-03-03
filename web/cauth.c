#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <errno.h>
#include <grp.h>
#include <pwd.h>

#define ERRORF(fmt, ...) \
            do { fprintf(stderr, "ERROR %s:%d] " fmt, __FILE__, __LINE__, __VA_ARGS__); } while (0)

#define MY_FREE(ptr) \
            do { if (ptr) { if (0) { fprintf(stderr, "ERROR %s:%d] freeing %p\n", __FILE__, __LINE__, ptr); } free(ptr); } } while (0)

#define MY_MALLOC(ptr, size) \
            do { ptr = malloc(size); if (0) { fprintf(stderr, "ERROR %s:%d] malloced %p size:%lu for %s\n", __FILE__, __LINE__, ptr, size, #ptr); } } while (0)

#define MY_REALLOC(ptr, size) \
  do { ptr = realloc(ptr, size); if (0) { fprintf(stderr, "ERROR %s:%d] realloced %p size:%lu for %s\n", __FILE__, __LINE__, ptr, size, #ptr); } } while (0)

#define _CP_MAX_GROUPS 100
#define _CP_ROOT "root"

static const int max_size = 16 * 1024 * 1024;

int getUserGID(const char *username, gid_t *pw_gid)
{
    int foundGID = 0;
    char *buffer = 0;
    struct passwd *pwd = 0;

    long pwsize = sysconf(_SC_GETPW_R_SIZE_MAX);
    if (pwsize < 0) {
        ERRORF("unable to call %s\n", "sysconf(_SC_GETPW_R_SIZE_MAX)");
        goto CLEAN_RETURN;
    }

    MY_MALLOC(buffer, pwsize * sizeof(char));
    if (!buffer) {
        ERRORF("unable to allocate space for %s\n", "getpwnam_r buffer");
        goto CLEAN_RETURN;
    }

    MY_MALLOC(pwd, sizeof(struct passwd));
    if (!pwd) {
        ERRORF("unable to allocate space for %s\n", "getpwnam_r pwd");
        goto CLEAN_RETURN;
    }

    struct passwd *result = 0;
 RETRY_GETPWNAM_R:
    errno = 0;
    int retval = getpwnam_r(username, pwd, buffer, pwsize, &result);
    if (retval < 0 || !result) {
      switch(errno) {
      case ERANGE:
        if ((pwsize *2) < pwsize || (pwsize * 2) > max_size) {
          ERRORF("error: buffer limit reached at 0x%lx.\n", pwsize*2);
          goto CLEAN_RETURN;
          break;
        }
        pwsize *= 2;
        // realloc and try again
        MY_REALLOC(buffer, pwsize * sizeof(char));
        goto RETRY_GETPWNAM_R;
        break; // not reachable, thanks to goto. Added for readability/maintainability.
      default:
        ERRORF("unable to get user info for %s - getpwnam_r returned %d: %s\n",
               username, retval, strerror(retval));
        goto CLEAN_RETURN;
        break;
      }
    }

    *pw_gid = pwd->pw_gid;
    foundGID = 1;

CLEAN_RETURN:
    MY_FREE(buffer);
    MY_FREE(pwd);
    return foundGID;
}

int isGroupMember(const char *username, const char *group)
{
    int found_wheel = 0;
    char *buffer = 0;
    struct group *grp = 0;
    gid_t *group_list = 0;

    long grsize = sysconf(_SC_GETGR_R_SIZE_MAX);
    if (grsize < 0) {
        ERRORF("unable to call sysconf(_SC_GETGR_R_SIZE_MAX) to check membership for user:%s in group:%s\n", username, group);
        goto CLEAN_RETURN;
    }

    MY_REALLOC(buffer, grsize * sizeof(char));
    if (!buffer) {
        ERRORF("unable to allocate space for getgrgid_r buffer to check membership for user:%s in group:%s\n", username, group);
        goto CLEAN_RETURN;
    }

    MY_REALLOC(grp, sizeof(struct group));
    if (!grp) {
        ERRORF("unable to allocate space for getgrgid_r pwd to check membership for user:%s in group:%s\n", username, group);
        goto CLEAN_RETURN;
    }

    int num_groups = _CP_MAX_GROUPS;
    MY_MALLOC(group_list, num_groups * sizeof(gid_t));
    if (!group_list) {
        ERRORF("unable to allocate space for getpwnam_r group_list to check membership for user:%s in group:%s\n", username, group);
        goto CLEAN_RETURN;
    }

    int pw_gid = 0;
    if (!getUserGID(username, &pw_gid)) {
        ERRORF("unable to getUserGID to check membership for user:%s in group:%s\n", username, group);
        goto CLEAN_RETURN;
    }
    if (getgrouplist(username, pw_gid, group_list, &num_groups) == -1) {
        ERRORF("unable to getgrouplist to check membership for user:%s in group:%s\n", username, group);
        goto CLEAN_RETURN;
    }

    int i=0;
    for (i=0; i < num_groups; i++) {
        struct group *result = 0;
    RETRY_GETGRGID_R:
        errno = 0;
        int retval = getgrgid_r(group_list[i], grp, buffer, grsize, &result);
        if (retval < 0 || !result) {
          switch(errno) {
          case ERANGE:
            if ((grsize *2) < grsize || (grsize * 2) > max_size) {
              ERRORF("error: buffer limit reached at 0x%lx.\n", grsize*2);
              goto CLEAN_RETURN;
              break;
            }
            // realloc and try again
            grsize *= 2;
            MY_REALLOC(buffer, grsize * sizeof(char));
            goto RETRY_GETGRGID_R;
            break; // not reachable, thanks to goto. Added for readability/maintainability.
          default:
            ERRORF("unable to check membership for user:%s in group:%s - getgrgid_r(%d, <grp>, <buffer>, %ld) returned %d: %s\n",
                   username, group, group_list[i], grsize, retval, strerror(retval));
            goto CLEAN_RETURN;
            break;
          }
        }
        if (strcmp(group, result->gr_name) == 0 || strcmp(_CP_ROOT, result->gr_name) == 0) {
            found_wheel = 1;
            goto CLEAN_RETURN;
        }
    }

CLEAN_RETURN:
    MY_FREE(buffer);
    MY_FREE(grp);
    MY_FREE(group_list);
    return found_wheel;
}
