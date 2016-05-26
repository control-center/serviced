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

package api

import (
	"fmt"
	"math"
	"reflect"

	"github.com/control-center/serviced/cli/api/mocks"
	"github.com/control-center/serviced/domain/service"
	"github.com/stretchr/testify/mock"
	"github.com/zenoss/elastigo/core"
	. "gopkg.in/check.v1"
)

func (s *TestAPISuite) testConvertOffsets(c *C, received []string, expected []uint64) {
	converted, err := convertOffsets(received)
	if err != nil {
		c.Fatalf("unexpected error converting offsets: %s", err)
	}
	if !reflect.DeepEqual(converted, expected) {
		c.Fatalf("got %v expected %v", converted, expected)
	}
}

func (s *TestAPISuite) testUint64sAreSorted(c *C, values []uint64, expected bool) {
	if uint64sAreSorted(values) != expected {
		c.Fatalf("expected %v for sortedness for values: %v", expected, values)
	}
}

func (s *TestAPISuite) testGetMinValue(c *C, values []uint64, expected uint64) {
	if getMinValue(values) != expected {
		c.Fatalf("expected min value %v from values: %v", expected, values)
	}
}

func (s *TestAPISuite) testGenerateOffsets(c *C, inMessages []string, inOffsets, expected []uint64) {
	converted := generateOffsets(inMessages, inOffsets)
	if !reflect.DeepEqual(converted, expected) {
		c.Fatalf("unexpected error generating offsets from %v:%v got %v expected %v", inMessages, inOffsets, converted, expected)
	}
}

func (s *TestAPISuite) TestLogs_Offsets(c *C) {
	s.testConvertOffsets(c, []string{"123", "456", "789"}, []uint64{123, 456, 789})
	s.testConvertOffsets(c, []string{"456", "123", "789"}, []uint64{456, 123, 789})

	s.testUint64sAreSorted(c, []uint64{123, 124, 125}, true)
	s.testUint64sAreSorted(c, []uint64{123, 125, 124}, false)
	s.testUint64sAreSorted(c, []uint64{125, 123, 124}, false)

	s.testGetMinValue(c, []uint64{}, math.MaxUint64)
	s.testGetMinValue(c, []uint64{125, 123, 124}, 123)

	s.testGenerateOffsets(c, []string{}, []uint64{}, []uint64{})
	s.testGenerateOffsets(c, []string{"abc", "def", "ghi"}, []uint64{456, 123, 789}, []uint64{123, 124, 125})
	s.testGenerateOffsets(c, []string{"abc", "def", "ghi"}, []uint64{456, 124}, []uint64{124, 125, 126})
}

func (s *TestAPISuite) TestLogs_BuildQuery_AllServices(c *C) {
	config := ExportLogsConfig{ServiceIDs: []string{}}
	exporter := logExporter{ExportLogsConfig: config}
	getServices := func() ([]service.Service, error) {
		c.Fatalf("GetServices called when it should not have been")
		return []service.Service{}, nil
	}

	query, err := exporter.buildQuery(getServices)

	c.Assert(query, Equals, "*")
	c.Assert(err, IsNil)
}

// If the DB has no services, we will at least query for the specified serviceID (e.g. could be logs from a deleted service)
func (s *TestAPISuite) TestLogs_BuildQuery_DBEmpty(c *C) {
	config := ExportLogsConfig{ServiceIDs: []string{"servicedID1"}}
	exporter := logExporter{ExportLogsConfig: config}
	getServices := func() ([]service.Service, error) {
		return []service.Service{}, nil
	}

	query, err := exporter.buildQuery(getServices)

	c.Assert(query, Equals, "service:(\"servicedID1\")")
	c.Assert(err, IsNil)
}

func (s *TestAPISuite) TestLogs_BuildQuery_OneService(c *C) {
	serviceID := "someServiceID"
	config := ExportLogsConfig{ServiceIDs: []string{serviceID}}
	exporter := logExporter{ExportLogsConfig: config}
	getServices := func() ([]service.Service, error) {
		return []service.Service{{ID: serviceID}}, nil
	}

	query, err := exporter.buildQuery(getServices)

	c.Assert(query, Equals, fmt.Sprintf("service:(\"%s\")", serviceID))
	c.Assert(err, IsNil)
}

