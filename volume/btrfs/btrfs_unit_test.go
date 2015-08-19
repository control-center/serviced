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

package btrfs

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"math"
	"testing"
)

type ParseDFTest struct {
	label  string
	in     []string
	out    []BtrfsDFData
	outmsg string
	err    error
	errmsg string
}

/*
type BtrfsDFData struct {
	DataType string
	Level    string
	Total    uint64
	Used     uint64
}*/

const (
	GiB = uint64(1024 * 1024 * 1024)
	MiB = uint64(1024 * 1024)
	KiB = uint64(1024)
	B   = uint64(1)
)

var parsedftests = []ParseDFTest{
	{
		label: "output from btrfs fi df (v3.12)",
		in: []string{
			"Data, single: total=9.00GiB, used=8.67GiB",
			"System, DUP: total=32.00MiB, used=16.00KiB",
			"Metadata, DUP: total=1.00GiB, used=466.88MiB",
		},
		out: []BtrfsDFData{
			{DataType: "Data", Level: "single", Total: toBytes(9.00, GiB), Used: toBytes(8.67, GiB)},
			{DataType: "System", Level: "DUP", Total: toBytes(32.00, MiB), Used: toBytes(16.00, KiB)},
			{DataType: "Metadata", Level: "DUP", Total: toBytes(1.00, GiB), Used: toBytes(466.88, MiB)},
		},
		outmsg: "output did not match expectation",
		err:    nil,
		errmsg: "error was not nil",
	},
	{
		label: "empty lines should be tolerated in input",
		in: []string{
			"Data, single: total=9.00GiB, used=8.67GiB",
			"System, DUP: total=32.00MiB, used=16.00KiB",
			"",
			"Metadata, DUP: total=1.00GiB, used=466.88MiB",
			"",
		},
		out: []BtrfsDFData{
			{DataType: "Data", Level: "single", Total: toBytes(9.00, GiB), Used: toBytes(8.67, GiB)},
			{DataType: "System", Level: "DUP", Total: toBytes(32.00, MiB), Used: toBytes(16.00, KiB)},
			{DataType: "Metadata", Level: "DUP", Total: toBytes(1.00, GiB), Used: toBytes(466.88, MiB)},
		},
		outmsg: "output did not match expectation",
		err:    nil,
		errmsg: "error was not nil",
	},
	{
		label: "output from btrfs fi df (v3.17)",
		in: []string{
			"System, DUP: total=8.00MiB, used=16.00KiB",
			"System, single: total=4.00MiB, used=0.00B",
			"Metadata, DUP: total=51.19MiB, used=112.00KiB",
			"Metadata, single: total=8.00MiB, used=0.00B",
			"GlobalReserve, single: total=16.00MiB, used=0.00B",
		},
		out: []BtrfsDFData{
			{DataType: "System", Level: "DUP", Total: toBytes(8.00, MiB), Used: toBytes(16.00, KiB)},
			{DataType: "System", Level: "single", Total: toBytes(4.00, MiB), Used: toBytes(0.00, B)},
			{DataType: "Metadata", Level: "DUP", Total: toBytes(51.19, MiB), Used: toBytes(112.00, KiB)},
			{DataType: "Metadata", Level: "single", Total: toBytes(8.00, MiB), Used: toBytes(0.00, B)},
			{DataType: "GlobalReserve", Level: "single", Total: toBytes(16.00, MiB), Used: toBytes(0.00, B)},
		},
		outmsg: "output did not match expectation",
		err:    nil,
		errmsg: "error was not nil",
	},
	{
		label: "output from btrfs fi df --raw (v3.17)",
		in: []string{
			"System, DUP: total=8388608, used=16384",
			"System, single: total=4194304, used=0",
			"Metadata, DUP: total=53673984, used=114688",
			"Metadata, single: total=8388608, used=0",
			"GlobalReserve, single: total=16777216, used=0",
		},
		out: []BtrfsDFData{
			{DataType: "System", Level: "DUP", Total: uint64(8388608), Used: uint64(16384)},
			{DataType: "System", Level: "single", Total: uint64(4194304), Used: uint64(0)},
			{DataType: "Metadata", Level: "DUP", Total: uint64(53673984), Used: uint64(114688)},
			{DataType: "Metadata", Level: "single", Total: uint64(8388608), Used: uint64(0)},
			{DataType: "GlobalReserve", Level: "single", Total: uint64(16777216), Used: uint64(0)},
		},
		outmsg: "output did not match expectation",
		err:    nil,
		errmsg: "error was not nil",
	},
	{
		label: "bad line in output (wrong number of fields)",
		in: []string{
			"Data, single: total=9.00GiB, used=8.67GiB",
			"System, DUP: total=32.00MiB, used=16.00KiB",
			"Metadata, DUP: total=1.00GiB, used=466.88MiB",
			"arglebarglefoo",
		},
		out:    []BtrfsDFData{},
		outmsg: "output should be empty",
		err:    fmt.Errorf("Wrong number of fields (%d, expected 4) in line %q", 1, "arglebarglefoo"),
		errmsg: "error did not match expectation",
	},
	{
		label: "bad line in output (bad first field name)",
		in: []string{
			"NotReal, single: total=9.00GiB, used=8.67GiB",
			"System, DUP: total=32.00MiB, used=16.00KiB",
			"Metadata, DUP: total=1.00GiB, used=466.88MiB",
		},
		out:    []BtrfsDFData{},
		outmsg: "output should be empty",
		err:    fmt.Errorf("Unrecognized field %q in line %q", "NotReal", "NotReal, single: total=9.00GiB, used=8.67GiB"),
		errmsg: "error did not match expectation",
	},
	{
		label: "bad line in output (wrong number of fields - too many)",
		in: []string{
			"Data, single: total=9.00GiB, used=8.67GiB, extraField=1.23KiB",
			"System, DUP: total=32.00MiB, used=16.00KiB",
			"Metadata, DUP: total=1.00GiB, used=466.88MiB",
		},
		out:    []BtrfsDFData{},
		outmsg: "output should be empty",
		err:    fmt.Errorf("Wrong number of fields (%d, expected 4) in line %q", 5, "Data, single: total=9.00GiB, used=8.67GiB, extraField=1.23KiB"),
		errmsg: "error did not match expectation",
	},
	{
		label: "bad line in output (wrong number of fields - too few)",
		in: []string{
			"Data, single: total=9.00GiB",
			"System, DUP: total=32.00MiB, used=16.00KiB",
			"Metadata, DUP: total=1.00GiB, used=466.88MiB",
		},
		out:    []BtrfsDFData{},
		outmsg: "output should be empty",
		err:    fmt.Errorf("Wrong number of fields (%d, expected 4) in line %q", 3, "Data, single: total=9.00GiB"),
		errmsg: "error did not match expectation",
	},
	{
		label: "bad line in output (total field missing)",
		in: []string{
			"Data, single: nottotal=9.00GiB, used=8.67GiB",
			"System, DUP: total=32.00MiB, used=16.00KiB",
			"Metadata, DUP: total=1.00GiB, used=466.88MiB",
		},
		out:    []BtrfsDFData{},
		outmsg: "output should be empty",
		err:    fmt.Errorf("expected total field, not found in line %q", "Data, single: nottotal=9.00GiB, used=8.67GiB"),
		errmsg: "error did not match expectation",
	},
	{
		label: "bad line in output (used field missing)",
		in: []string{
			"Data, single: total=9.00GiB, notused=8.67GiB",
			"System, DUP: total=32.00MiB, used=16.00KiB",
			"Metadata, DUP: total=1.00GiB, used=466.88MiB",
		},
		out:    []BtrfsDFData{},
		outmsg: "output should be empty",
		err:    fmt.Errorf("expected used field, not found in line %q", "Data, single: total=9.00GiB, notused=8.67GiB"),
		errmsg: "error did not match expectation",
	},
	{
		label: "too few lines in output",
		in: []string{
			"System, DUP: total=32.00MiB, used=16.00KiB",
			"Metadata, DUP: total=1.00GiB, used=466.88MiB",
		},
		out:    []BtrfsDFData{},
		outmsg: "output should be empty",
		err: fmt.Errorf("insufficient output: %v",
			"System, DUP: total=32.00MiB, used=16.00KiB\nMetadata, DUP: total=1.00GiB, used=466.88MiB"),
		errmsg: "error did not match expectation",
	},
	{
		label: "too few lines in output - with blank lines",
		in: []string{
			"System, DUP: total=32.00MiB, used=16.00KiB",
			"",
			"Metadata, DUP: total=1.00GiB, used=466.88MiB",
		},
		out:    []BtrfsDFData{},
		outmsg: "output should be empty",
		err: fmt.Errorf("insufficient output: %v",
			"System, DUP: total=32.00MiB, used=16.00KiB\nMetadata, DUP: total=1.00GiB, used=466.88MiB"),
		errmsg: "error did not match expectation",
	},
}

func toBytes(value float64, multiplier uint64) uint64 {
	return uint64(math.Floor(value * float64(multiplier)))
}

func TestStub(t *testing.T) {
	assert.True(t, true, "Test environment set up properly.")
}

func TestParseDF(t *testing.T) {
	for _, tc := range parsedftests {
		result, err := parseDF(tc.in)
		assert.Equal(t, err, tc.err, fmt.Sprintf("%s: %s", tc.label, tc.errmsg))
		assert.Equal(t, result, tc.out, fmt.Sprintf("%s: %s", tc.label, tc.outmsg))
	}
}
