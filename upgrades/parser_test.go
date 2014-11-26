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

func (vs *UpgradeSuite) Test_parseFile(t *C) {
	ctx, err := parseFile("descriptor_test.txt")
	t.Assert(err, IsNil)

	ctx.line = "USE  zenoss/resmgr-stable:5.0.1"
	ctx.lineNum = 9
	use1, _ := parseImageID(ctx, "USE", []string{"zenoss/resmgr-stable:5.0.1"})

	ctx.line = "USE  zenoss/hbase:v5"
	ctx.lineNum = 10
	use2, _ := parseImageID(ctx, "USE", []string{"zenoss/hbase:v5"})
	expected := []node{
		node{lineNum: 3, cmd: DESCRIPTION, args: []string{"Zenoss", "RM", "5.0.1", "upgrade"}, line: "DESCRIPTION  Zenoss RM 5.0.1 upgrade"},
		node{lineNum: 4, cmd: VERSION, args: []string{"resmgr-5.0.1"}, line: "VERSION   resmgr-5.0.1"},
		node{lineNum: 5, cmd: DEPENDENCY, args: []string{"1.1"}, line: "DEPENDENCY 1.1"},
		node{lineNum: 6, cmd: SNAPSHOT, line: "SNAPSHOT", args: []string{}},
		use1,
		use2,
		node{lineNum: 11, cmd: SVC_RUN, line: "SVC_RUN   /zope upgrade", args: []string{"/zope", "upgrade"}},
		node{lineNum: 12, cmd: SVC_RUN, line: "SVC_RUN   /hbase/regionserver upgrade arg1 arg2", args: []string{"/hbase/regionserver", "upgrade", "arg1", "arg2"}},
	}
	t.Assert(len(ctx.nodes), Equals, len(expected))

	for i, val := range expected {
		t.Assert(ctx.nodes[i], DeepEquals, val)
	}

}

func (vs *UpgradeSuite) Test_parseDescriptor(t *C) {
	r := strings.NewReader(descriptor)
	ctx, err := parseDescriptor(r)
	t.Assert(err, IsNil)

	ctx.line = "USE  zenoss/resmgr-stable:5.0.1"
	ctx.lineNum = 9
	use1, _ := parseImageID(ctx, "USE", []string{"zenoss/resmgr-stable:5.0.1"})

	ctx.line = "USE  zenoss/hbase:v5"
	ctx.lineNum = 10
	use2, _ := parseImageID(ctx, "USE", []string{"zenoss/hbase:v5"})
	expected := []node{
		node{lineNum: 3, cmd: DESCRIPTION, args: []string{"Zenoss", "RM", "5.0.1", "upgrade"}, line: "DESCRIPTION  Zenoss RM 5.0.1 upgrade"},
		node{lineNum: 4, cmd: VERSION, args: []string{"resmgr-5.0.1"}, line: "VERSION   resmgr-5.0.1"},
		node{lineNum: 5, cmd: DEPENDENCY, args: []string{"1.1"}, line: "DEPENDENCY 1.1"},
		node{lineNum: 6, cmd: SNAPSHOT, line: "SNAPSHOT", args: []string{}},
		use1,
		use2,
		node{lineNum: 11, cmd: SVC_RUN, line: "SVC_RUN   /zope upgrade", args: []string{"/zope", "upgrade"}},
		node{lineNum: 12, cmd: SVC_RUN, line: "SVC_RUN   /hbase/regionserver upgrade arg1 arg2", args: []string{"/hbase/regionserver", "upgrade", "arg1", "arg2"}},
	}
	t.Assert(len(ctx.nodes), Equals, len(expected))

	for i, val := range expected {
		t.Assert(ctx.nodes[i], DeepEquals, val)
	}

}

func (vs *UpgradeSuite) Test_parseErrors(t *C) {
	testDescriptor := `
DESCRIPTION  Zenoss RM 5.0.1 upgrade
DESCRIPTION  blam
USE blam
SNAPSHOT
SVC_RUN blam foo
#DEPENDENCY cannot appear after USE, SVC_RUN, SNAPSHOT
DEPENDENCY 1.1
	`
	r := strings.NewReader(testDescriptor)
	ctx, err := parseDescriptor(r)
	t.Assert(err, IsNil)
	t.Assert(len(ctx.errors), Equals, 4)
	t.Assert(ctx.errors[0], ErrorMatches, "line 3: extra DESCRIPTION: DESCRIPTION  blam")
	t.Assert(ctx.errors[1], ErrorMatches, "line 8: DEPENDENCY must be declared before USE")
	t.Assert(ctx.errors[2], ErrorMatches, "line 8: DEPENDENCY must be declared before SNAPSHOT")
	t.Assert(ctx.errors[3], ErrorMatches, "line 8: DEPENDENCY must be declared before SVC_RUN")

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
