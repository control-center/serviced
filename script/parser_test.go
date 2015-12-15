// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

// +build unit

package script

import (
	"strings"
	"testing"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type ScriptSuite struct{}

var _ = Suite(&ScriptSuite{})

var descriptor = `
#comment
DESCRIPTION  Zenoss RM 5.0.1 upgrade
VERSION   resmgr-5.0.1
DEPENDENCY 1.1
SNAPSHOT

#comment 2
SVC_USE  zenoss/resmgr_5.0:5.0.1
SVC_USE  zenoss/hbase:v5
SVC_RUN   /zope upgrade
SVC_RUN   /hbase/regionserver upgrade arg1 arg2
`

func (vs *ScriptSuite) Test_parseFile(t *C) {
	ctx, err := parseFile("descriptor_test.txt")
	t.Assert(err, IsNil)

	ctx.line = "SVC_USE  zenoss/resmgr_5.0:5.0.1"
	ctx.lineNum = 10
	use1, _ := nodeFactories[USE](ctx, USE, []string{"zenoss/resmgr_5.0:5.0.1"})

	ctx.line = "SVC_USE  zenoss/hbase:v5"
	ctx.lineNum = 11
	use2, _ := nodeFactories[USE](ctx, USE, []string{"zenoss/hbase:v5"})

	ctx.line = "SVC_USE  zenoss/testy:5.5 zenoss/old_testy"
	ctx.lineNum = 12
	use3, _ := nodeFactories[USE](ctx, USE, []string{"zenoss/testy:5.5", "zenoss/old_testy"})

	ctx.line = "SVC_USE  zenoss/multi_replace:6.7 replace_me org/replace_me_too"
	ctx.lineNum = 13
	use4, _ := nodeFactories[USE](ctx, USE, []string{"zenoss/multi_replace:6.7", "replace_me", "org/replace_me_too"})

	expected := []node{
		node{lineNum: 3, cmd: DESCRIPTION, args: []string{"Zenoss", "RM", "5.0.1", "upgrade"}, line: "DESCRIPTION  Zenoss RM 5.0.1 upgrade"},
		node{lineNum: 4, cmd: VERSION, args: []string{"resmgr-5.0.1"}, line: "VERSION   resmgr-5.0.1"},
		node{lineNum: 5, cmd: DEPENDENCY, args: []string{"1.1"}, line: "DEPENDENCY 1.1"},
		node{lineNum: 6, cmd: REQUIRE_SVC, line: REQUIRE_SVC, args: []string{}},
		node{lineNum: 7, cmd: SNAPSHOT, line: SNAPSHOT, args: []string{}},
		use1,
		use2,
		use3,
		use4,
		node{lineNum: 14, cmd: SVC_START, line: "SVC_START Zenoss.core/mariadb", args: []string{"Zenoss.core/mariadb"}},
		node{lineNum: 15, cmd: SVC_WAIT, line: "SVC_WAIT Zenoss.core/mariadb started 30", args: []string{"Zenoss.core/mariadb", "started", "30"}},
		node{lineNum: 16, cmd: SVC_STOP, line: "SVC_STOP Zenoss.core/mariadb", args: []string{"Zenoss.core/mariadb"}},
		node{lineNum: 17, cmd: SVC_WAIT, line: "SVC_WAIT Zenoss.core/mariadb stopped 0", args: []string{"Zenoss.core/mariadb", "stopped", "0"}},
		node{lineNum: 18, cmd: SVC_START, line: "SVC_START Zenoss.core/mariadb", args: []string{"Zenoss.core/mariadb"}},
		node{lineNum: 19, cmd: SVC_WAIT, line: "SVC_WAIT Zenoss.core/mariadb started 30", args: []string{"Zenoss.core/mariadb", "started", "30"}},
		node{lineNum: 20, cmd: SVC_RESTART, line: "SVC_RESTART Zenoss.core/mariadb", args: []string{"Zenoss.core/mariadb"}},
		node{lineNum: 21, cmd: SVC_WAIT, line: "SVC_WAIT Zenoss.core/mariadb started 30", args: []string{"Zenoss.core/mariadb", "started", "30"}},
		node{lineNum: 22, cmd: SVC_RUN, line: "SVC_RUN  Zenoss.core/Zope upgrade", args: []string{"Zenoss.core/Zope", "upgrade"}},
		node{lineNum: 23, cmd: SVC_RUN, line: "SVC_RUN  Zenoss.core/HBase/RegionServer upgrade arg1 arg2", args: []string{"Zenoss.core/HBase/RegionServer", "upgrade", "arg1", "arg2"}},
		node{lineNum: 24, cmd: SVC_EXEC, line: "SVC_EXEC COMMIT Zenoss.core/Zope command1", args: []string{"COMMIT", "Zenoss.core/Zope", "command1"}},
		node{lineNum: 25, cmd: SVC_EXEC, line: "SVC_EXEC NO_COMMIT Zenoss.core/zenhub command2 with args", args: []string{"NO_COMMIT", "Zenoss.core/zenhub", "command2", "with", "args"}},
		node{lineNum: 26, cmd: SVC_START, line: "SVC_START Zenoss.core", args: []string{"Zenoss.core"}},
		node{lineNum: 27, cmd: SVC_WAIT, line: "SVC_WAIT Zenoss.core started recursive", args: []string{"Zenoss.core", "started", "recursive"}},
	}
	t.Assert(len(ctx.nodes), Equals, len(expected))

	for i, val := range expected {
		t.Assert(ctx.nodes[i], DeepEquals, val)
	}

}

func (vs *ScriptSuite) Test_parseDescriptor(t *C) {
	r := strings.NewReader(descriptor)
	ctx, err := parseDescriptor(r)
	t.Assert(err, IsNil)

	ctx.line = "SVC_USE  zenoss/resmgr_5.0:5.0.1"
	ctx.lineNum = 9
	use1, _ := nodeFactories[USE](ctx, USE, []string{"zenoss/resmgr_5.0:5.0.1"})

	ctx.line = "SVC_USE  zenoss/hbase:v5"
	ctx.lineNum = 10
	use2, _ := nodeFactories[USE](ctx, USE, []string{"zenoss/hbase:v5"})
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

func (vs *ScriptSuite) Test_parseErrors(t *C) {
	testDescriptor := `
DESCRIPTION  Zenoss RM 5.0.1 upgrade
DESCRIPTION  blam
SVC_USE blam
SNAPSHOT
SVC_RUN blam foo
#DEPENDENCY cannot appear after USE, SVC_RUN, SNAPSHOT
DEPENDENCY 1.1
	`
	r := strings.NewReader(testDescriptor)
	ctx, err := parseDescriptor(r)
	t.Assert(err, IsNil)
	t.Assert(len(ctx.errors), Equals, 7)
	t.Assert(ctx.errors[0], ErrorMatches, "line 3: extra DESCRIPTION: DESCRIPTION  blam")
	t.Assert(ctx.errors[1], ErrorMatches, "line 4: SVC_USE depends on REQUIRE_SVC")
	t.Assert(ctx.errors[2], ErrorMatches, "line 5: SNAPSHOT depends on REQUIRE_SVC")
	t.Assert(ctx.errors[3], ErrorMatches, "line 6: SVC_RUN depends on REQUIRE_SVC")
	t.Assert(ctx.errors[4], ErrorMatches, "line 8: DEPENDENCY must be declared before SVC_USE")
	t.Assert(ctx.errors[5], ErrorMatches, "line 8: DEPENDENCY must be declared before SNAPSHOT")
	t.Assert(ctx.errors[6], ErrorMatches, "line 8: DEPENDENCY must be declared before SVC_RUN")

}

func (vs *ScriptSuite) Test_parseLine(t *C) {
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
		test{"cmd5 arg1 \"arg2a arg2b\" arg3", "cmd5", []string{"arg1", "arg2a arg2b", "arg3"}},
		test{"cmd6 arg1 'arg2a arg2b' arg3", "cmd6", []string{"arg1", "arg2a arg2b", "arg3"}},
	}

	for _, testCase := range tests {
		cmd, args, err := parseLine(testCase.line)

		t.Assert(err, IsNil)
		t.Assert(cmd, Equals, testCase.cmd)
		t.Assert(args, DeepEquals, testCase.args)
	}
}

func (vs *ScriptSuite) Test_parseBadLines(t *C) {
	tests := []string{
		"cmd1 \"",
		"cmd1 '",
		"cmd1 \"arg",
		"cmd1 'arg",
		"cmd1 arg\"",
		"cmd1 arg'",
		"cmd1 arg1 'ar g2' 'arg3",
		"cmd1 arg1 'ar g2' arg3'",
		"cmd1 'arg1 arg2\"",
		"cmd1 \"arg1 arg2'",
	}

	for _, testCase := range tests {
		cmd, args, err := parseLine(testCase)

		t.Assert(cmd, Equals, "")
		t.Assert(len(args), Equals, 0)
		t.Assert(err, ErrorMatches, "invalid command line string")
	}
}
