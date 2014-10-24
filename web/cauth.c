#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <grp.h>
#include <pwd.h>
#include <security/pam_appl.h>

#define ERRORF(fmt, ...) \
            do { fprintf(stderr, "ERROR %s:%d] " fmt, __FILE__, __LINE__, __VA_ARGS__); } while (0)

#define MY_FREE(ptr) \
            do { if (ptr) { if (0) { fprintf(stderr, "ERROR %s:%d] freeing %p\n", __FILE__, __LINE__, ptr); } free(ptr); } } while (0)

#define MY_MALLOC(ptr, size) \
            do { ptr = malloc(size); if (0) { fprintf(stderr, "ERROR %s:%d] malloced %p size:%lu for %s\n", __FILE__, __LINE__, ptr, size, #ptr); } } while (0)

#define _CP_MAX_GROUPS 100
#define _CP_ROOT "root"
#define _CP_SUCCESS    0
#define _CP_FAIL_START 1
#define _CP_FAIL_AUTH  2
#define _CP_FAIL_ACCT  3
#define _CP_FAIL_WHEEL 4

int conv(int num_msg, 
         const struct pam_message **msg,
         struct pam_response **resp, 
         void *appdata_ptr)
{
        resp[0] = (struct pam_response*)appdata_ptr;
        return (PAM_SUCCESS);
}


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
    int retval = getpwnam_r(username, pwd, buffer, pwsize, &result);
    if (retval < 0 || !result) {
        ERRORF("unable to get user info for %s - getpwnam_r returned %d: %s\n",
            username, retval, strerror(retval));
        goto CLEAN_RETURN;
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

    MY_MALLOC(buffer, grsize * sizeof(char));
    if (!buffer) {
        ERRORF("unable to allocate space for getgrgid_r buffer to check membership for user:%s in group:%s\n", username, group);
        goto CLEAN_RETURN;
    }

    MY_MALLOC(grp, sizeof(struct group));
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
        int retval = getgrgid_r(group_list[i], grp, buffer, grsize, &result);
        if (retval < 0 || !result) {
            ERRORF("unable to check membership for user:%s in group:%s - getgrgid_r(%d) returned %d: %s\n",
                username, group, group_list[i], retval, strerror(retval));
            break;
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

/* Enforces group membership */
int authenticate(const char *pam_file, const char *username, const char* pass, const char *group)
{
    int return_code = _CP_FAIL_AUTH;
    int retval = 0;
    pam_handle_t *pamh = 0;

    struct pam_response *pw_reply = 0;
    MY_MALLOC(pw_reply, sizeof(struct pam_response));
    if (!pw_reply) {
        ERRORF("unable to allocate space for pam_response to authenticate user:%s\n", username);
        goto CLEAN_RETURN;
    }

    pw_reply->resp = strdup(pass);
    if (!pw_reply->resp) {
        ERRORF("unable to strdup to authenticate user:%s\n", username);
        goto CLEAN_RETURN;
    }
    struct pam_conv pam_conversation = { conv, (void*)pw_reply };

    if ((retval = pam_start(pam_file, username, &pam_conversation, &pamh)) != PAM_SUCCESS) {
        ERRORF("pam_start for user:%s returned %d: %s\n", username, retval, pam_strerror(pamh, retval));
        pamh = 0;    // set to zero to prevent cleanup from calling pam_end
        return_code = _CP_FAIL_START;
        goto CLEAN_RETURN;
    }
    if ((retval = pam_authenticate(pamh, PAM_DISALLOW_NULL_AUTHTOK)) != PAM_SUCCESS) {
        ERRORF("pam_authenticate for user:%s returned %d: %s\n", username, retval, pam_strerror(pamh, retval));
        return_code = _CP_FAIL_AUTH;
        goto CLEAN_RETURN;
    }
    if ((retval = pam_acct_mgmt(pamh, PAM_SILENT)) != PAM_SUCCESS) {
        ERRORF("pam_acct_mgmt for user:%s returned %d: %s\n", username, retval, pam_strerror(pamh, retval));
        return_code = _CP_FAIL_ACCT;
        goto CLEAN_RETURN;
    }

    int found_wheel = isGroupMember(username, group);
    if (!found_wheel) {
        ERRORF("unable to find user:%s in group:%s\n", username, group);
        return_code = _CP_FAIL_WHEEL;
    } else {
        return_code = _CP_SUCCESS;
    }

CLEAN_RETURN:
    if (pamh) pam_end(pamh, PAM_DATA_SILENT);  // pam_end probably cleans up pw_reply
    // HACK: commented this out on purpose due to coredump ---> if (pw_reply && pw_reply->resp) MY_FREE(pw_reply->resp);
    // HACK: commented this out on purpose due to coredump ---> MY_FREE(pw_reply);
    return return_code;
}

#if defined(UNIT_TEST_CAUTH)
/* compile unit test with: gcc -DUNIT_TEST_CAUTH -o cauth cauth.c -lpam */
int main(int argc, char *argv[])
{
    if (argc != 5) {
        fprintf(stderr, "Usage: %s FILE USER PASS GROUP\n", argv[0]);
        exit(-1);
    }

    int rc = authenticate(argv[1], argv[2], argv[3], argv[4]);
    printf("authenticate() returned %d\n", rc);
    return rc;
}
#endif
