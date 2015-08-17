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

// +build unit

package utils

import (
	"bytes"
	"os"
	"reflect"
	"testing"
)

func TestEnvironConfigReader_parse(t *testing.T) {
	config := EnvironConfigReader{"SERVICEDTEST_"}

	// Set some environment variables
	os.Setenv("SERVICEDTEST_STRING", "hello world")
	os.Setenv("SERVICEDTEST_STRINGSLICE", "apple,orange,banana")
	os.Setenv("SERVICEDTEST_INT", "5")
	os.Setenv("SERVICEDTEST_TBOOL", "true")
	os.Setenv("SERVICEDTEST_FBOOL", "f")
	t.Logf("Environment: %v", os.Environ())

	// Verify data
	t.Logf("SERVICEDTEST_STRING: %s", os.Getenv("SERVICEDTEST_STRING"))
	verify(t, "SERVICEDTEST_STRING", config.StringVal("STRING", ""), "hello world")
	verify(t, "SERVICEDTEST_DEFAULTSTRING", config.StringVal("DEFAULTSTRING", "goodbye, world"), "goodbye, world")

	t.Logf("SERVICEDTEST_STRINGSLICE: %s", os.Getenv("SERVICEDTEST_STRINGSLICE"))
	verify(t, "SERVICEDTEST_STRINGSLICE", config.StringSlice("STRINGSLICE", []string{}), []string{"apple", "orange", "banana"})
	verify(t, "SERVICEDTEST_DEFAULTSTRINGSLICE", config.StringSlice("DEFAULTSTRINGSLICE", []string{"grapes", "mangos", "papayas"}), []string{"grapes", "mangos", "papayas"})

	t.Logf("SERVICEDTEST_INT: %s", os.Getenv("SERVICEDTEST_INT"))
	verify(t, "SERVICEDTEST_INT", config.IntVal("INT", 0), 5)
	verify(t, "SERVICEDTEST_DEFAULTINT", config.IntVal("DEFAULTINT", 10), 10)

	t.Logf("SERVICEDTEST_TBOOL: %s", os.Getenv("SERVICEDTEST_TBOOL"))
	verify(t, "SERVICEDTEST_TBOOL", config.BoolVal("TBOOL", false), true)

	t.Logf("SERVICEDTEST_FBOOL: %s", os.Getenv("SERVICEDTEST_FBOOL"))
	verify(t, "SERVICEDTEST_FBOOL", config.BoolVal("FBOOL", true), false)
	verify(t, "SERVICEDTEST_DEFAULTBOOL", config.BoolVal("DEFAULTBOOL", true), true)

	// Parser test
	examplefile := `
# SERVICEDTEST_STRING=applesauce
SERVICEDTEST_STRINGSLICE=big,red,guava
SERVICEDTEST_INT=100 # Some additional comments
SERVICEDTEST_BOOL=no`

	reader := bytes.NewBufferString(examplefile)
	if err := config.parse(reader); err != nil {
		t.Fatalf("Could not parse reader: %s", err)
	}

	// Verify data
	verify(t, "SERVICEDTEST_STRING", config.StringVal("STRING", ""), "hello world")
	verify(t, "SERVICEDTEST_STRINGSLICE", config.StringSlice("STRINGSLICE", []string{}), []string{"big", "red", "guava"})
	verify(t, "SERVICEDTEST_INT", config.IntVal("INT", 0), 100)
	verify(t, "SERVICEDTEST_BOOL", config.BoolVal("BOOL", true), false)
}

func verify(t *testing.T, key string, actual, expected interface{}) {
	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("Key: %v; Expected %v; Got %v", key, expected, actual)
	}
}
