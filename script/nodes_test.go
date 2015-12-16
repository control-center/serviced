// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

// +build unit

package script

import (
	"fmt"
	"regexp"
	"strings"

	. "gopkg.in/check.v1"
)

func (vs *ScriptSuite) Test_empty(t *C) {
	ctx := newParseContext()
	n, err := parseEmptyCommand(ctx, "", []string{})
	t.Assert(err, IsNil)
	t.Assert(n, DeepEquals, emptyNode)

	ctx.line = "#new comment"
	n, err = parseEmptyCommand(ctx, "#", []string{"new comment"})
	t.Assert(err, IsNil)
	t.Assert(n, DeepEquals, emptyNode)
}

func (vs *ScriptSuite) Test_description(t *C) {
	ctx := newParseContext()
	line := "DESCRIPTION new desc"
	ctx.line = line
	n, err := nodeFactories[DESCRIPTION](ctx, DESCRIPTION, []string{"new", "desc"})
	t.Assert(err, IsNil)
	t.Assert(n, DeepEquals, node{cmd: DESCRIPTION, line: line, args: []string{"new", "desc"}})

	ctx.line = "DESCRIPTION"
	n, err = nodeFactories[DESCRIPTION](ctx, DESCRIPTION, []string{})
	t.Assert(err, NotNil)
}

func (vs *ScriptSuite) Test_NoArgs(t *C) {
	ctx := newParseContext()
	ctx.line = REQUIRE_SVC
	cmd, err := nodeFactories[REQUIRE_SVC](ctx, REQUIRE_SVC, []string{})
	t.Assert(err, IsNil)
	t.Assert(cmd, DeepEquals, node{cmd: REQUIRE_SVC, line: REQUIRE_SVC, args: []string{}})

	ctx.line = "REQUIRE_SVC 1"
	cmd, err = nodeFactories[REQUIRE_SVC](ctx, REQUIRE_SVC, []string{"1"})
	t.Assert(err, NotNil)
}

func (vs *ScriptSuite) Test_OneArg(t *C) {
	ctx := newParseContext()
	line := "DEPENDENCY 1.1"
	ctx.line = line
	cmd, err := nodeFactories[DEPENDENCY](ctx, DEPENDENCY, []string{"1.1"})
	ctx.nodes = append(ctx.nodes, cmd)
	t.Assert(err, IsNil)
	t.Assert(cmd, DeepEquals, node{cmd: DEPENDENCY, line: line, args: []string{"1.1"}})

	ctx.line = "DEPENDENCY 1 1"
	cmd, err = nodeFactories[DEPENDENCY](ctx, DEPENDENCY, []string{"1", "1"})
	t.Assert(err, NotNil)

	ctx.line = "DEPENDENCY"
	cmd, err = nodeFactories[DEPENDENCY](ctx, DEPENDENCY, []string{})
	t.Assert(err, NotNil)
}

func (vs *ScriptSuite) Test_snapshot(t *C) {
	ctx := newParseContext()
	ctx.line = SNAPSHOT
	cmd, err := nodeFactories[SNAPSHOT](ctx, SNAPSHOT, []string{})
	t.Assert(err, IsNil)
	t.Assert(cmd, DeepEquals, node{cmd: SNAPSHOT, line: SNAPSHOT, args: []string{}})

	ctx.line = "SNAPSHOT 1"
	cmd, err = nodeFactories[SNAPSHOT](ctx, SNAPSHOT, []string{"1"})
	t.Assert(err, IsNil)
	t.Assert(cmd, DeepEquals, node{cmd: SNAPSHOT, line: ctx.line, args: []string{"1"}})

	ctx.line = "SNAPSHOT 1 1"
	cmd, err = nodeFactories[SNAPSHOT](ctx, SNAPSHOT, []string{"1", "1"})
	t.Assert(err, NotNil)
}