func (s *TestAPISuite) TestLogs_BuildQuery_ServiceWithChildren(c *C) {
	parentServiceID := "parentServiceID"
	config := ExportLogsConfig{ServiceIDs: []string{parentServiceID}}
	exporter := logExporter{ExportLogsConfig: config}
	getServices := func() ([]service.Service, error) {
		services := []service.Service{
			{ID: parentServiceID},
			{ID: "child1", ParentServiceID: parentServiceID},
			{ID: "child2", ParentServiceID: parentServiceID},
		}
		return services, nil
	}

	query, err := exporter.buildQuery(getServices)

	c.Assert(query, Equals, fmt.Sprintf("service:(\"child1\" OR \"child2\" OR \"%s\")", parentServiceID))
	c.Assert(err, IsNil)
}

func (s *TestAPISuite) TestLogs_BuildQuery_MultipleServices(c *C) {
	config := ExportLogsConfig{ServiceIDs: []string{"service1", "service2", "service3"}}
	exporter := logExporter{ExportLogsConfig: config}
	getServices := func() ([]service.Service, error) {
		services := []service.Service{
			{ID: "service1"},
			{ID: "service2"},
			{ID: "service3"},
		}
		return services, nil
	}

	query, err := exporter.buildQuery(getServices)

	c.Assert(query, Equals, "service:(\"service1\" OR \"service2\" OR \"service3\")")
	c.Assert(err, IsNil)
}

func (s *TestAPISuite) TestLogs_BuildQuery_ChildrenAreNotDuplicated(c *C) {
	config := ExportLogsConfig{ServiceIDs: []string{"service1", "service2", "service3"}}
	exporter := logExporter{ExportLogsConfig: config}
	getServices := func() ([]service.Service, error) {
		services := []service.Service{
			{ID: "service1"},
			{ID: "service2", ParentServiceID: "service1"},
			{ID: "service3", ParentServiceID: "service1"},
		}
		return services, nil
	}

	query, err := exporter.buildQuery(getServices)

	c.Assert(query, Equals, "service:(\"service1\" OR \"service2\" OR \"service3\")")
	c.Assert(err, IsNil)
}

func (s *TestAPISuite) TestLogs_BuildQuery_DBFails(c *C) {
	expectedError := fmt.Errorf("GetServices failed")
	config := ExportLogsConfig{ServiceIDs: []string{"servicedID1"}}
	exporter := logExporter{ExportLogsConfig: config}
	getServices := func() ([]service.Service, error) {
		return nil, expectedError
	}

	query, err := exporter.buildQuery(getServices)

	c.Assert(query, Equals, "")
	c.Assert(err, Equals, expectedError)
}

func (s *TestAPISuite) TestLogs_BuildQuery_InvalidServiceIDs(c *C) {
	config := ExportLogsConfig{ServiceIDs: []string{"!@#$%^&*()"}}
	exporter := logExporter{ExportLogsConfig: config}
	getServices := func() ([]service.Service, error) {
		return []service.Service{}, nil
	}

	query, err := exporter.buildQuery(getServices)

	c.Assert(query, Equals, "")
	c.Assert(err, ErrorMatches, "invalid service ID format: .*")
}

func (s *TestAPISuite) TestLogs_RetrieveLogs_NoDateMatch(c *C) {
	logstashDays := []string{"2112.01.01"}
	serviceIDs := []string{"someServiceID"}
	fromDate := "2015.01.01"
	toDate := "2015.01.01"
	exporter, mockLogDriver, err := setupRetrieveLogTest(logstashDays, serviceIDs, fromDate, toDate)
	defer func () {
		if exporter != nil {
			exporter.cleanup()
		}
	}()
	c.Assert(err, IsNil)

	mockLogDriver.On("StartSearch", mock.AnythingOfType("string"), mock.AnythingOfType("string")).
		Return(core.SearchResult{}, fmt.Errorf("StartSearch was called unexpectedly"))
	outputFiles := []outputFileInfo{}
	fileIndex := make(map[string]map[string]int) // containerID => filename => index

	foundIndexedDay, numWarnings, err := exporter.retrieveLogs(outputFiles, fileIndex)

	c.Assert(foundIndexedDay, Equals, false)
	c.Assert(numWarnings, Equals, 0)
	c.Assert(err, IsNil)
}

