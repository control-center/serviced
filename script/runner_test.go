// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package script

import (
	. "gopkg.in/check.v1"
)

func (vs *ScriptSuite) Test_evalNodes(t *C) {
	//	pullImage = testPullSucceed
	//	findImage = testFindImageSucceed
	//	tagImage = testTagSucceed
	//	execCommand = testExec
	config := Config{NoOp: true, ServiceID: "TEST_SERVICE_ID_12345"}
	runner, err := NewRunnerFromFile("descriptor_test.txt", config)
	t.Assert(err, IsNil)
	err = runner.Run()
	t.Assert(err, IsNil)
}

//func testExec(cmd string, args ...string) error {
//	return nil
//}
//func testTagSucceed(image *docker.Image, newTag string) (*docker.Image, error) {
//	return image, nil
//}
//func testPullSucceed(image string) error {
//	return nil
//}
//
//func testFindImageSucceed(image string, pull bool) (*docker.Image, error) {
//	id, err := commons.ParseImageID(image)
//	if err != nil {
//		return nil, err
//	}
//	return &docker.Image{"123456789", *id}, nil
//}
