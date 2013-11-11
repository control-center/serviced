package web

//#include <stdlib.h>
//#include <string.h>
//#include <security/pam_appl.h>
//#cgo LDFLAGS: -lpam
//extern int authenticate(const char *pam_file, const char *username, const char* pass);
import "C"
import (
	"github.com/zenoss/glog"

	"fmt"
	"os/user"
	"unsafe"
)

// currently logged in user
var currentUser *user.User

func init() {
	var err error
	currentUser, err = user.Current()
	if err != nil {
		panic(fmt.Errorf("Could not get current user: %s", err))
	}
}

func validateLogin(creds *Login) bool {
	var cprog *C.char = C.CString("sudo")
	defer C.free(unsafe.Pointer(cprog))
	var cuser *C.char = C.CString(creds.Username)
	defer C.free(unsafe.Pointer(cuser))
	var cpass *C.char = C.CString(creds.Password)
	defer C.free(unsafe.Pointer(cpass))
	auth_res := C.authenticate(cprog, cuser, cpass)
	glog.V(1).Infof("PAM result for %s was %d", creds.Username, auth_res)
	if auth_res != 0 && currentUser.Username != creds.Username && currentUser.Uid != "0" {
		glog.Errorf("This process must run as root to authenticate users other than %s", currentUser.Username)
	}
	return (auth_res == 0)
}
