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

// +build integration

package web

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"testing"
)

var (
	ErrTestUserExists = errors.New("test user exists. could not create new")
)
//var adminGroup = "sudo"

const (
	testUserName = "testuser"
	testUserPassword = "secret"
	testUserPasswordEncrypted = "$1$Qd8H95T5$RYSZQeoFbEB.gS19zS99A0"
	testUserSalt = "$1$Qd8H95T5$"
)

type testUser struct {
	username   string
	password   string
	group      string
	expired    bool
	wasCreated bool
}

var testUsers = map[string]testUser{
	"gooduser": testUser{"ztestgooduser", "ztestgooduserpass", adminGroup, false, false},
	"nonrootuser": testUser{"ztestplainuser", "ztestplainuserpass", "users", false, false},
	"oldbutgooduser": testUser{"ztestolduser", "ztestolduserpass", adminGroup, true, false},
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
		description:         "good user with correct password should validate",
	},
	testCase{
		user:                testUsers["gooduser"],
		testPassword:        "",
		testGroup:           adminGroup,
		expectedPamResult:   false,
		expectedGroupResult: true,
		expectedResult:      false,
		description:         "good user with empty password should fail",
	},
	testCase{
		user:                testUsers["nonrootuser"],
		testPassword:        testUsers["nonrootuser"].password,
		testGroup:           adminGroup,
		expectedPamResult:   true,
		expectedGroupResult: false,
		expectedResult:      false,
		description:         "nonroot user should fail",
	},
	testCase{
		user:                testUsers["oldbutgooduser"],
		testPassword:        testUsers["oldbutgooduser"].password,
		testGroup:           adminGroup,
		expectedPamResult:   false,
		expectedGroupResult: true,
		expectedResult:      false,
		description:         "user with expired login should fail",
	},
}


func TestMain(m *testing.M) {
	err := CreateTestUsers()
	if err != nil {
		fmt.Printf("Error creating test user: %s\n", err)
	}
	fmt.Printf("Running tests\n")
	result := m.Run()
	fmt.Printf("Removing test users\n")
	RemoveTestUsers()
	fmt.Printf("Exiting\n")
	os.Exit(result)
}

func CreateTestUsers() error {
	for name, u := range (testUsers) {
		err := createTestUser(name, &u)
		if err != nil {
			fmt.Printf("Error creating user %s: %s\n", name, err)
			return err
		}
		fmt.Printf("Created user %s. user = %v\n", name, u )
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
	args := []string {"useradd", userobj.username, "-p", encryptedPassword, "-G", userobj.group }
	if userobj.expired {
		args = append(args, "-e", "1970-01-01")
	}
	command := exec.Command(cmdName, args...)
	fmt.Printf("Creating test user %s: command is %v\n", testUserName, command)
	output, cmderr := command.CombinedOutput()
	if nil != cmderr {
		fmt.Printf("Error creating testuser: %s\n", cmderr)
		fmt.Printf("Combined output: %s\n", output)
	} else {
		userobj.wasCreated = true
		createdUsers[name] = *userobj
	}
	return cmderr
}

func RemoveTestUsers() {
	for _, user := range (createdUsers) {
		if user.wasCreated {
			err := RemoveTestUser(user.username)
			if err != nil {
				fmt.Printf("Error deleting user %s: %s\n", user.username, err)
			}
		} else {
			fmt.Printf("Not removing user %s - was not created by this test run\n", user.username)
		}
	}
}

func RemoveTestUser(testUserName string) error {
	fmt.Printf("RemoveTestUser(%s) invoked.\n", testUserName);
	command := exec.Command("sudo",  "userdel", testUserName)
	fmt.Printf("Deleting test user %s\n", testUserName)

	output, cmderr := command.CombinedOutput()
	if nil != cmderr {
		fmt.Printf("Error deleting test user %s: %s\n", testUserName, cmderr)
		fmt.Printf("Combined output: %s\n", output)
	}
	return cmderr
}


func TestCrypt(t *testing.T) {
	fmt.Println("TestCrypt()\n")
	cryptResult := crypt(testUserPassword, testUserSalt)
	if cryptResult != testUserPasswordEncrypted {
		t.Fatal("crypt() function validation failed.")
	}
}

func TestOldPamValidateLogin(t *testing.T) {
	fmt.Println("TestOldPamValidateLogin()\n")
	creds := login{Username: testUserName, Password: testUserPassword}
	if !oldPamValidateLogin(&creds, adminGroup) {
		t.Fatal("pam validation for user failed.")
	}
}

func TestAuthentication(t *testing.T) {
	fmt.Println("TestAuthentication()")
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

func TestNewPamValidateLogin(t *testing.T) {
	fmt.Println("TestNewPamValidateLogin()\n")
	creds := login{Username: testUserName, Password: testUserPassword}
	if !pamValidateLogin(&creds, adminGroup) {
		t.Fatal("pam validation for user failed.")
	}
	if !isGroupMember(testUserName, "sudo") {
		t.Fatal("group membership for user failed.")
	}
}


// Not validating for now - only messing with PAM validation. A good test should be done for CP Validation later.
//func TestCPValidateLogin(t *testing.T) {
//	creds := login{ Username: testUserName, Password: testUserPassword,}
//	if !cpValidateLogin(&creds, client) {
//		t.Fatal("pam validation for user failed.")
//	}
//}