func (vs *ScriptSuite) Test_use(t *C) {
	ctx := newParseContext()
	line := "USE zenoss/resmgr-stable:5.0.1"
	ctx.line = line
	cmd, err := nodeFactories[USE](ctx, USE, []string{"zenoss/resmgr-stable:5.0.1"})
	t.Assert(err, IsNil)
	t.Assert(cmd, DeepEquals, node{cmd: USE, line: line, args: []string{"zenoss/resmgr-stable:5.0.1"}})

	ctx.line = "USE zenoss/resmgr_5.0:5.0.4"
	cmd, err = nodeFactories[USE](ctx, USE, []string{"zenoss/resmgr_5.0:5.0.4"})
	t.Assert(err, IsNil)
	t.Assert(cmd, DeepEquals, node{cmd: USE, line: ctx.line, args: []string{"zenoss/resmgr_5.0:5.0.4"}})

	ctx.line = "USE zenoss/resmgr-stable:5.0.1 blam blammy"
	cmd, err = nodeFactories[USE](ctx, USE, []string{"zenoss/resmgr-stable:5.0.1", "blam", "blammy"})
	t.Assert(err, IsNil)

	ctx.line = "USE"
	cmd, err = nodeFactories[USE](ctx, USE, []string{})
	t.Assert(err, NotNil)
	t.Assert(err, ErrorMatches, "line 0: expected at least 1, got 0: USE")

	ctx.line = "USE *(^&*blamo::::"
	cmd, err = nodeFactories[USE](ctx, USE, []string{"*(^&*blamo::::"})
	t.Assert(err, NotNil)
	t.Assert(err, ErrorMatches, "invalid ImageID .*")

	ctx.line = "USE zenoss/resmgr_5.1 zenoss/newRepo"
	cmd, err = nodeFactories[USE](ctx, USE, []string{"zenoss/resmgr_5.1", "zenoss/newRepo"})
	t.Assert(err, IsNil)

	ctx.line = "USE zenoss/resmgr_5.1 zenoss/newRepo:tag"
	cmd, err = nodeFactories[USE](ctx, USE, []string{"zenoss/resmgr_5.1", "zenoss/newRepo:tag"})
	t.Assert(err, NotNil)
	t.Assert(err, ErrorMatches, "image string zenoss/newRepo:tag should only specify a repo")

	ctx.line = "USE zenoss/resmgr_5.1 localhost:5000/zenoss/newRepo"
	cmd, err = nodeFactories[USE](ctx, USE, []string{"zenoss/resmgr_5.1", "localhost:5000/zenoss/newRepo"})
	t.Assert(err, NotNil)
	t.Assert(err, ErrorMatches, "image string localhost:5000/zenoss/newRepo should only specify a repo")

	ctx.line = "USE zenoss/resmgr_5.1 zenoss/newRepo secondRepo"
	cmd, err = nodeFactories[USE](ctx, USE, []string{"zenoss/resmgr_5.1", "zenoss/newRepo", "secondRepo"})
	t.Assert(err, IsNil)

	ctx.line = "USE zenoss/resmgr_5.1 zenoss/newRepo secondRepo:tag"
	cmd, err = nodeFactories[USE](ctx, USE, []string{"zenoss/resmgr_5.1", "zenoss/newRepo", "secondRepo:tag"})
	t.Assert(err, NotNil)
	t.Assert(err, ErrorMatches, "image string secondRepo\\:tag should only specify a repo")
}

func (vs *ScriptSuite) Test_svcrun(t *C) {
	ctx := newParseContext()
	line := "SVC_RUN zope upgrade"
	ctx.line = line
	cmd, err := nodeFactories[SVC_RUN](ctx, SVC_RUN, []string{"zope", "upgrade"})
	t.Assert(err, IsNil)
	t.Assert(cmd, DeepEquals, node{cmd: SVC_RUN, line: line, args: []string{"zope", "upgrade"}})

	line = "SVC_RUN zope upgrade arg"
	ctx.line = line
	cmd, err = nodeFactories[SVC_RUN](ctx, SVC_RUN, []string{"zope", "upgrade", "arg"})
	t.Assert(err, IsNil)
	t.Assert(cmd, DeepEquals, node{cmd: SVC_RUN, line: line, args: []string{"zope", "upgrade", "arg"}})

	ctx.line = "SVC_RUN blam"
	cmd, err = nodeFactories[SVC_RUN](ctx, SVC_RUN, []string{"blam"})
	t.Assert(err, ErrorMatches, "line 0: expected at least 2, got 1: SVC_RUN blam")

	ctx.line = "SVC_RUN"
	cmd, err = nodeFactories[SVC_RUN](ctx, SVC_RUN, []string{})
	t.Assert(err, ErrorMatches, "line 0: expected at least 2, got 0: SVC_RUN")

}

