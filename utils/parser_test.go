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
	"io/ioutil"
	"os"
	"reflect"
	"testing"
	"time"

	. "gopkg.in/check.v1"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type S struct {
	dir string
}

var _ = Suite(&S{})

func (s *S) SetUpTest(c *C) {
	s.dir = c.MkDir()
}

func (s *S) tmpFile(name, contents string) string {
	f, _ := ioutil.TempFile(s.dir, name)
	_, _ = f.WriteString(contents)
	f.Close()
	return f.Name()
}

func TestEnvironConfigReader_parse(t *testing.T) {
	config := EnvironConfigReader{"", "SERVICEDTEST_", map[string]ConfigValue{}}

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

	parsedValues := config.GetConfigValues()
	if len(parsedValues) != 9 {
		t.Errorf("len(parsedValues) failed: expected %d got %d", 9, len(parsedValues))
	}
	verifyConfigValue(t, "STRING", parsedValues, ConfigValue{"SERVICEDTEST_STRING", "hello world"})
	verifyConfigValue(t, "DEFAULTSTRING", parsedValues, ConfigValue{"SERVICEDTEST_DEFAULTSTRING", "goodbye, world"})
	verifyConfigValue(t, "STRINGSLICE", parsedValues, ConfigValue{"SERVICEDTEST_STRINGSLICE", "apple,orange,banana"})
	verifyConfigValue(t, "DEFAULTSTRINGSLICE", parsedValues, ConfigValue{"SERVICEDTEST_DEFAULTSTRINGSLICE", "grapes,mangos,papayas"})
	verifyConfigValue(t, "INT", parsedValues, ConfigValue{"SERVICEDTEST_INT", "5"})
	verifyConfigValue(t, "DEFAULTINT", parsedValues, ConfigValue{"SERVICEDTEST_DEFAULTINT", "10"})
	verifyConfigValue(t, "TBOOL", parsedValues, ConfigValue{"SERVICEDTEST_TBOOL", "true"})
	verifyConfigValue(t, "FBOOL", parsedValues, ConfigValue{"SERVICEDTEST_FBOOL", "f"})
	verifyConfigValue(t, "DEFAULTBOOL", parsedValues, ConfigValue{"SERVICEDTEST_DEFAULTBOOL", "true"})

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

	// There is only one new value in examplefile, so the length of parsed values should only increase by 1
	parsedValues = config.GetConfigValues()
	if len(parsedValues) != 10 {
		t.Errorf("len(parsedValues) failed: expected %d got %d", 10, len(parsedValues))
	}
	verifyConfigValue(t, "STRING", parsedValues, ConfigValue{"SERVICEDTEST_STRING", "hello world"})
	verifyConfigValue(t, "STRINGSLICE", parsedValues, ConfigValue{"SERVICEDTEST_STRINGSLICE", "big,red,guava"})
	verifyConfigValue(t, "INT", parsedValues, ConfigValue{"SERVICEDTEST_INT", "100"})
	verifyConfigValue(t, "BOOL", parsedValues, ConfigValue{"SERVICEDTEST_BOOL", "no"})
}

func (s *S) TestDiffEnvironConfigReader(c *C) {
	path1 := s.tmpFile("orig", "AAA_A=MARIO\nAAA_B=LUIGI\nAAA_C=PEACH")
	reader1, err := NewEnvironConfigReader(path1, "AAA")
	c.Assert(err, IsNil)
	// Mutate a param
	path2 := s.tmpFile("orig", "AAA_A=MARIO\nAAA_B=LUIGI\nAAA_C=PEACHH")
	reader2, err := NewEnvironConfigReader(path2, "AAA")
	c.Assert(err, IsNil)
	configs := reader1.diff(reader2)
	c.Assert(configs, DeepEquals, []ConfigValue{ConfigValue{"AAA_C", "PEACHH"}})
	// Remove a param
	path3 := s.tmpFile("orig", "AAA_A=MARIO\nAAA_B=LUIGI")
	reader3, err := NewEnvironConfigReader(path3, "AAA")
	configs = reader1.diff(reader3)
	c.Assert(len(configs), Equals, 0)
	// Add a param
	path4 := s.tmpFile("orig", "AAA_A=MARIO\nAAA_B=LUIGI\nAAA_C=PEACH\nAAA_D=BOWSER")
	reader4, err := NewEnvironConfigReader(path4, "AAA")
	configs = reader1.diff(reader4)
	c.Assert(configs, DeepEquals, []ConfigValue{ConfigValue{"AAA_D", "BOWSER"}})
	// Add, delete, and mutate
	path5 := s.tmpFile("orig", "AAA_A=MARIO\nAAA_C=PEACHH\nAAA_D=BOWSER")
	reader5, err := NewEnvironConfigReader(path5, "AAA")
	configs = reader1.diff(reader5)
	c.Assert(configs, DeepEquals, []ConfigValue{ConfigValue{"AAA_C", "PEACHH"}, ConfigValue{"AAA_D", "BOWSER"}})
}

func (s *S) TestWatchEnvironConfigReader(c *C) {
	origPath := s.tmpFile("orig", "AAA_A=MARIO\nAAA_B=LUIGI\nAAA_C=PEACH")
	reader, err := NewEnvironConfigReader(origPath, "AAA")
	c.Assert(err, IsNil)
	cancelChan := make(chan struct{}, 1)
	configChan := make(chan []ConfigValue, 1)
	go WatchEnvironConfigReader(reader, configChan, cancelChan, 5)
	defer close(cancelChan)
	// modify the file
	err = ioutil.WriteFile(origPath, []byte("AAA_A=MARIO\nAAA_B=LUIGI\nAAA_C=PEACHH"), 666) // add extra 'H' to PEACH
	c.Assert(err, IsNil)
	// see if the watcher works
	var config []ConfigValue
	select {
	case config = <-configChan:
	case <-time.After(20 * time.Second):
	}
	c.Assert(config, DeepEquals, []ConfigValue{ConfigValue{"AAA_C", "PEACHH"}})

	err = ioutil.WriteFile(origPath, []byte("AAA_A=MARIO\nAAA_B=LUIGI\nAAA_C=PEACH"), 666) // remove H from PEACH
	c.Assert(err, IsNil)
	select {
	case config = <-configChan:
	case <-time.After(20 * time.Second):
	}
	c.Assert(config, DeepEquals, []ConfigValue{ConfigValue{"AAA_C", "PEACH"}})
}

func verify(t *testing.T, key string, actual, expected interface{}) {
	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("Key: %v; Expected %v; Got %v", key, expected, actual)
	}
}

func verifyConfigValue(t *testing.T, key string, parsedValues map[string]ConfigValue, expected ConfigValue) {
	if actual, ok := parsedValues[key]; !ok {
		t.Errorf("Could not find key %s in parsedValues", key)
	} else if !reflect.DeepEqual(actual, expected) {
		t.Errorf("parsedValues[%s] incorrect; Expected %v; Got %v", key, expected, actual)
	}
}
