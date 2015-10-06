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
//extern int isGroupMember(const char *username, const char *group);
import "C"
import (
	"github.com/msteinert/pam"
	"github.com/zenoss/glog"

	"fmt"
	"os/user"
	"unsafe"
	"errors"
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

func oldPamValidateLogin(creds *login, group string) bool {
	var cprog = C.CString("sudo")
	defer C.free(unsafe.Pointer(cprog))
	var cuser = C.CString(creds.Username)
	defer C.free(unsafe.Pointer(cuser))
	var cpass = C.CString(creds.Password)
	defer C.free(unsafe.Pointer(cpass))
	var cgroup = C.CString(group)
	defer C.free(unsafe.Pointer(cgroup))
	authRes := C.authenticate(cprog, cuser, cpass, cgroup)
	glog.V(1).Infof("PAM result for user:%s group:%s was %d", creds.Username, group, authRes)
	if authRes != 0 && currentUser.Username != creds.Username && currentUser.Uid != "0" {
		glog.Errorf("This process must run as root to authenticate users other than %s", currentUser.Username)
	}
	return (authRes == 0)
}

func isGroupMember(username, group string) bool {
	var cuser = C.CString(username)
	defer C.free(unsafe.Pointer(cuser))
	var cgroup = C.CString(group)
	defer C.free(unsafe.Pointer(cgroup))
	result := C.isGroupMember(cuser, cgroup)
	glog.Infof("C.isGroupMember(%s, %s) returned %d.\n", username, group, result)
	return (result != 0)
}

func pamValidateLogin(creds *login, group string) bool {
	return pamValidateLoginOnly(creds, group) && isGroupMember(creds.Username, group)
}

func makePamConvHandler(creds *login) func(pam.Style, string) (string, error) {
	return   func(s pam.Style, msg string) (string, error) {
		switch s {
		case pam.PromptEchoOff:
			return creds.Password, nil
		case pam.PromptEchoOn:
			glog.V(1).Infof("PAM Prompt: %s\n", msg)
			return creds.Username, nil
		case pam.ErrorMsg:
			glog.Errorf("PAM ERROR: %s\n", msg)
			return "", nil
		case pam.TextInfo:
			glog.V(1).Infof("PAM MESSAGE: %s\n", msg)
			return "", nil
		}
		return "", errors.New("Unrecognized message style")
	}
}

func pamValidateLoginOnly(creds *login, group string) bool {
	t, err := pam.StartFunc("", "",  makePamConvHandler(creds))
	if err != nil {
		glog.Errorf("Start: %s", err.Error())
		return false
	}
	err = t.Authenticate(0)
	if err != nil {
		glog.Errorf("Authentication failed for user %s: Authenticate: %s", creds.Username, err.Error())
		return false
	}
	err = t.AcctMgmt(pam.Silent)
	if err != nil {
		glog.Errorf("Authentication failed for usere %s: AcctMgmt: %s", creds.Username, err.Error())
		return false
	}

	return true
}


