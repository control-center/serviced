// Copyright 2014 The Serviced Authors.
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

// +build unit

package commons

import (
	"testing"

	. "gopkg.in/check.v1"
)

type TestCommonsSuite struct{
	// add suite-specific data here such as mocks
}

// verify TestGoofySuite implements the Suite interface
var _ = Suite(&TestCommonsSuite{})

// Wire gocheck into the go test runner
func TestCommons(t *testing.T) { TestingT(t) }

type ImageIDTest struct {
	in        string
	out       *ImageID
	roundtrip string
}

var imgidtests = []ImageIDTest{
	// no host
	{
		"dobbs/sierramadre",
		&ImageID{
			User: "dobbs",
			Repo: "sierramadre",
		},
		"",
	},
	// no host, underscore in repo
	{
		"dobbs/sierra_madre",
		&ImageID{
			User: "dobbs",
			Repo: "sierra_madre",
		},
		"",
	},
	// no host top-level
	{
		"sierramadre",
		&ImageID{
			Repo: "sierramadre",
		},
		"",
	},
	// no host top-level underscore in repo
	{
		"sierra_madre",
		&ImageID{
			Repo: "sierra_madre",
		},
		"",
	},
	// no host tagged
	{
		"dobbs/sierramadre:1925",
		&ImageID{
			User: "dobbs",
			Repo: "sierramadre",
			Tag:  "1925",
		},
		"",
	},
	// no host top-level tagged
	{
		"sierramadre:1925",
		&ImageID{
			Repo: "sierramadre",
			Tag:  "1925",
		},
		"",
	},
	// host top-level
	{
		"warner.bros/sierramadre",
		&ImageID{
			Host: "warner.bros",
			Repo: "sierramadre",
		},
		"",
	},
	// host top-level tagged
	{
		"warner.bros/sierramadre:1925",
		&ImageID{
			Host: "warner.bros",
			Repo: "sierramadre",
			Tag:  "1925",
		},
		"",
	},
	// host:port top-level
	{
		"warner.bros:1948/sierramadre",
		&ImageID{
			Host: "warner.bros",
			Port: 1948,
			Repo: "sierramadre",
		},
		"",
	},
	// host:port top-level tagged
	{
		"warner.bros:1948/sierramadre:1925",
		&ImageID{
			Host: "warner.bros",
			Port: 1948,
			Repo: "sierramadre",
			Tag:  "1925",
		},
		"",
	},
	// host
	{
		"warner.bros/dobbs/sierramadre",
		&ImageID{
			Host: "warner.bros",
			User: "dobbs",
			Repo: "sierramadre",
		},
		"",
	},
	// short host
	{
		"warner/dobbs/sierramadre",
		&ImageID{
			Host: "warner",
			User: "dobbs",
			Repo: "sierramadre",
		},
		"",
	},
	// host tagged
	{
		"warner.bros/dobbs/sierramadre:1925",
		&ImageID{
			Host: "warner.bros",
			User: "dobbs",
			Repo: "sierramadre",
			Tag:  "1925",
		},
		"",
	},
	// host:port
	{
		"warner.bros:1948/dobbs/sierramadre",
		&ImageID{
			Host: "warner.bros",
			Port: 1948,
			User: "dobbs",
			Repo: "sierramadre",
		},
		"",
	},
	// host:port tagged
	{
		"warner.bros:1948/dobbs/sierramadre:1925",
		&ImageID{
			Host: "warner.bros",
			Port: 1948,
			User: "dobbs",
			Repo: "sierramadre",
			Tag:  "1925",
		},
		"",
	},
	// short hostname:port uuid tag
	{
		"warner:1948/sierramadre:543c56d1-2510-cd37-c0f4-cab544df985d",
		&ImageID{
			Host: "warner",
			Port: 1948,
			Repo: "sierramadre",
			Tag:  "543c56d1-2510-cd37-c0f4-cab544df985d",
		},
		"",
	},
	// Docker ParseRepositoryTag testcase
	{
		"localhost.localdomain:5000/samalba/hipache:latest",
		&ImageID{
			Host: "localhost.localdomain",
			Port: 5000,
			User: "samalba",
			Repo: "hipache",
			Tag:  "latest",
		},
		"",
	},
	// numbers in host name
	{
		"niblet3:5000/devimg:latest",
		&ImageID{
			Host: "niblet3",
			Port: 5000,
			Repo: "devimg",
			Tag:  "latest",
		},
		"",
	},
	{
		"cp:5000/7j8ihkqdlkmqvvia886tvyf8g/zenoss5x",
		&ImageID{
			Host: "cp",
			Port: 5000,
			User: "7j8ihkqdlkmqvvia886tvyf8g",
			Repo: "zenoss5x",
		},
		"",
	},
	{
		"quay.io/zenossinc/daily-zenoss5-resmgr:5.0.0_421",
		&ImageID{
			Host: "quay.io",
			User: "zenossinc",
			Repo: "daily-zenoss5-resmgr",
			Tag:  "5.0.0_421",
		},
		"",
	},
	{
		"ubuntu:13.10",
		&ImageID{
			Repo: "ubuntu",
			Tag:  "13.10",
		},
		"",
	},
	// Unstable format - user, repo (with period), tag
	{
		"zenoss/resmgr_5.0:5.0.0_1234_unstable",
		&ImageID{
			User: "zenoss",
			Repo: "resmgr_5.0",
			Tag:  "5.0.0_1234_unstable",
		},
		"",
	},
	// Repo (with period), tag
	{
		"resmgr_5.0:5.0.0_1234_unstable",
		&ImageID{
			Repo: "resmgr_5.0",
			Tag:  "5.0.0_1234_unstable",
		},
		"",
	},
	// host, port, user, repo (with period), tag
	{
		"localhost:5000/16k18ljj5lwkfoe7tgegzfic8/resmgr_5.0:latest",
		&ImageID{
			Host: "localhost",
			Port: 5000,
			User: "16k18ljj5lwkfoe7tgegzfic8",
			Repo: "resmgr_5.0",
			Tag:  "latest",
		},
		"",
	},

	// host, user, repo (with period), tag
	{
		"localhost/16k18ljj5lwkfoe7tgegzfic8/resmgr_5.0:latest",
		&ImageID{
			Host: "localhost",
			User: "16k18ljj5lwkfoe7tgegzfic8",
			Repo: "resmgr_5.0",
			Tag:  "latest",
		},
		"",
	},
}