func (vs *ScriptSuite) Test_svcWait(t *C) {
	ctx := newParseContext()
	line := "SVC_WAIT zope started"
	ctx.line = line
	cmd, err := nodeFactories[SVC_WAIT](ctx, SVC_WAIT, []string{"zope", "started"})
	t.Assert(err, IsNil)
	t.Assert(cmd, DeepEquals, node{cmd: SVC_WAIT, line: line, args: []string{"zope", "started"}})

	line = "SVC_WAIT zope stopped"
	ctx.line = line
	cmd, err = nodeFactories[SVC_WAIT](ctx, SVC_WAIT, []string{"zope", "stopped"})
	t.Assert(err, IsNil)
	t.Assert(cmd, DeepEquals, node{cmd: SVC_WAIT, line: line, args: []string{"zope", "stopped"}})

	line = "SVC_WAIT zope paused"
	ctx.line = line
	cmd, err = nodeFactories[SVC_WAIT](ctx, SVC_WAIT, []string{"zope", "paused"})
	t.Assert(err, IsNil)
	t.Assert(cmd, DeepEquals, node{cmd: SVC_WAIT, line: line, args: []string{"zope", "paused"}})

	line = "SVC_WAIT"
	ctx.line = line
	cmd, err = nodeFactories[SVC_WAIT](ctx, SVC_WAIT, []string{})
	t.Assert(err, NotNil)

	ctx.line = "SVC_WAIT zope"
	cmd, err = nodeFactories[SVC_WAIT](ctx, SVC_WAIT, []string{"zope"})
	t.Assert(err, NotNil)

	ctx.line = "SVC_WAIT zope garbage"
	cmd, err = nodeFactories[SVC_WAIT](ctx, SVC_WAIT, []string{"zope", "garbage"})
	t.Assert(err, NotNil)

	ctx.line = "SVC_WAIT zope started extra"
	cmd, err = nodeFactories[SVC_WAIT](ctx, SVC_WAIT, []string{"zope", "started", "extra"})
	t.Assert(err, NotNil)

	line = "SVC_WAIT zope started 0"
	ctx.line = line
	cmd, err = nodeFactories[SVC_WAIT](ctx, SVC_WAIT, []string{"zope", "started", "0"})
	t.Assert(err, IsNil)
	t.Assert(cmd, DeepEquals, node{cmd: SVC_WAIT, line: line, args: []string{"zope", "started", "0"}})

	ctx.line = "SVC_WAIT zope started -1"
	cmd, err = nodeFactories[SVC_WAIT](ctx, SVC_WAIT, []string{"zope", "started", "-1"})
	t.Assert(err, NotNil)

	ctx.line = "SVC_WAIT zope started extra"
	cmd, err = nodeFactories[SVC_WAIT](ctx, SVC_WAIT, []string{"zope", "started", "extra"})
	t.Assert(err, NotNil)

	ctx.line = "SVC_WAIT zope started 4 extra"
	cmd, err = nodeFactories[SVC_WAIT](ctx, SVC_WAIT, []string{"zope", "started", "4", "extra"})
	t.Assert(err, NotNil)

	ctx.line = "SVC_WAIT zope started 4 recursive"
	cmd, err = nodeFactories[SVC_WAIT](ctx, SVC_WAIT, []string{"zope", "started", "4", "recursive"})
	t.Assert(err, IsNil)

	ctx.line = "SVC_WAIT zope started recursive"
	cmd, err = nodeFactories[SVC_WAIT](ctx, SVC_WAIT, []string{"zope", "started", "recursive"})
	t.Assert(err, IsNil)

	ctx.line = "SVC_WAIT zope started recursive 5"
	cmd, err = nodeFactories[SVC_WAIT](ctx, SVC_WAIT, []string{"zope", "started", "recursive", "5"})
	t.Assert(err, NotNil)

	ctx.line = "SVC_WAIT zope started 4 garbage"
	cmd, err = nodeFactories[SVC_WAIT](ctx, SVC_WAIT, []string{"zope", "started", "4", "garbage"})
	t.Assert(err, NotNil)

	line = "SVC_WAIT zope mariadb started 0"
	ctx.line = line
	cmd, err = nodeFactories[SVC_WAIT](ctx, SVC_WAIT, strings.Split(line, " "))
	t.Assert(err, IsNil)
	t.Assert(cmd, DeepEquals, node{cmd: SVC_WAIT, line: line, args: strings.Split(line, " ")})

	line = "SVC_WAIT zope mariadb started"
	ctx.line = line
	cmd, err = nodeFactories[SVC_WAIT](ctx, SVC_WAIT, strings.Split(line, " "))
	t.Assert(err, IsNil)
	t.Assert(cmd, DeepEquals, node{cmd: SVC_WAIT, line: line, args: strings.Split(line, " ")})

	line = "SVC_WAIT zope mariadb started extra"
	ctx.line = line
	cmd, err = nodeFactories[SVC_WAIT](ctx, SVC_WAIT, strings.Split(line, " "))
	t.Assert(err, NotNil)

	ctx.line = "SVC_WAIT zope mariadb started extra recursive"
	cmd, err = nodeFactories[SVC_WAIT](ctx, SVC_WAIT, strings.Split(line, " "))
	t.Assert(err, NotNil)
}

