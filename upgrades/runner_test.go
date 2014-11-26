// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package upgrades

import (
	"bufio"
	"os"

	. "gopkg.in/check.v1"
)

func (vs *UpgradeSuite) Test_evalNodes(t *C) {
	f, err := os.Open("descriptor_test.txt")
	t.Assert(err, IsNil)
	r := bufio.NewReader(f)
	pctx, err := parseDescriptor(r)
	t.Assert(err, IsNil)

	run := &runner{}
	err = run.evalNodes(pctx.nodes)
	t.Assert(err, IsNil)

}