func doTest(c *C, parse func(string) (*ImageID, error), name string, tests []ImageIDTest) {
	for _, tt := range tests {
		imgid, err := parse(tt.in)
		c.Assert(err, IsNil)
		c.Assert(imgid, DeepEquals, tt.out)
		c.Assert(imgid.String(), Equals, tt.in)
	}
}

func (s *TestCommonsSuite) TestParse(c *C) {
	doTest(c, ParseImageID, "Parse", imgidtests)
}

func (s *TestCommonsSuite) TestString(c *C) {
	expectedID := "warner.bros:1948/dobbs/sierramadre:1925"
	iid, err := ParseImageID(expectedID)
	c.Assert(err, IsNil)
	c.Assert(iid.String(), Equals, expectedID)
}

func (s *TestCommonsSuite) TestBogusTag(c *C) {
	_, err := ParseImageID("sierramadre:feature/classic")
	c.Assert(err, Not(IsNil))
}

func (s *TestCommonsSuite) TestValidateInvalid(c *C) {
	iid := &ImageID{
		Host: "warner.bros",
		Port: 1948,
		User: "d%bbs",
		Repo: "sierramadre",
		Tag:  "feature",
	}

	c.Assert(iid.Validate(), Equals, false)
}

func (s *TestCommonsSuite) TestValidateValid(c *C) {
	iid := &ImageID{
		Repo: "sierramadre",
		Tag:  "543c56d1-2510-cd37-c0f4-cab544df985d",
	}

	c.Assert(iid.Validate(), Equals, true)
}

type ImageEqualsTest struct {
	id1, id2 string
	expected bool
}

