// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package upgrades

import (
	. "gopkg.in/check.v1"
)

func (vs *UpgradeSuite) Test_emtpy(t *C) {
	ctx := newParseContext()
	cmd, err := newEmtpyCommand(ctx, []string{})
	t.Assert(err, IsNil)
	t.Assert(cmd, Equals, emptyCMD)

	cmd, err = newEmtpyCommand(ctx, []string{"blamo"})
	t.Assert(err, NotNil)

	ctx.line = "sjfskd"
	cmd, err = newEmtpyCommand(ctx, []string{})
	t.Assert(err, NotNil)
}

func (vs *UpgradeSuite) Test_comment(t *C) {
	ctx := newParseContext()
	ctx.line = "#new comment"
	cmd, err := newComment(ctx, []string{"new", "comment"})
	t.Assert(err, IsNil)
	t.Assert(cmd, Equals, comment("#new comment"))

	ctx.line = "    # other comment"
	cmd, err = newComment(ctx, []string{})
	t.Assert(err, IsNil)

	ctx.line = "//bad comment"
	cmd, err = newComment(ctx, []string{})
	t.Assert(err, NotNil)
}

func (vs *UpgradeSuite) Test_description(t *C) {
	ctx := newParseContext()
	ctx.line = "DESCRIPTION new desc"
	cmd, err := newDescription(ctx, []string{"new", "desc"})
	t.Assert(err, IsNil)
	t.Assert(cmd, Equals, description("new desc"))

	ctx.line = "DESCRIPTION"
	cmd, err = newDescription(ctx, []string{})
	t.Assert(err, NotNil)
}

func (vs *UpgradeSuite) Test_dependency(t *C) {
	ctx := newParseContext()
	ctx.line = "DEPENDENCY 1.1"
	cmd, err := newDependency(ctx, []string{"1.1"})
	t.Assert(err, IsNil)
	t.Assert(cmd, Equals, dependency("1.1"))

	ctx.line = "DEPENDENCY 1 1"
	cmd, err = newDependency(ctx, []string{"1", "1"})
	t.Assert(err, NotNil)

	ctx.line = "DEPENDENCY"
	cmd, err = newDependency(ctx, []string{})
	t.Assert(err, NotNil)
}

func (vs *UpgradeSuite) Test_version(t *C) {
	ctx := newParseContext()
	ctx.line = "VERSION 1.1"
	cmd, err := newVersion(ctx, []string{"1.1"})
	t.Assert(err, IsNil)
	t.Assert(cmd, Equals, version("1.1"))

	ctx.line = "VERSION 1 1"
	cmd, err = newVersion(ctx, []string{"1", "1"})
	t.Assert(err, NotNil)

	ctx.line = "VERSION"
	cmd, err = newVersion(ctx, []string{})
	t.Assert(err, NotNil)
}

func (vs *UpgradeSuite) Test_snapshot(t *C) {
	ctx := newParseContext()
	ctx.line = "SNAPSHOT"
	cmd, err := newSnapshot(ctx, []string{})
	t.Assert(err, IsNil)
	t.Assert(cmd, Equals, snapshot("SNAPSHOT"))

	ctx.line = "SNAPSHOT 1"
	cmd, err = newSnapshot(ctx, []string{"1"})
	t.Assert(err, NotNil)
}

func (vs *UpgradeSuite) Test_use(t *C) {
	ctx := newParseContext()
	ctx.line = "USE zenoss/resmgr-stable:5.0.1"
	cmd, err := newUse(ctx, []string{"zenoss/resmgr-stable:5.0.1"})
	t.Assert(err, IsNil)
	expected, _ := createUse("zenoss/resmgr-stable:5.0.1")
	t.Assert(cmd, Equals, expected)

	ctx.line = "USE"
	cmd, err = newUse(ctx, []string{})
	t.Assert(err, NotNil)

	ctx.line = "USE *(^&*blamo::::"
	cmd, err = newUse(ctx, []string{"*(^&*blamo::::"})
	t.Assert(err, NotNil)
	t.Assert(err, ErrorMatches, "invalid ImageID .*")
}

func (vs *UpgradeSuite) Test_svcrun(t *C) {
	ctx := newParseContext()
	ctx.line = "SVC_RUN zope upgrade"
	cmd, err := newSvcRun(ctx, []string{"zope", "upgrade"})
	t.Assert(err, IsNil)
	t.Assert(cmd, DeepEquals, svc_run{"zope", "upgrade", []string{}})

	ctx.line = "SVC_RUN zope upgrade arg"
	cmd, err = newSvcRun(ctx, []string{"zope", "upgrade", "arg"})
	t.Assert(err, IsNil)
	t.Assert(cmd, DeepEquals, svc_run{"zope", "upgrade", []string{"arg"}})

	ctx.line = "SVC_RUN blam"
	cmd, err = newSvcRun(ctx, []string{"blam"})
	t.Assert(err, ErrorMatches, "expected at least two arguments.*")

	ctx.line = "SVC_RUN"
	cmd, err = newSvcRun(ctx, []string{})
	t.Assert(err, ErrorMatches, "expected at least two arguments.*")

}
//	expected := []command{
//		emptyCMD,
//		comment("#comment"),
//		description("Zenoss RM 5.0.1 upgrade"),
//		version("resmgr-5.0.1"),
//		dependency("1.1"),
//		snapshot("SNAPSHOT"),
//		emptyCMD,
//		comment("#comment 2"),
//		use("zenoss/resmgr-stable:5.0.1"),
//		use("zenoss/hbase:v5"),
//		svc_run{"/zope", "upgrade"},
//		svc_run{"/hbase/regionserver", "upgrade"},
//	}
