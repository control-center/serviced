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

package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/control-center/serviced/datastore"
	datastoreMocks "github.com/control-center/serviced/datastore/mocks"
	facadeMocks "github.com/control-center/serviced/facade/mocks"
	"github.com/zenoss/go-json-rest"
	. "gopkg.in/check.v1"
)

// Wire gocheck into the go test runner
func TestWeb(t *testing.T) { TestingT(t) }

type TestWebSuite struct{
	ctx        *requestContext
	recorder   *httptest.ResponseRecorder
	writer     rest.ResponseWriter
	mockDriver *datastoreMocks.Driver
	mockFacade *facadeMocks.FacadeInterface
}

// verify TestWebSuite implements the Suite interface
var _ = Suite(&TestWebSuite{})

func (s *TestWebSuite) SetUpTest(c *C) {
	s.mockDriver = &datastoreMocks.Driver{}
	datastore.Register(s.mockDriver)

	s.mockFacade = &facadeMocks.FacadeInterface{}
	config := ServiceConfig{facade: s.mockFacade}
	s.ctx = newRequestContext(&config)

	s.recorder = httptest.NewRecorder()
	s.writer = rest.NewResponseWriter(s.recorder, false)
}

func (s *TestWebSuite) TearDownTest(c *C) {
	s.ctx = nil
	s.recorder = nil
	s.writer = rest.ResponseWriter{}
	s.mockDriver = nil
	s.mockFacade = nil
}

// Build a REST request suitable for testing.
func (s *TestWebSuite) buildRequest(action, url, payload string) rest.Request {
	reader := strings.NewReader(payload)
	httpRequest, _ := http.NewRequest(action, url, reader)
	return rest.Request{
		httpRequest,
		map[string]string{},
	}
}

func (s *TestWebSuite) getResult(c *C, result interface{}) {
	body := s.recorder.Body.String()
	c.Assert(len(body), Not(Equals), 0)

	err := json.Unmarshal([]byte(body), result)
	c.Assert(err, IsNil)
}

func (s *TestWebSuite) assertMapKeys(c *C, actual interface{}, expected interface{}) {
	actualMap := reflect.ValueOf(actual)
	expectedMap := reflect.ValueOf(expected)
	c.Assert(len(actualMap.MapKeys()), Equals, len(expectedMap.MapKeys()))
	for _, key := range expectedMap.MapKeys() {
		v := actualMap.MapIndex(key)
		if !v.IsValid() {
			c.Logf("Could not find key %s", key)
		}
		c.Assert(v.IsValid(), Equals, true)
	}
}

// Verify that the Body string from the recorder is a valid 'simpleResponse' containing
func (s *TestWebSuite) assertSimpleResponse(c *C, expectedDetails string, expectedLinks []link) {
	body := s.recorder.Body.String()
	c.Assert(len(body), Not(Equals), 0)

	response := simpleResponse{}
	err := json.Unmarshal([]byte(body), &response)
	c.Assert(err, IsNil)

	c.Assert(response.Detail, Matches, expectedDetails)
	s.assertLinks(c, response.Links, expectedLinks)
}

func (s *TestWebSuite) assertLinks(c *C, actual, expected []link) {
	c.Assert(len(actual), Equals, len(expected))
	c.Assert(actual, DeepEquals, expected)
}

func (s *TestWebSuite) assertBadRequest(c *C) {
	c.Assert(s.recorder.Code, Equals, http.StatusInternalServerError)
	s.assertSimpleResponse(c, "Bad Request: .*",  homeLink())
}

func (s *TestWebSuite) assertServerError(c *C, expectedError error) {
	c.Assert(s.recorder.Code, Equals, http.StatusInternalServerError)
	s.assertSimpleResponse(c, fmt.Sprintf("Internal Server Error: %s", expectedError),  homeLink())
}

// Some methods return a slightly different response on error
func (s *TestWebSuite) assertAltServerError(c *C, expectedError error) {
	c.Assert(s.recorder.Code, Equals, http.StatusInternalServerError)
	s.assertSimpleResponse(c, expectedError.Error(),  homeLink())
}
