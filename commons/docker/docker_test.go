// Copyright 2016 The Serviced Authors.
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

// +build integration,!quick

package docker

import (
	"testing"

	. "gopkg.in/check.v1"
)

type TestDockerSuite struct {
	// add suite-specific data here such as mocks
}

// verify TestDockerSuite implements the Suite interface
var _ = Suite(&TestDockerSuite{})

// Wire gocheck into the go test runner
func TestDockerSync(t *testing.T) { TestingT(t) }