func (vs *ScriptSuite) Test_svcexec(t *C) {
	ctx := newParseContext()
	line := "SVC_EXEC NO_COMMIT zope ls -al"
	ctx.line = line
	cmd, err := nodeFactories[SVC_EXEC](ctx, SVC_EXEC, []string{"NO_COMMIT", "zope", "ls", "-al"})
	t.Assert(err, IsNil)
	t.Assert(cmd, DeepEquals, node{cmd: SVC_EXEC, line: line, args: []string{"NO_COMMIT", "zope", "ls", "-al"}})

	line = "SVC_EXEC garbage zope ls -al"
	ctx.line = line
	cmd, err = nodeFactories[SVC_EXEC](ctx, SVC_EXEC, []string{"garbage", "zope", "ls", "-al"})
	t.Assert(err, ErrorMatches, regexp.QuoteMeta("line 0: arg garbage did not match ^(NO_)?COMMIT$"))

	line = "SVC_EXEC COMMIT zope"
	ctx.line = line
	cmd, err = nodeFactories[SVC_EXEC](ctx, SVC_EXEC, []string{"COMMIT", "zope"})
	t.Assert(err, ErrorMatches, regexp.QuoteMeta("line 0: expected at least 3, got 2: SVC_EXEC COMMIT zope"))

	ctx.line = "SVC_EXEC"
	cmd, err = nodeFactories[SVC_EXEC](ctx, SVC_EXEC, []string{})
	t.Assert(err, ErrorMatches, "line 0: expected at least 3, got 0: SVC_EXEC")
}

func (vs *ScriptSuite) Test_svcStartStopRestart(t *C) {

	cmds := []string{SVC_START, SVC_STOP, SVC_RESTART}

	for _, cmd := range cmds {
		ctx := newParseContext()
		line := fmt.Sprintf("%s zope", cmd)
		ctx.line = line
		n, err := nodeFactories[cmd](ctx, cmd, []string{"zope"})
		t.Assert(err, IsNil)
		t.Assert(n, DeepEquals, node{cmd: cmd, line: line, args: []string{"zope"}})

		line = fmt.Sprintf("%s zope auto", cmd)
		ctx.line = line
		n, err = nodeFactories[cmd](ctx, cmd, []string{"zope", "auto"})
		t.Assert(err, IsNil)
		t.Assert(n, DeepEquals, node{cmd: cmd, line: line, args: []string{"zope", "auto"}})

		line = fmt.Sprintf("%s zope wait", cmd)
		ctx.line = line
		n, err = nodeFactories[cmd](ctx, cmd, []string{"zope", "wait"})
		t.Assert(err, NotNil)

		line = fmt.Sprintf("%s zope recurse blam", cmd)
		ctx.line = line
		n, err = nodeFactories[cmd](ctx, cmd, []string{"zope", "recurse", "blam"})
		t.Assert(err, NotNil)

	}

}
