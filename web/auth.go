package web
//#include <stdlib.h>
//#include <string.h>
//#include <security/pam_appl.h>
//#cgo LDFLAGS: -lpam
//extern int authenticate(const char *pam_file, const char *username, const char* pass);
import "C"
import (
	"github.com/zenoss/glog"
	
	"unsafe"
)

func validateLogin(creds *Login) bool {
	var cprog *C.char = C.CString("sudo")
	defer C.free(unsafe.Pointer(cprog))
	var cuser *C.char = C.CString(creds.Username)
	defer C.free(unsafe.Pointer(cuser))
	var cpass *C.char = C.CString(creds.Password)
	defer C.free(unsafe.Pointer(cpass))
	auth_res := C.authenticate(cprog, cuser, cpass)
	glog.Infof("PAM result for %s was %d", creds.Username, auth_res);
	return (auth_res == 0)
}

