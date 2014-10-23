// Copyright 2014 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package web

//#include <stdlib.h>
//#include <string.h>
//#include <security/pam_appl.h>
//#cgo LDFLAGS: -lpam
//extern int authenticate(const char *pam_file, const char *username, const char* pass, const char* group);
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

func authResError(authRes int) error {
	errs := []error{
		fmt.Errorf("pam: succeeded"),
		fmt.Errorf("pam: start failed"),
		fmt.Errorf("pam: authentication failed"),
		fmt.Errorf("pam: invalid account"),
		fmt.Errorf("pam: invalid admin group"),
	}

	index := int(authRes)
	if index >= len(errs) || index < 0 {
		return fmt.Errorf("auth: index:%d out of valid range 0 .. %d", index, len(errs)-1)
	}

	return errs[authRes]
}

func pamValidateLogin(creds *login, group string) bool {
	var cprog = C.CString("sudo")
	defer C.free(unsafe.Pointer(cprog))
	var cuser = C.CString(creds.Username)
	defer C.free(unsafe.Pointer(cuser))
	var cpass = C.CString(creds.Password)
	defer C.free(unsafe.Pointer(cpass))
	var cgroup = C.CString(group)
	defer C.free(unsafe.Pointer(cgroup))
	authRes := C.authenticate(cprog, cuser, cpass, cgroup)
	if authRes != 0 {
		glog.Errorf("PAM result for user:%s group:%s was %d: %v", creds.Username, group, authRes, authResError(int(authRes)))
	} else {
		glog.V(1).Infof("PAM result for user:%s group:%s was %d: %v", creds.Username, group, authRes, authResError(int(authRes)))
	}
	if authRes != 0 && currentUser.Username != creds.Username && currentUser.Uid != "0" {
		glog.Errorf("This process must run as root to authenticate users other than %s", currentUser.Username)
	}
	return (authRes == 0)
}
