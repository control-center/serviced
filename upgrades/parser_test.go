// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package upgrades

import (
	"strings"
	"testing"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type UpgradeSuite struct{}

var _ = Suite(&UpgradeSuite{})

var descriptor = `
#comment
DESCRIPTION  Zenoss RM 5.0.1 upgrade
VERSION   resmgr-5.0.1
DEPENDENCY 1.1
SNAPSHOT

#comment 2
USE  zenoss/resmgr-stable:5.0.1
USE  zenoss/hbase:v5
SVC_RUN   /zope upgrade
SVC_RUN   /hbase/regionserver upgrade arg1 arg2
`

func (vs *UpgradeSuite) Test_parseDescriptor(t *C) {
	r := strings.NewReader(descriptor)
	ctx, err := parseDescriptor(r)
	t.Assert(err, IsNil)

	use1, _ := createUse("zenoss/resmgr-stable:5.0.1")
	use2, _ := createUse("zenoss/hbase:v5")
	expected := []command{
		emptyCMD,
		comment("#comment"),
		description("Zenoss RM 5.0.1 upgrade"),
		version("resmgr-5.0.1"),
		dependency("1.1"),
		snapshot("SNAPSHOT"),
		emptyCMD,
		comment("#comment 2"),
		use1,
		use2,
		svc_run{"/zope", "upgrade", []string{}},
		svc_run{"/hbase/regionserver", "upgrade", []string{"arg1", "arg2"}},
	}
	t.Assert(len(ctx.commands), Equals, len(expected))

	for i, val := range expected {
		t.Assert(ctx.commands[i], DeepEquals, val)
	}

}

func (vs *UpgradeSuite) Test_parseErrors(t *C) {
	testDescriptor := `
DESCRIPTION  Zenoss RM 5.0.1 upgrade
DESCRIPTION  blam
	`
	r := strings.NewReader(testDescriptor)
	ctx, err := parseDescriptor(r)
	t.Assert(err, IsNil)
	t.Assert(len(ctx.errors), Equals, 1)
	t.Assert(ctx.errors[0].Error(), Equals, "Extra DESCRIPTION at line 3: DESCRIPTION  blam")

}

func (vs *UpgradeSuite) Test_parseLine(t *C) {
	type test struct {
		line string
		cmd  string
		args []string
	}

	tests := []test{
		test{"", "", []string{}},
		test{"   ", "", []string{}},
		test{"\t", "", []string{}},
		test{"cmd", "cmd", []string{}},
		test{"    cmd2   ", "cmd2", []string{}},
		test{"cmd arg1", "cmd", []string{"arg1"}},
		test{"cmd3 arg1 arg2", "cmd3", []string{"arg1", "arg2"}},
		test{"cmd4   arg1  \t  arg2   ", "cmd4", []string{"arg1", "arg2"}},
	}

	for _, testCase := range tests {
		cmd, args := parseLine(testCase.line)
		t.Assert(cmd, Equals, testCase.cmd)
		t.Assert(args, DeepEquals, testCase.args)
	}
}
func (vs *UpgradeSuite) Test_parseCommand(t *C) {
	type test struct {
		line string
		cmd  string
		args []string
	}

	tests := []test{
		test{"", "", []string{}},
		test{"   ", "", []string{}},
		test{"\t", "", []string{}},
		test{"cmd", "cmd", []string{}},
		test{"    cmd2   ", "cmd2", []string{}},
		test{"cmd arg1", "cmd", []string{"arg1"}},
		test{"cmd3 arg1 arg2", "cmd3", []string{"arg1", "arg2"}},
		test{"cmd4   arg1  \t  arg2   ", "cmd4", []string{"arg1", "arg2"}},
	}

	for _, testCase := range tests {
		cmd, args := parseLine(testCase.line)
		t.Assert(cmd, Equals, testCase.cmd)
		t.Assert(args, DeepEquals, testCase.args)
	}
}
