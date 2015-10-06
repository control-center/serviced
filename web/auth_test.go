// Copyright 2015 The Serviced Authors.
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

// +build root, integration

package web

import (
	"errors"
	"os"
	"os/exec"
	"os/user"
	"testing"
	"github.com/zenoss/glog"
)

var (
	ErrTestUserExists = errors.New("test user exists. could not create new")
)

const (
	testUserPassword = "secret"
	testUserPasswordEncrypted = "$1$Qd8H95T5$RYSZQeoFbEB.gS19zS99A0"
	testUserSalt = "$1$Qd8H95T5$"
)

type testUser struct {
	username string
	password string
	group    string
	expired  bool
}

var testUsers = map[string]testUser{
	"gooduser": testUser{"ztestgooduser", "ztestgooduserpass", adminGroup, false},
	"nonrootuser": testUser{"ztestplainuser", "ztestplainuserpass", "users", false},
	"oldbutgooduser": testUser{"ztestolduser", "ztestolduserpass", adminGroup, true},
}

var createdUsers = make(map[string]testUser)

type testCase struct {
	user                testUser
	testPassword        string
	testGroup           string
	expectedPamResult   bool
	expectedGroupResult bool
	expectedResult      bool
	description         string
}

var testCases = []testCase{
	testCase{
		user:                testUsers["gooduser"],
		testPassword:        testUsers["gooduser"].password,
		testGroup:           adminGroup,
		expectedPamResult:   true,
		expectedGroupResult: true,
		expectedResult:      true,
		description:         "good user with correct password should pass PAM, pass group check, and should validate",
	},
	testCase{
		user:                testUsers["gooduser"],
		testPassword:        "",
		testGroup:           adminGroup,
		expectedPamResult:   false,
		expectedGroupResult: true,
		expectedResult:      false,
		description:         "good user with empty password should fail PAM, pass group check, and should not validate.",
	},
	testCase{
		user:                testUsers["nonrootuser"],
		testPassword:        testUsers["nonrootuser"].password,
		testGroup:           adminGroup,
		expectedPamResult:   true,
		expectedGroupResult: false,
		expectedResult:      false,
		description:         "nonroot user should pass PAM, fail authentication, and should not validate.",
	},
	testCase{
		user:                testUsers["oldbutgooduser"],
		testPassword:        testUsers["oldbutgooduser"].password,
		testGroup:           adminGroup,
		expectedPamResult:   false,
		expectedGroupResult: true,
		expectedResult:      false,
		description:         "user with expired login should pass PAM, fail group check, and should not validate.",
	},
}


func TestMain(m *testing.M) {
	if 0 != os.Geteuid() {
		glog.Infof("Must be root to run integration tests. Exiting (no tests run).")
		os.Exit(0)
	}
	err := CreateTestUsers()
	if err != nil {
		glog.Errorf("Error creating test user: %s\n", err)
	}
	glog.Infof("Running tests - some errors expected from negative tests.\n")
	result := m.Run()
	glog.Infof("Removing test users\n")
	RemoveTestUsers()
	glog.Infof("Exiting\n")
	os.Exit(result)
}

func CreateTestUsers() error {
	for name, u := range (testUsers) {
		err := createTestUser(name, &u)
		if err != nil {
			glog.Errorf("Error creating user %s: %s\n", name, err)
			return err
		}
		glog.V(2).Infof("Created user %s. user = %v\n", name, u)
	}
	return nil
}


func createTestUser(name string, userobj *testUser) error {
	testUserName := userobj.username
	if _, err := user.Lookup(testUserName); err == nil {
		return ErrTestUserExists
	}
	encryptedPassword := crypt(userobj.password, testUserSalt)
	cmdName := "sudo"
	args := []string{"useradd", userobj.username, "-p", encryptedPassword, "-G", userobj.group }
	if userobj.expired {
		args = append(args, "-e", "1970-01-01")
	}
	command := exec.Command(cmdName, args...)
	glog.V(2).Infof("Creating test user %s: command is %v\n", testUserName, command)
	output, cmderr := command.CombinedOutput()
	if nil != cmderr {
		glog.Errorf("Error creating testuser: %s\n%s\n", cmderr, output)
	} else {
		createdUsers[name] = *userobj
	}
	return cmderr
}

func RemoveTestUsers() {
	for _, user := range (createdUsers) {
		err := RemoveTestUser(user.username)
		if err != nil {
			glog.Infof("Error deleting user %s: %s\n", user.username, err)
		}  else {
			glog.Infof("Successfully removed user %s\n", user.username)
		}
	}
}

func RemoveTestUser(testUserName string) error {
	glog.V(2).Infof("RemoveTestUser(%s) invoked.\n", testUserName)
	command := exec.Command("sudo", "userdel", testUserName)
	glog.V(2).Infof("Deleting test user %s\n", testUserName)

	output, cmderr := command.CombinedOutput()
	if nil != cmderr {
		glog.Errorf("Error deleting test user %s: %s\n%s\n", testUserName, cmderr, output)
	}
	return cmderr
}

func TestCrypt(t *testing.T) {
	glog.V(2).Infof("TestCrypt()\n")
	cryptResult := crypt(testUserPassword, testUserSalt)
	if cryptResult != testUserPasswordEncrypted {
		t.Fatal("crypt() function validation failed.")
	}
}

func TestAuthentication(t *testing.T) {
	glog.V(2).Infof("TestAuthentication()")
	for _, tc := range (testCases) {
		user := tc.user
		creds := login{Username: user.username, Password: tc.testPassword}
		pamResult := pamValidateLoginOnly(&creds, adminGroup)
		if pamResult != tc.expectedPamResult {
			t.Errorf("pam validation for user %s failed: %s", user.username, tc.description)
		}
		groupResult := isGroupMember(user.username, adminGroup)
		if groupResult != tc.expectedGroupResult {
			t.Errorf("group membership for user %s failed: %s", user.username, tc.description)
		}
		result := pamValidateLogin(&creds, adminGroup)
		if result != tc.expectedResult {
			t.Errorf("User Authentication for user %s failed: %s", user.username, tc.description)
		}
	}
}

