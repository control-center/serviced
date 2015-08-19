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

package rsync

import (
	"fmt"
	"testing"

	"github.com/control-center/serviced/volume"
	"github.com/stretchr/testify/assert"
)

type ParseDFTest struct {
	label   string
	inlabel string
	inbytes []byte
	out     []volume.Usage
	outmsg  string
	err     error
	errmsg  string
}

var parsedftests = []ParseDFTest{
	{
		label:   "output from df",
		inlabel: "volumelabel",
		inbytes: []byte(`Filesystem         1B-blocks         Used         Avail
/dev/sdb1      1901233012736 172685119488 1631947202560`),
		out: []volume.Usage{
			{Label: "volumelabel on /dev/sdb1", Type: "Total Bytes", Value: 1901233012736},
			{Label: "volumelabel on /dev/sdb1", Type: "Used Bytes", Value: 172685119488},
			{Label: "volumelabel on /dev/sdb1", Type: "Available Bytes", Value: 1631947202560},
		},
		outmsg: "output did not match expectation",
		err:    nil,
		errmsg: "error was not nil",
	},
	{
		label:   "empty input handled gracefully",
		inlabel: "volumelabel",
		inbytes: []byte{},
		out:     []volume.Usage{},
		outmsg:  "output did not match expectation",
		err:     nil,
		errmsg:  "error was not nil",
	},
}

func TestStub(t *testing.T) {
	assert.True(t, true, "Test environment set up properly.")
}

func TestParseDF(t *testing.T) {
	for _, tc := range parsedftests {
		result, err := parseDFCommand(tc.inlabel, tc.inbytes)
		assert.Equal(t, err, tc.err, fmt.Sprintf("%s: %s", tc.label, tc.errmsg))
		assert.Equal(t, result, tc.out, fmt.Sprintf("%s: %s", tc.label, tc.outmsg))
	}
}
