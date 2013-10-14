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
	glog.Infof("Attempting PAM auth for %s", creds.Username)
	var cprog *C.char = C.CString("su")
	defer C.free(unsafe.Pointer(cprog))
	var cuser *C.char = C.CString(creds.Username)
	defer C.free(unsafe.Pointer(cuser))
	var cpass *C.char = C.CString(creds.Password)
	defer C.free(unsafe.Pointer(cpass))
	return (C.authenticate(cprog, cuser, cpass) == 0);
}

