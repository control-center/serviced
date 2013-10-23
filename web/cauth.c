#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <security/pam_appl.h>
#include "_cgo_export.h"

int conv(int num_msg, 
         const struct pam_message **msg,
         struct pam_response **resp, 
         void *appdata_ptr)
{
        resp[0] = (struct pam_response*)appdata_ptr;
        return (PAM_SUCCESS);
}

int authenticate(const char *pam_file, const char *username, const char* pass)
{
        pam_handle_t *pamh;
        int retval;
        struct pam_response *pw_reply = malloc(sizeof(struct pam_response));
        pw_reply->resp = strdup(pass);
        struct pam_conv pam_conversation = { conv, (void*)pw_reply };
        if ((retval = pam_start(pam_file, username, &pam_conversation, &pamh)) != PAM_SUCCESS) 
        {
                return -1;
        }
        if ((retval = pam_authenticate(pamh, PAM_DISALLOW_NULL_AUTHTOK)) != PAM_SUCCESS) {
                retval = pam_end(pamh, PAM_DATA_SILENT);
                return -1;
        }
        if ((retval = pam_acct_mgmt(pamh, 0)) != PAM_SUCCESS) {
                retval = pam_end(pamh, PAM_DATA_SILENT);
                return -1;
        }
        if ((retval = pam_end(pamh, PAM_DATA_SILENT)) != PAM_SUCCESS) {
                return -1;
        }
        return 0;
}