func doImageEqualsTest(c *C, tests []ImageEqualsTest) {
	for _, tt := range tests {
		iid1, err := ParseImageID(tt.id1)
		c.Assert(err, IsNil)

		iid2, err := ParseImageID(tt.id2)
		c.Assert(err, IsNil)

		c.Assert(iid1.Equals(*iid2), Equals, tt.expected)
		c.Assert(iid2.Equals(*iid1), Equals, tt.expected)
	}
}

func (s *TestCommonsSuite) TestEquals(c *C) {
	tests := []ImageEqualsTest{
		{"warner.bros:1948/dobbs/sierramadre:1925", "warner.bros:1948/dobbs/sierramadre:1925", true},
		{"warner.bros:1948/dobbs/sierramadre:1925", "niblet3:5000/devimg:1925", false},
		{"warner.bros:1948/dobbs/sierramadre:1925", "warner.bros:1948/dobbs/sierramadre:1234", false},
		{"warner.bros:1948/dobbs/sierramadre:1925", "warner.bros:1948/dobbs/sierramadre", false},
		{"warner.bros:1948/dobbs/sierramadre:1925", "warner.bros:1948/dobbs/sierramadre:latest", false},
		{"warner.bros:1948/dobbs/sierramadre", "warner.bros:1948/dobbs/sierramadre:latest", true},
		{"warner.bros:1948/dobbs/sierramadre", "warner.bros:1948/dobbs/sierramadre", true},
		{"warner.bros:1948/dobbs/sierramadre", "niblet3:5000/devimg", false},
		{"warner.bros:1948/dobbs/sierramadre", "niblet3:5000/devimg:latest", false},
	}
	doImageEqualsTest(c, tests)
}

type RenameTest struct {
	registry string
	tenant   string
	imgID    string
	tag      string
	ImageID  *ImageID
	anErr    bool
}

func (rt1 RenameTest) Copy() RenameTest {
	rt2 := &RenameTest{}
	rt2.registry = rt1.registry
	rt2.tenant = rt1.tenant
	rt2.imgID = rt1.imgID
	rt2.tag = rt1.tag
	rt2.ImageID = rt1.ImageID.Copy()
	rt2.anErr = rt1.anErr
	return *rt2
}

var renameTest = RenameTest{
	"localhost:5000",
	"the_tenant",
	"zenoss/core-unstable:5.0.0",
	"latest",
	&ImageID{
		Host: "localhost",
		Port: 5000,
		User: "the_tenant",
		Repo: "core-unstable",
		Tag:  "latest",
	},
	false,
}

var renameTests = []RenameTest{
	renameTest.Copy(),
	func() RenameTest {
		rt := renameTest.Copy()
		rt.imgID = ""
		rt.anErr = true
		return rt
	}(),
}

func doRenameImageIdTest(c *C, tests []RenameTest) {
	for _, test := range tests {
		image, err := RenameImageID(test.registry, test.tenant, test.imgID, test.tag)
		if test.anErr {
			c.Assert(err, Not(IsNil))
			continue
		}
		c.Assert(err, IsNil)
		c.Assert(image.Equals(*test.ImageID), Equals, true)
	}
}

func (s *TestCommonsSuite) TestRenameImageID(c *C) {
	doRenameImageIdTest(c, renameTests)
}

func (s *TestCommonsSuite) TestMerge(c *C) {
	img1_orig := &ImageID{
		"host",
		1,
		"user",
		"repo",
		"tag",
	}

	img1 := img1_orig.Copy()
	img2 := &ImageID{"host2", 2, "user2", "repo2", "tag2"}
	img1.Merge(img2)
	c.Assert(img2, DeepEquals, img1)

	img1 = img1_orig.Copy()
	img1.Merge(&ImageID{Repo: "apples"})
	c.Assert(img1, DeepEquals, &ImageID{"host", 1, "user", "apples", "tag"})

	img1 = img1_orig.Copy()
	img1.Merge(&ImageID{})
	c.Assert(img1, DeepEquals, img1_orig)

}
