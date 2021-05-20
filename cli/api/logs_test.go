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
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"time"

	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/domain/service"
	"github.com/stretchr/testify/mock"
	. "gopkg.in/check.v1"
)

func (s *TestAPISuite) testConvertOffsets(c *C, received interface{}, expected []uint64) {
	converted, err := convertGenericOffsets(received)
	if err != nil {
		c.Fatalf("unexpected error converting offsets: %s", err)
	}
	if !reflect.DeepEqual(converted, expected) {
		c.Fatalf("got %v expected %v", converted, expected)
	}
}

func (s *TestAPISuite) testConvertMessages(c *C, received interface{}, expected []string) {
	converted, err := convertGenericMessages(received)
	if err != nil {
		c.Fatalf("unexpected error converting messages: %s", err)
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
	s.testConvertOffsets(c, []json.Number{"123", "456", "789"}, []uint64{123, 456, 789})
	s.testConvertOffsets(c, []json.Number{"456", "123", "789"}, []uint64{456, 123, 789})
	s.testConvertOffsets(c, json.Number("456"), []uint64{456})
	s.testConvertOffsets(c, []float64{12345.6789}, []uint64{12345})
	s.testConvertOffsets(c, float64(12345.6789), []uint64{12345})
	s.testConvertOffsets(c, []uint64{12345}, []uint64{12345})
	s.testConvertOffsets(c, uint64(12345), []uint64{12345})

	s.testUint64sAreSorted(c, []uint64{123, 124, 125}, true)
	s.testUint64sAreSorted(c, []uint64{123, 125, 124}, false)
	s.testUint64sAreSorted(c, []uint64{125, 123, 124}, false)

	s.testGetMinValue(c, []uint64{}, math.MaxUint64)
	s.testGetMinValue(c, []uint64{125, 123, 124}, 123)

	s.testGenerateOffsets(c, []string{}, []uint64{}, []uint64{})
	s.testGenerateOffsets(c, []string{"abc", "def", "ghi"}, []uint64{456, 123, 789}, []uint64{123, 124, 125})
	s.testGenerateOffsets(c, []string{"abc", "def", "ghi"}, []uint64{456, 124}, []uint64{124, 125, 126})
}

func (s *TestAPISuite) TestLogs_Messages(c *C) {
	s.testConvertMessages(c, []string{"s1", "s2", "s3"}, []string{"s1", "s2", "s3"})
	s.testConvertMessages(c, "s1", []string{"s1"})
}

func (s *TestAPISuite) TestLogs_BuildQuery_AllServices(c *C) {
	config := ExportLogsConfig{ServiceIDs: []string{}, Debug: true}
	exporter := logExporter{ExportLogsConfig: config}
	getServices := func() ([]service.ServiceDetails, error) {
		c.Fatalf("GetServices called when it should not have been")
		return []service.ServiceDetails{}, nil
	}

	query, err := exporter.buildQuery(getServices)

	c.Assert(query, Equals, "*")
	c.Assert(err, IsNil)
}

// If the DB has no services, we will at least query for the specified serviceID (e.g. could be logs from a deleted service)
func (s *TestAPISuite) TestLogs_BuildQuery_DBEmpty(c *C) {
	config := ExportLogsConfig{ServiceIDs: []string{"servicedID1"}, Debug: true}
	exporter := logExporter{ExportLogsConfig: config}
	getServices := func() ([]service.ServiceDetails, error) {
		return []service.ServiceDetails{}, nil
	}

	query, err := exporter.buildQuery(getServices)

	c.Assert(query, Equals, "fields.service:(\"servicedID1\")")
	c.Assert(err, IsNil)
}

func (s *TestAPISuite) TestLogs_BuildQuery_OneService(c *C) {
	serviceID := "someServiceID"
	config := ExportLogsConfig{ServiceIDs: []string{serviceID}, Debug: true}
	exporter := logExporter{ExportLogsConfig: config}
	getServices := func() ([]service.ServiceDetails, error) {
		return []service.ServiceDetails{{ID: serviceID}}, nil
	}

	query, err := exporter.buildQuery(getServices)

	c.Assert(query, Equals, fmt.Sprintf("fields.service:(\"%s\")", serviceID))
	c.Assert(err, IsNil)
}

func (s *TestAPISuite) TestLogs_BuildQuery_ServiceWithChildren(c *C) {
	parentServiceID := "parentServiceID"
	config := ExportLogsConfig{ServiceIDs: []string{parentServiceID}, Debug: true}
	exporter := logExporter{ExportLogsConfig: config}
	getServices := func() ([]service.ServiceDetails, error) {
		services := []service.ServiceDetails{
			{ID: parentServiceID},
			{ID: "child1", ParentServiceID: parentServiceID},
			{ID: "child2", ParentServiceID: parentServiceID},
		}
		return services, nil
	}

	query, err := exporter.buildQuery(getServices)

	c.Assert(query, Equals, fmt.Sprintf("fields.service:(\"child1\" OR \"child2\" OR \"%s\")", parentServiceID))
	c.Assert(err, IsNil)
}

func (s *TestAPISuite) TestLogs_BuildQuery_MultipleServices(c *C) {
	config := ExportLogsConfig{ServiceIDs: []string{"service1", "service2", "service3"}, Debug: true}
	exporter := logExporter{ExportLogsConfig: config}
	getServices := func() ([]service.ServiceDetails, error) {
		services := []service.ServiceDetails{
			{ID: "service1"},
			{ID: "service2"},
			{ID: "service3"},
		}
		return services, nil
	}

	query, err := exporter.buildQuery(getServices)

	c.Assert(query, Equals, "fields.service:(\"service1\" OR \"service2\" OR \"service3\")")
	c.Assert(err, IsNil)
}

func (s *TestAPISuite) TestLogs_BuildQuery_ChildrenAreNotDuplicated(c *C) {
	config := ExportLogsConfig{ServiceIDs: []string{"service1", "service2", "service3"}, Debug: true}
	exporter := logExporter{ExportLogsConfig: config}
	getServices := func() ([]service.ServiceDetails, error) {
		services := []service.ServiceDetails{
			{ID: "service1"},
			{ID: "service2", ParentServiceID: "service1"},
			{ID: "service3", ParentServiceID: "service1"},
		}
		return services, nil
	}

	query, err := exporter.buildQuery(getServices)

	c.Assert(query, Equals, "fields.service:(\"service1\" OR \"service2\" OR \"service3\")")
	c.Assert(err, IsNil)
}

func (s *TestAPISuite) TestLogs_BuildQuery_DBFails(c *C) {
	expectedError := fmt.Errorf("GetServices failed")
	config := ExportLogsConfig{ServiceIDs: []string{"servicedID1"}, Debug: true}
	exporter := logExporter{ExportLogsConfig: config}
	getServices := func() ([]service.ServiceDetails, error) {
		return nil, expectedError
	}

	query, err := exporter.buildQuery(getServices)

	c.Assert(query, Equals, "")
	c.Assert(err, Equals, expectedError)
}

func (s *TestAPISuite) TestLogs_BuildQuery_InvalidServiceIDs(c *C) {
	config := ExportLogsConfig{ServiceIDs: []string{"!@#$%^&*()"}, Debug: true}
	exporter := logExporter{ExportLogsConfig: config}
	getServices := func() ([]service.ServiceDetails, error) {
		return []service.ServiceDetails{}, nil
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
	defer func() {
		if exporter != nil {
			exporter.cleanup()
		}
	}()
	c.Assert(err, IsNil)

	mockLogDriver.On("StartSearch", mock.AnythingOfType("string"), mock.AnythingOfType("string")).
		Return(ElasticSearchResults{}, fmt.Errorf("StartSearch was called unexpectedly"))

	foundIndexedDay, numWarnings, err := exporter.retrieveLogs()

	c.Assert(foundIndexedDay, Equals, false)
	c.Assert(numWarnings, Equals, 0)
	c.Assert(err, IsNil)
	c.Assert(len(exporter.outputFiles), Equals, 0)
}

func (s *TestAPISuite) TestLogs_RetrieveLogs_StartSearchFails(c *C) {
	exporter, mockLogDriver, err := setupSimpleRetrieveLogTest()
	defer func() {
		if exporter != nil {
			exporter.cleanup()
		}
	}()
	c.Assert(err, IsNil)

	expectedError := fmt.Errorf("StartSearch failed")
	mockLogDriver.On("StartSearch", mock.AnythingOfType("string"), mock.AnythingOfType("string")).
		Return(ElasticSearchResults{}, expectedError)

	foundIndexedDay, numWarnings, err := exporter.retrieveLogs()

	c.Assert(foundIndexedDay, Equals, true)
	c.Assert(numWarnings, Equals, 0)
	c.Assert(err, ErrorMatches, fmt.Sprintf(".*%s", expectedError))
	c.Assert(len(exporter.outputFiles), Equals, 0)
}

func (s *TestAPISuite) TestLogs_RetrieveLogs_SearchHasNoHits(c *C) {
	exporter, mockLogDriver, err := setupSimpleRetrieveLogTest()
	defer func() {
		if exporter != nil {
			exporter.cleanup()
		}
	}()
	c.Assert(err, IsNil)

	mockLogDriver.On("StartSearch", mock.AnythingOfType("string"), mock.AnythingOfType("string")).
		Return(ElasticSearchResults{}, nil)

	foundIndexedDay, numWarnings, err := exporter.retrieveLogs()

	c.Assert(foundIndexedDay, Equals, true)
	c.Assert(numWarnings, Equals, 0)
	c.Assert(err, IsNil)
	c.Assert(len(exporter.outputFiles), Equals, 0)
}

func (s *TestAPISuite) TestLogs_RetrieveLogs_SearchFindsOneFileWithOneScroll(c *C) {
	exporter, mockLogDriver, err := setupSimpleRetrieveLogTest()
	defer func() {
		if exporter != nil {
			exporter.cleanup()
		}
	}()
	c.Assert(err, IsNil)
	source := setupOneSearchResult(c, "log", "host1", "ServiceID", "container1", "file1", "message1")
	searchStart := ElasticSearchResults{
		ScrollID: "search1",
		Hits: ExternalHit{
			Total:    Total{ Value: 1, Relation: ""},
			MaxScore: 0,
			Hits:     []InternalHit{
				{Source: (*json.RawMessage)(&source)},
			},
		},
	}
	mockLogDriver.On("StartSearch", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(searchStart, nil)

	// Because ScrollSearch() does not accept a pointer, we have to fake things a little by using separate
	// values of ScrollID for for each call, but in reality a real interaction with ES would reuse the same
	// value of ScrollID for mutliple calls
	firstSearchResult := searchStart
	firstSearchResult.ScrollID = "lastSearch"
	lastSearchResult := ElasticSearchResults{
		ScrollID: "lastSearch",
	}
	mockLogDriver.On("ScrollSearch", searchStart.ScrollID).Return(firstSearchResult, nil)
	mockLogDriver.On("ScrollSearch", firstSearchResult.ScrollID).Return(lastSearchResult, nil)

	foundIndexedDay, numWarnings, err := exporter.retrieveLogs()

	c.Assert(foundIndexedDay, Equals, true)
	c.Assert(numWarnings, Equals, 0)
	c.Assert(err, IsNil)
	c.Assert(len(exporter.outputFiles), Equals, 1)
	c.Assert(exporter.outputFiles[0].ContainerID, Equals, "container1")
	c.Assert(exporter.outputFiles[0].LogFileName, Equals, "file1")
	c.Assert(exporter.outputFiles[0].LineCount, Equals, 1)
	c.Assert(exporter.outputFiles[0].ServiceID, Equals, "ServiceID")
}

// Same as the previous test, but tests multiple messages for the same file split across
//     more than one call to ScrollSearch()
func (s *TestAPISuite) TestLogs_RetrieveLogs_SearchFindsOneFileWithTwoScrolls(c *C) {
	exporter, mockLogDriver, err := setupSimpleRetrieveLogTest()
	defer func() {
		if exporter != nil {
			exporter.cleanup()
		}
	}()
	c.Assert(err, IsNil)
	source := setupOneSearchResult(c, "log", "host1", "ServiceID", "container1", "file1", "message1")
	searchStart := ElasticSearchResults{
		ScrollID: "search1",
		Hits: ExternalHit{
			Total:    Total{ Value: 1, Relation: ""},
			MaxScore: 0,
			Hits:     []InternalHit{
				{Source: (*json.RawMessage)(&source)},
			},
		},
	}

	mockLogDriver.On("StartSearch", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(searchStart, nil)

	// Because ScrollSearch() does not accept a pointer, we have to fake things a little by using separate
	// values of ScrollID for for each call, but in reality a real interaction with ES would reuse the same
	// value of ScrollID for mutliple calls
	firstSearchResult := searchStart
	firstSearchResult.ScrollID = "search2"
	secondSearchResult := searchStart
	secondSearchResult.ScrollID = "lastSearch"
	lastSearchResult := ElasticSearchResults{
		ScrollID: "lastSearch",
	}
	mockLogDriver.On("ScrollSearch", searchStart.ScrollID).Return(firstSearchResult, nil)
	mockLogDriver.On("ScrollSearch", firstSearchResult.ScrollID).Return(secondSearchResult, nil)
	mockLogDriver.On("ScrollSearch", secondSearchResult.ScrollID).Return(lastSearchResult, nil)

	foundIndexedDay, numWarnings, err := exporter.retrieveLogs()

	c.Assert(foundIndexedDay, Equals, true)
	c.Assert(numWarnings, Equals, 0)
	c.Assert(err, IsNil)
	c.Assert(len(exporter.outputFiles), Equals, 1)
	c.Assert(exporter.outputFiles[0].ContainerID, Equals, "container1")
	c.Assert(exporter.outputFiles[0].LogFileName, Equals, "file1")
	c.Assert(exporter.outputFiles[0].ServiceID, Equals, "ServiceID")
	c.Assert(exporter.outputFiles[0].LineCount, Equals, 2)
}

func (s *TestAPISuite) TestLogs_RetrieveLogs_SearchFindsTwoFiles(c *C) {
	exporter, mockLogDriver, err := setupSimpleRetrieveLogTest()
	defer func() {
		if exporter != nil {
			exporter.cleanup()
		}
	}()
	c.Assert(err, IsNil)
	source := setupOneSearchResult(c, "log", "host1", "ServiceID1", "container1", "file1", "message1")
	searchStart := ElasticSearchResults{
		ScrollID: "search1",
		Hits: ExternalHit{
			Total:    Total{ Value: 1, Relation: ""},
			MaxScore: 0,
			Hits:     []InternalHit{
				{Source: (*json.RawMessage)(&source)},
			},
		},
	}

	mockLogDriver.On("StartSearch", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(searchStart, nil)

	// Because ScrollSearch() does not accept a pointer, we have to fake things a little by using separate
	// values of ScrollID for for each call, but in reality a real interaction with ES would reuse the same
	// value of ScrollID for mutliple calls
	firstSearchResult := searchStart
	firstSearchResult.ScrollID = "search2"
	source = setupOneSearchResult(c, "log", "host2", "ServiceID2", "container2", "file2", "message1")
	secondSearchResult := ElasticSearchResults{
		ScrollID: "lastSearch",
		Hits: ExternalHit{
			Total:    Total{ Value: 1, Relation: ""},
			MaxScore: 0,
			Hits:     []InternalHit{
				{Source: (*json.RawMessage)(&source)},
			},
		},
	}

	lastSearchResult := ElasticSearchResults{
		ScrollID: "lastSearch",
	}
	mockLogDriver.On("ScrollSearch", searchStart.ScrollID).Return(firstSearchResult, nil)
	mockLogDriver.On("ScrollSearch", firstSearchResult.ScrollID).Return(secondSearchResult, nil)
	mockLogDriver.On("ScrollSearch", secondSearchResult.ScrollID).Return(lastSearchResult, nil)

	foundIndexedDay, numWarnings, err := exporter.retrieveLogs()

	c.Assert(foundIndexedDay, Equals, true)
	c.Assert(numWarnings, Equals, 0)
	c.Assert(err, IsNil)
	c.Assert(len(exporter.outputFiles), Equals, 1)

	c.Assert(exporter.outputFiles[0].HostID, Equals, "host2")
	c.Assert(exporter.outputFiles[0].ContainerID, Equals, "container2")
	c.Assert(exporter.outputFiles[0].LogFileName, Equals, "file2")
	c.Assert(exporter.outputFiles[0].LineCount, Equals, 2)
	c.Assert(exporter.outputFiles[0].ServiceID, Equals, "ServiceID2")
}

func (s *TestAPISuite) TestLogs_RetrieveLogs_ExcludesCCLogs(c *C) {
	exporter, mockLogDriver, err := setupSimpleRetrieveLogTest()
	defer func() {
		if exporter != nil {
			exporter.cleanup()
		}
	}()
	c.Assert(err, IsNil)

	// Note that the results for serviced and controller are using different files as way to verify
	// that only the first message is exported.
	source1 := setupOneSearchResult(c, "log", "host1", "ServiceID1", "container1", "file1", "message1")
	source2 := setupOneSearchResult(c, "serviced-cchost", "cchost", "", "", "file2", "cc message")
	source3 := setupOneSearchResult(c, "controller-controllerhost", "controllerhost", "", "", "file3", "controller message")
	searchStart := ElasticSearchResults{
		ScrollID: "search1",
		Hits: ExternalHit{
			Total:    Total{ Value: 3, Relation: ""},
			MaxScore: 0,
			Hits:     []InternalHit{
				{Source: (*json.RawMessage)(&source1)},
				{Source: (*json.RawMessage)(&source2)},
				{Source: (*json.RawMessage)(&source3)},
			},
		},
	}
	mockLogDriver.On("StartSearch", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(searchStart, nil)

	// Because ScrollSearch() does not accept a pointer, we have to fake things a little by using separate
	// values of ScrollID for for each call, but in reality a real interaction with ES would reuse the same
	// value of ScrollID for mutliple calls
	firstSearchResult := searchStart
	firstSearchResult.ScrollID = "lastSearch"
	lastSearchResult := ElasticSearchResults{
		ScrollID: "lastSearch",
	}
	mockLogDriver.On("ScrollSearch", searchStart.ScrollID).Return(firstSearchResult, nil)
	mockLogDriver.On("ScrollSearch", firstSearchResult.ScrollID).Return(lastSearchResult, nil)

	foundIndexedDay, numWarnings, err := exporter.retrieveLogs()

	c.Assert(foundIndexedDay, Equals, true)
	c.Assert(numWarnings, Equals, 0)
	c.Assert(err, IsNil)
	c.Assert(len(exporter.outputFiles), Equals, 1)
	c.Assert(exporter.outputFiles[0].HostID, Equals, "host1")
	c.Assert(exporter.outputFiles[0].ContainerID, Equals, "container1")
	c.Assert(exporter.outputFiles[0].LogFileName, Equals, "file1")
	c.Assert(exporter.outputFiles[0].LineCount, Equals, 1)
	c.Assert(exporter.outputFiles[0].ServiceID, Equals, "ServiceID1")
}

func (s *TestAPISuite) TestLogs_RetrieveLogs_ScrollFails(c *C) {
	exporter, mockLogDriver, err := setupSimpleRetrieveLogTest()
	defer func() {
		if exporter != nil {
			exporter.cleanup()
		}
	}()
	c.Assert(err, IsNil)

	source := []byte(`{"host": "container1", "file": "file1", "message": "message1", "service": "ServiceID"}`)

	searchStart := ElasticSearchResults{
		ScrollID: "search1",
		Hits: ExternalHit{
			Total:    Total{ Value: 1, Relation: ""},
			MaxScore: 0,
			Hits:     []InternalHit{
				{Source: (*json.RawMessage)(&source)},
			},
		},
	}

	mockLogDriver.On("StartSearch", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(searchStart, nil)
	expectedError := fmt.Errorf("ScrollSearch failed")
	mockLogDriver.On("ScrollSearch", searchStart.ScrollID).Return(ElasticSearchResults{}, expectedError)

	foundIndexedDay, numWarnings, err := exporter.retrieveLogs()

	c.Assert(foundIndexedDay, Equals, true)
	c.Assert(numWarnings, Equals, 0)
	c.Assert(err, Equals, expectedError)
}

func setupSimpleRetrieveLogTest() (*logExporter, *ExportLogDriverMock, error) {
	logstashDays := []string{"2112.01.01"}
	serviceIDs := []string{"someServiceID"}
	fromDate := logstashDays[0]
	toDate := logstashDays[0]
	return setupRetrieveLogTest(logstashDays, serviceIDs, fromDate, toDate)
}

func setupRetrieveLogTest(logstashDays, serviceIDs []string, fromDate, toDate string) (*logExporter, *ExportLogDriverMock, error) {
	mockLogDriver := &ExportLogDriverMock{}
	mockLogDriver.On("LogstashDays").Return(logstashDays, nil)

	config := ExportLogsConfig{
		ServiceIDs: serviceIDs,
		FromDate:   fromDate,
		ToDate:     toDate,
		Driver:     mockLogDriver,
		Debug:      true,
	}
	getServices := func() ([]service.ServiceDetails, error) {
		return []service.ServiceDetails{}, nil
	}
	getHostMap := func() (map[string]host.Host, error) {
		return make(map[string]host.Host), nil
	}

	exporter, err := buildExporter(config, time.Now().UTC(), "timestamp", getServices, getHostMap)
	return exporter, mockLogDriver, err
}

func setupOneSearchResult(c *C, logType, hostID, serviceID, containerID, fileName, message string) []byte {
	oneResultLine := logSingleLine{
		Type:    logType,
		File:    fileName,
		Message: message,
		FileBeat: beatProps{
			Name:     containerID,
			Hostname: containerID,
		},
		Fields: fieldProps{
			CCWorkerID: hostID,
			Service:    serviceID,
		},
	}

	jsonResult, err := json.Marshal(oneResultLine)
	c.Assert(err, IsNil)
	return jsonResult
}
