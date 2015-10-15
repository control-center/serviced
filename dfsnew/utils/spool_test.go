// Copyright 2015 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// +build integration

package utils_test

import (
	"bytes"
	"io/ioutil"
	"testing"

	. "github.com/control-center/serviced/dfsnew/utils"
	. "gopkg.in/check.v1"
)

type SpoolTestSuite struct {
	dir   string
	spool Spooler
}

var _ = Suite(&SpoolTestSuite{})

func TestSpool(t *testing.T) { TestingT(t) }

func (s *SpoolTestSuite) SetUpTest(c *C) {
	s.dir = c.MkDir()
	spool, err := NewSpool(s.dir)
	c.Assert(err, IsNil)
	c.Assert(spool, NotNil)
	s.spool = spool
}

func (s *SpoolTestSuite) TestWrite(c *C) {
	defer s.spool.Close()
	buf := bytes.NewBufferString("this is a test")
	n, err := s.spool.Write(buf.Bytes())
	c.Assert(err, IsNil)
	c.Assert(n, Equals, buf.Len())
	c.Assert(s.spool.Size(), Equals, int64(buf.Len()))
}

func (s *SpoolTestSuite) TestWriteTo(c *C) {
	defer s.spool.Close()
	expected := bytes.NewBufferString("this is a test")
	n, err := s.spool.Write(expected.Bytes())
	c.Assert(err, IsNil)
	c.Assert(n, Equals, expected.Len())
	c.Assert(s.spool.Size(), Equals, int64(expected.Len()))

	actual := bytes.NewBufferString("")
	n64, err := s.spool.WriteTo(actual)
	c.Assert(err, IsNil)
	c.Assert(n64, Equals, int64(expected.Len()))
	c.Assert(s.spool.Size(), DeepEquals, int64(0))
	c.Assert(actual.Bytes(), DeepEquals, expected.Bytes())

	actual.Reset()
	n64, err = s.spool.WriteTo(actual)
	c.Assert(err, IsNil)
	c.Assert(n64, DeepEquals, int64(0))
	c.Assert(s.spool.Size(), DeepEquals, int64(0))
	c.Assert(actual.Len(), DeepEquals, 0)

	expected.Reset()
	expected.WriteString("this is another test")
	n, err = s.spool.Write(expected.Bytes())
	c.Assert(err, IsNil)
	c.Assert(n, Equals, expected.Len())
	c.Assert(s.spool.Size(), Equals, int64(expected.Len()))

	actual.Reset()
	n64, err = s.spool.WriteTo(actual)
	c.Assert(err, IsNil)
	c.Assert(n64, Equals, int64(expected.Len()))
	c.Assert(s.spool.Size(), DeepEquals, int64(0))
	c.Assert(actual.Bytes(), DeepEquals, expected.Bytes())
}

func (s *SpoolTestSuite) TestReset(c *C) {
	defer s.spool.Close()
	n1, err := s.spool.Write([]byte("now we will test reset"))
	c.Assert(err, IsNil)
	c.Assert(s.spool.Size(), Equals, int64(n1))
	n2, err := s.spool.Write([]byte("but first, lets try writing"))
	c.Assert(err, IsNil)
	c.Assert(s.spool.Size(), Equals, int64(n1+n2))
	err = s.spool.Reset()
	c.Assert(err, IsNil)
	c.Assert(s.spool.Size(), DeepEquals, int64(0))

	expected := bytes.NewBufferString("this should be the only thing i read")
	n, err := s.spool.Write(expected.Bytes())
	c.Assert(err, IsNil)
	c.Assert(s.spool.Size(), Equals, int64(n))
	c.Assert(s.spool.Size(), Equals, int64(expected.Len()))
	actual := bytes.NewBufferString("")
	n64, err := s.spool.WriteTo(actual)
	c.Assert(err, IsNil)
	c.Assert(n64, Equals, int64(expected.Len()))
	c.Assert(s.spool.Size(), DeepEquals, int64(0))
	c.Assert(actual.Bytes(), DeepEquals, expected.Bytes())
}

func (s *SpoolTestSuite) TestSize(c *C) {
	defer s.spool.Close()
	c.Assert(s.spool.Size(), DeepEquals, int64(0))
	buf := bytes.NewBufferString("let's make sure the size is right")
	n1, err := s.spool.Write(buf.Bytes())
	c.Assert(err, IsNil)
	c.Assert(n1, Equals, buf.Len())
	c.Assert(s.spool.Size(), Equals, int64(buf.Len()))
	c.Log(1)

	buf.Reset()
	buf.WriteString("now I am adding more stuff")
	n2, err := s.spool.Write(buf.Bytes())
	c.Assert(err, IsNil)
	c.Assert(n2, Equals, buf.Len())
	c.Assert(s.spool.Size(), Equals, int64(n1+n2))
	c.Log(2)

	buf.Reset()
	n64, err := s.spool.WriteTo(buf)
	c.Assert(err, IsNil)
	c.Assert(n64, Equals, int64(n1+n2))
	c.Assert(n64, Equals, int64(buf.Len()))
	c.Assert(s.spool.Size(), DeepEquals, int64(0))
	c.Log(3)

	n3, err := s.spool.Write(buf.Bytes())
	c.Assert(err, IsNil)
	c.Assert(n3, Equals, buf.Len())
	c.Assert(s.spool.Size(), Equals, int64(buf.Len()))
	c.Log(4)

	err = s.spool.Reset()
	c.Assert(err, IsNil)
	c.Assert(s.spool.Size(), DeepEquals, int64(0))
}

func (s *SpoolTestSuite) TestClose(c *C) {
	buf := bytes.NewBufferString("testing the closer")
	n, err := s.spool.Write(buf.Bytes())
	c.Assert(err, IsNil)
	c.Assert(n, Equals, buf.Len())
	c.Assert(s.spool.Size(), Equals, int64(buf.Len()))
	err = s.spool.Close()
	c.Assert(err, IsNil)
	fis, err := ioutil.ReadDir(s.dir)
	c.Assert(err, IsNil)
	c.Assert(fis, HasLen, 0)
}
