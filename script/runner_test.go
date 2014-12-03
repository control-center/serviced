// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package script

import (
	"errors"
	"fmt"

	. "gopkg.in/check.v1"
)

func (vs *ScriptSuite) Test_Run(t *C) {
	config := Config{
		NoOp:          true,
		ServiceID:     "TEST_SERVICE_ID_12345",
		TenantLookup:  func(service string) (string, error) { return service, nil },
		SvcIDFromPath: func(tenantID string, path string) (string, error) { return tenantID, nil },
	}
	runner, err := NewRunnerFromFile("descriptor_test.txt", &config)
	t.Assert(err, IsNil)
	err = runner.Run()
	t.Assert(err, IsNil)

	runner, err = NewRunnerFromFile("bad_descriptor.txt", &config)
	t.Assert(err, ErrorMatches, "error parsing script")

	config = Config{
		NoOp:      true,
		ServiceID: "TEST_SERVICE_ID_12345",
		TenantLookup: func(service string) (string, error) {
			fmt.Println("BLKDJSFLKSDJFSD")
			return "", errors.New("tenant error for test")
		},
	}
	runner, err = NewRunnerFromFile("descriptor_test.txt", &config)
	t.Assert(err, IsNil)
	err = runner.Run()
	t.Assert(err, ErrorMatches, "tenant error for test")

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
