// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

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
		panic(fmt.Errorf("could not get current user: %s", err))
	}
}

func pamValidateLogin(creds *login) bool {
	var cprog = C.CString("sudo")
	defer C.free(unsafe.Pointer(cprog))
	var cuser = C.CString(creds.Username)
	defer C.free(unsafe.Pointer(cuser))
	var cpass = C.CString(creds.Password)
	defer C.free(unsafe.Pointer(cpass))
	authRes := C.authenticate(cprog, cuser, cpass)
	glog.V(1).Infof("PAM result for %s was %d", creds.Username, authRes)
	if authRes != 0 && currentUser.Username != creds.Username && currentUser.Uid != "0" {
		glog.Errorf("This process must run as root to authenticate users other than %s", currentUser.Username)
	}
	return (authRes == 0)
}
