#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <grp.h>
#include <pwd.h>
#include <security/pam_appl.h>

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

/* Enforces group membership */
int authenticate(const char *pam_file, const char *username, const char* pass, const char *group)
{
        pam_handle_t *pamh;
        int retval, i, found_wheel, num_groups;
        struct passwd *pw;
        struct group *gr;
        num_groups = _CP_MAX_GROUPS;
        found_wheel = 0;
        gid_t *group_list = malloc(_CP_MAX_GROUPS * sizeof(gid_t));
        struct pam_response *pw_reply = malloc(sizeof(struct pam_response));
        pw_reply->resp = strdup(pass);
        struct pam_conv pam_conversation = { conv, (void*)pw_reply };
        if ((retval = pam_start(pam_file, username, &pam_conversation, &pamh)) != PAM_SUCCESS) {
                free(group_list);
                return _CP_FAIL_START;
        }
        if ((retval = pam_authenticate(pamh, PAM_DISALLOW_NULL_AUTHTOK)) != PAM_SUCCESS) {
                free(group_list);
                pam_end(pamh, PAM_DATA_SILENT);
                return _CP_FAIL_AUTH;
        }
        if ((retval = pam_acct_mgmt(pamh, PAM_SILENT)) != PAM_SUCCESS) {
                free(group_list);
                pam_end(pamh, PAM_DATA_SILENT);
                return _CP_FAIL_ACCT;
        }
        pw = getpwnam(username);
        if (getgrouplist(username, pw->pw_gid, group_list, &num_groups) == -1) {
                free(group_list);
                pam_end(pamh, PAM_DATA_SILENT);
                return _CP_FAIL_WHEEL;
        }
        for (i=0; i < _CP_MAX_GROUPS; i++) {
                gr = getgrgid(group_list[i]);
                if (gr == NULL) {
                        break;
                }
                if (strcmp(group, gr->gr_name) == 0 || strcmp(_CP_ROOT, gr->gr_name) == 0) {
                        found_wheel = 1;
                        break;
                }
        }

        free(group_list);
        pam_end(pamh, PAM_DATA_SILENT);
        return found_wheel? _CP_SUCCESS : _CP_FAIL_WHEEL;
}
