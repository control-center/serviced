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

package utils

import (
	"os"
	"testing"

	. "gopkg.in/check.v1"
)

func TestIOStat(t *testing.T) { TestingT(t) }

type IOStatSuite struct{}

var _ = Suite(&IOStatSuite{})

func (s *IOStatSuite) TestIOStatParser(c *C) {
	r, err := os.Open("testfiles/iostat.out")
	if err != nil {
		panic(err)
	}
	defer r.Close()

	m, err := ParseIOStat(r)
	if err != nil {
		panic(err)
	}
	// Hand picked data to verify
	c.Check(len(m), Equals, 2)
	report := m["sda"]
	c.Check(report, DeepEquals, DeviceUtilizationReport{
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
	})

	report = m["sdb"]
	c.Check(report, DeepEquals, DeviceUtilizationReport{
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
	})
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
	if err != nil {
		panic(err)
	}
	c.Check(simple, DeepEquals, SimpleIOStat{
		Device: "sda",
		RPS:    float64(10.73),
		WPS:    float64(59.68),
		Await:  float64(8.71),
	})
}
