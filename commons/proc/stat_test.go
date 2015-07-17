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

// +build unit

package proc

import (
	"testing"
)

func TestGetProcStat(t *testing.T) {

	// mock up our proc dir
	defer func(s string) {
		procDir = s
	}(procDir)
	procDir = ""

	procStat, err := GetProcStat(0)
	if err != nil {
		t.Fatalf("could not get procstat: %s", err)
	}

	testProc := &ProcStat{
		Pid:      10132,
		Filename: "(cp something)",
		State:    "R",
		Ppid:     2028,
		Pgrp:     10132,
		Session:  2028,
	}
	if *testProc != *procStat {
		t.Fatalf("%+v != %+v", testProc, procStat)
	}
}