func (s *TestAPISuite) TestLogs_RetrieveLogs_StartSearchFails(c *C) {
	logstashDays := []string{"2112.01.01"}
	serviceIDs := []string{"someServiceID"}
	fromDate := logstashDays[0]
	toDate := logstashDays[0]
	exporter, mockLogDriver, err := setupRetrieveLogTest(logstashDays, serviceIDs, fromDate, toDate)
	defer func () {
		if exporter != nil {
			exporter.cleanup()
		}
	}()
	c.Assert(err, IsNil)

	expectedError := fmt.Errorf("StartSearch failed")
	mockLogDriver.On("StartSearch", mock.AnythingOfType("string"), mock.AnythingOfType("string")).
		Return(core.SearchResult{}, expectedError)
	outputFiles := []outputFileInfo{}
	fileIndex := make(map[string]map[string]int) // containerID => filename => index

	foundIndexedDay, numWarnings, err := exporter.retrieveLogs(outputFiles, fileIndex)

	c.Assert(foundIndexedDay, Equals, true)
	c.Assert(numWarnings, Equals, 0)
	c.Assert(err, ErrorMatches, fmt.Sprintf(".*%s", expectedError))
}

func (s *TestAPISuite) TestLogs_RetrieveLogs_SearchHasNoHits(c *C) {
	logstashDays := []string{"2112.01.01"}
	serviceIDs := []string{"someServiceID"}
	fromDate := logstashDays[0]
	toDate := logstashDays[0]
	exporter, mockLogDriver, err := setupRetrieveLogTest(logstashDays, serviceIDs, fromDate, toDate)
	defer func () {
		if exporter != nil {
			exporter.cleanup()
		}
	}()
	c.Assert(err, IsNil)

	mockLogDriver.On("StartSearch", mock.AnythingOfType("string"), mock.AnythingOfType("string")).
		Return(core.SearchResult{}, nil)
	outputFiles := []outputFileInfo{}
	fileIndex := make(map[string]map[string]int) // containerID => filename => index

	foundIndexedDay, numWarnings, err := exporter.retrieveLogs(outputFiles, fileIndex)

	c.Assert(foundIndexedDay, Equals, true)
	c.Assert(numWarnings, Equals, 0)
	c.Assert(err, IsNil)
	c.Assert(len(outputFiles), Equals, 0)
	c.Assert(len(fileIndex), Equals, 0)
}

func (s *TestAPISuite) TestLogs_RetrieveLogs_SearchHasAHit(c *C) {
	logstashDays := []string{"2112.01.01"}
	serviceIDs := []string{"someServiceID"}
	fromDate := logstashDays[0]
	toDate := logstashDays[0]
	exporter, mockLogDriver, err := setupRetrieveLogTest(logstashDays, serviceIDs, fromDate, toDate)
	defer func () {
		if exporter != nil {
			exporter.cleanup()
		}
	}()
	c.Assert(err, IsNil)

	searchResult := core.SearchResult{
		ScrollId: "search1",
		Hits: core.Hits{
			Total:    1,
			Hits:  []core.Hit{
				core.Hit{Source: []byte(`{"host": "container1", "file": "file1", "message": "message1"}`),},
			},
		},
	}
	mockLogDriver.On("StartSearch", mock.AnythingOfType("string"), mock.AnythingOfType("string")).
		Return(searchResult, nil)

	firstSearchResult := searchResult
	firstSearchResult.ScrollId = "lastSearch"
	lastSearchResult := core.SearchResult{
		ScrollId: "lastSearch",
		Hits: core.Hits{
			Total:    0,
			Hits:  []core.Hit{},
		},
	}
	mockLogDriver.On("ScrollSearch", searchResult.ScrollId).Return(firstSearchResult, nil)
	mockLogDriver.On("ScrollSearch", firstSearchResult.ScrollId).Return(lastSearchResult, nil)

	outputFiles := []outputFileInfo{}
	fileIndex := make(map[string]map[string]int) // containerID => filename => index

	foundIndexedDay, numWarnings, err := exporter.retrieveLogs(outputFiles, fileIndex)

	c.Assert(foundIndexedDay, Equals, true)
	c.Assert(numWarnings, Equals, 0)
	c.Assert(err, IsNil)
	c.Assert(len(outputFiles), Equals, 0)
	c.Assert(len(fileIndex), Equals, 0)
}

func setupRetrieveLogTest(logstashDays, serviceIDs []string, fromDate, toDate string) (*logExporter, *mocks.ExportLogDriver, error) {
	mockLogDriver := &mocks.ExportLogDriver{}
	mockLogDriver.On("LogstashDays").Return(logstashDays, nil)

	config := ExportLogsConfig{
		ServiceIDs: serviceIDs,
		FromDate:   fromDate,
		ToDate:     toDate,
		Driver:     mockLogDriver,
	}
	getServices := func() ([]service.Service, error) {
		return []service.Service{}, nil
	}

	exporter, err := buildExporter(config, getServices)
	return exporter, mockLogDriver, err
}
