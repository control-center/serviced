// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package script

import (
	. "gopkg.in/check.v1"
)

func (vs *ScriptSuite) Test_emtpy(t *C) {
	ctx := newParseContext()
	n, err := parseEmtpyCommand(ctx, "", []string{})
	t.Assert(err, IsNil)
	t.Assert(n, DeepEquals, emptyNode)

	ctx.line = "#new comment"
	n, err = parseEmtpyCommand(ctx, "#", []string{"new comment"})
	t.Assert(err, IsNil)
	t.Assert(n, DeepEquals, emptyNode)
}

func (vs *ScriptSuite) Test_description(t *C) {
	ctx := newParseContext()
	line := "DESCRIPTION new desc"
	ctx.line = line
	n, err := parseDescription(ctx, DESCRIPTION, []string{"new", "desc"})
	t.Assert(err, IsNil)
	t.Assert(n, DeepEquals, node{cmd: DESCRIPTION, line: line, args: []string{"new", "desc"}})

	ctx.line = "DESCRIPTION"
	n, err = parseDescription(ctx, DESCRIPTION, []string{})
	t.Assert(err, NotNil)
}

func (vs *ScriptSuite) Test_NoArgs(t *C) {
	ctx := newParseContext()
	ctx.line = SNAPSHOT
	cmd, err := parseNoArgs(ctx, SNAPSHOT, []string{})
	t.Assert(err, IsNil)
	t.Assert(cmd, DeepEquals, node{cmd: SNAPSHOT, line: SNAPSHOT, args: []string{}})

	ctx.line = "SNAPSHOT 1"
	cmd, err = parseNoArgs(ctx, SNAPSHOT, []string{"1"})
	t.Assert(err, NotNil)
}

func (vs *ScriptSuite) Test_OneArg(t *C) {
	ctx := newParseContext()
	line := "DEPENDENCY 1.1"
	ctx.line = line
	cmd, err := parseOneArg(ctx, DEPENDENCY, []string{"1.1"})
	ctx.nodes = append(ctx.nodes, cmd)
	t.Assert(err, IsNil)
	t.Assert(cmd, DeepEquals, node{cmd: DEPENDENCY, line: line, args: []string{"1.1"}})

	ctx.line = "DEPENDENCY 1 1"
	cmd, err = parseOneArg(ctx, DEPENDENCY, []string{"1", "1"})
	t.Assert(err, NotNil)

	ctx.line = "DEPENDENCY"
	cmd, err = parseOneArg(ctx, DEPENDENCY, []string{})
	t.Assert(err, NotNil)
}

func (vs *ScriptSuite) Test_use(t *C) {
	ctx := newParseContext()
	line := "USE zenoss/resmgr-stable:5.0.1"
	ctx.line = line
	cmd, err := parseImageID(ctx, USE, []string{"zenoss/resmgr-stable:5.0.1"})
	t.Assert(err, IsNil)
	t.Assert(cmd, DeepEquals, node{cmd: USE, line: line, args: []string{"zenoss/resmgr-stable:5.0.1"}})

	ctx.line = "USE zenoss/resmgr-stable:5.0.1 blam"
	cmd, err = parseImageID(ctx, USE, []string{})
	t.Assert(err, NotNil)
	t.Assert(err, ErrorMatches, "line 0: expected one argument, got: USE zenoss/resmgr-stable:5.0.1 blam")

	ctx.line = "USE"
	cmd, err = parseImageID(ctx, USE, []string{})
	t.Assert(err, NotNil)
	t.Assert(err, ErrorMatches, "line 0: expected one argument, got: USE")

	ctx.line = "USE *(^&*blamo::::"
	cmd, err = parseImageID(ctx, USE, []string{"*(^&*blamo::::"})
	t.Assert(err, NotNil)
	t.Assert(err, ErrorMatches, "invalid ImageID .*")
}

func (vs *ScriptSuite) Test_svcrun(t *C) {
	ctx := newParseContext()
	line := "SVC_RUN zope upgrade"
	ctx.line = line
	cmd, err := parseSvcRun(ctx, SVC_RUN, []string{"zope", "upgrade"})
	t.Assert(err, IsNil)
	t.Assert(cmd, DeepEquals, node{cmd: SVC_RUN, line: line, args: []string{"zope", "upgrade"}})

	line = "SVC_RUN zope upgrade arg"
	ctx.line = line
	cmd, err = parseSvcRun(ctx, SVC_RUN, []string{"zope", "upgrade", "arg"})
	t.Assert(err, IsNil)
	t.Assert(cmd, DeepEquals, node{cmd: SVC_RUN, line: line, args: []string{"zope", "upgrade", "arg"}})

	ctx.line = "SVC_RUN blam"
	cmd, err = parseSvcRun(ctx, SVC_RUN, []string{"blam"})
	t.Assert(err, ErrorMatches, "line 0: expected at least two arguments, got: SVC_RUN blam")

	ctx.line = "SVC_RUN"
	cmd, err = parseSvcRun(ctx, SVC_RUN, []string{})
	t.Assert(err, ErrorMatches, "line 0: expected at least two arguments, got: SVC_RUN")

}
