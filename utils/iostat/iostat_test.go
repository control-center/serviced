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

// +build unit

package iostat

import (
	"io"
	"io/ioutil"
	"os"
	"testing"
	"time"

	. "gopkg.in/check.v1"
)

func TestIOStat(t *testing.T) { TestingT(t) }

type IOStatSuite struct {
	testFile       string
	parsedTestFile map[string]DeviceUtilizationReport
}

var _ = Suite(&IOStatSuite{})

func (s *IOStatSuite) SetUpSuite(c *C) {
	s.testFile = "testfiles/iostat.out"
	s.parsedTestFile = make(map[string]DeviceUtilizationReport)
	s.parsedTestFile["sda"] = DeviceUtilizationReport{
		Device:  "sda",
		RRQMPS:  float64(0.13),
		WRQMPS:  float64(74.32),
		RPS:     float64(10.73),
		WPS:     float64(59.68),
		RKBPS:   float64(217.26),
		WKBPS:   float64(4130.44),
		AvgRqSz: float64(123.50),
		AvgQuSz: float64(0.61),
		Await:   float64(8.71),
		PctUtil: float64(3.91),
	}

	s.parsedTestFile["sdb"] = DeviceUtilizationReport{
		Device:  "sdb",
		RRQMPS:  float64(24.80),
		WRQMPS:  float64(27.34),
		RPS:     float64(161.93),
		WPS:     float64(32.07),
		RKBPS:   float64(4640.81),
		WKBPS:   float64(857.53),
		AvgRqSz: float64(56.68),
		AvgQuSz: float64(3.10),
		Await:   float64(15.95),
		PctUtil: float64(32.89),
	}

}

func (s *IOStatSuite) TestIOStatParser(c *C) {
	r, err := os.Open(s.testFile)
	c.Assert(err, IsNil)
	defer r.Close()

	m, err := ParseIOStat(r)
	c.Assert(err, IsNil)
	// Hand picked data to verify
	c.Check(len(m), Equals, 2)
	report := m["sda"]
	c.Check(report, DeepEquals, s.parsedTestFile["sda"])

	report = m["sdb"]
	c.Check(report, DeepEquals, s.parsedTestFile["sdb"])
}

func (s *IOStatSuite) TestToSimpleIOStat(c *C) {
	report := DeviceUtilizationReport{
		Device:  "sda",
		RRQMPS:  float64(0.13),
		WRQMPS:  float64(74.32),
		RPS:     float64(10.73),
		WPS:     float64(59.68),
		RKBPS:   float64(217.26),
		WKBPS:   float64(4130.44),
		AvgRqSz: float64(123.50),
		AvgQuSz: float64(0.61),
		Await:   float64(8.71),
		PctUtil: float64(3.91),
	}
	simple, err := report.ToSimpleIOStat()
	c.Assert(err, IsNil)

	c.Check(simple, DeepEquals, SimpleIOStat{
		Device: "sda",
		RPS:    float64(10.73),
		WPS:    float64(59.68),
		Await:  float64(8.71),
	})
}

func (s *IOStatSuite) TestparseIOStatWatcher_Success(c *C) {
	fileContents, err := ioutil.ReadFile(s.testFile)
	c.Assert(err, IsNil)
	reader, writer := io.Pipe()
	resultChan := make(chan map[string]DeviceUtilizationReport)
	quitChan := make(chan interface{})
	done := make(chan interface{})

	go func() {
		defer close(done)
		parseIOStatWatcher(reader, resultChan, quitChan)
	}()

	// Send one batch of data, which includes 2 newlines at the end
	writer.Write(fileContents)

	// Make sure we get it back out of the channel
	timer := time.NewTimer(time.Second)
	select {
	case result := <-resultChan:
		c.Assert(result, DeepEquals, s.parsedTestFile)
	case <-timer.C:
		c.Fatalf("Failed to read result from channel")
	}

	// Send it again and make sure it still works
	writer.Write(fileContents)
	timer.Reset(time.Second)
	select {
	case result := <-resultChan:
		c.Assert(result, DeepEquals, s.parsedTestFile)
	case <-timer.C:
		c.Fatalf("Failed to read result from channel")
	}

	// Close the quit channel and make sure we exit
	close(quitChan)
	timer.Reset(time.Second)
	select {
	case <-done:
	case <-timer.C:
		c.Fatalf("method did not exit")
	}
}

func (s *IOStatSuite) TestparseIOStatWatcher_parseFailed(c *C) {
	reader, writer := io.Pipe()
	resultChan := make(chan map[string]DeviceUtilizationReport)
	quitChan := make(chan interface{})
	done := make(chan interface{})

	go func() {
		defer close(done)
		parseIOStatWatcher(reader, resultChan, quitChan)
	}()

	// Send text that will cause a parse error
	garbage := []byte("Device tps\nmydevice notafloat\n\n")
	writer.Write(garbage)

	// Make sure we exit
	timer := time.NewTimer(time.Second)
	select {
	case <-done:
	case <-timer.C:
		c.Fatalf("method did not exit")
	}
}

func (s *IOStatSuite) TestparseIOStatWatcher_EOF(c *C) {
	reader, writer := io.Pipe()
	resultChan := make(chan map[string]DeviceUtilizationReport)
	quitChan := make(chan interface{})
	done := make(chan interface{})

	go func() {
		defer close(done)
		parseIOStatWatcher(reader, resultChan, quitChan)
	}()

	// Close the writer
	writer.Close()

	// Make sure we exit
	timer := time.NewTimer(time.Second)
	select {
	case <-done:
	case <-timer.C:
		c.Fatalf("method did not exit")
	}
}
