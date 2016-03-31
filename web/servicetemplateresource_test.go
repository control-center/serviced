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
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"

	"github.com/control-center/serviced/domain/servicedefinition"
	"github.com/control-center/serviced/domain/servicetemplate"
	"github.com/stretchr/testify/mock"
	"github.com/zenoss/go-json-rest"
	. "gopkg.in/check.v1"
)

func (s *TestWebSuite) TestRestAddAppTemplate(c *C) {
	expectedTemplateID := "someTemplateID"
	jsonBuffer, err := getTestTemplateJson()
	if err != nil {
		c.Fatalf("Unable to build JSON for test template: %s", err)
	}
	request, err := buildUploadRequest("POST", "/templates/add", jsonBuffer)
	if err != nil {
		c.Fatalf("Unable to build mock upload request: %s", err)
	}
	s.mockFacade.
		On("AddServiceTemplate", s.ctx.getDatastoreContext(), mock.AnythingOfType("servicetemplate.ServiceTemplate")).
		Return(expectedTemplateID, nil)

	restAddAppTemplate(&(s.writer), &request, s.ctx)

	c.Assert(s.recorder.Code, Equals, http.StatusOK)
	s.assertSimpleResponse(c, expectedTemplateID, servicesLinks())
}

func (s *TestWebSuite) TestRestAddAppTemplateFails(c *C) {
	expectedError := fmt.Errorf("mock AddServiceTemplate failed")
	jsonBuffer, err := getTestTemplateJson()
	if err != nil {
		c.Fatalf("Unable to build JSON for test template: %s", err)
	}
	request, err := buildUploadRequest("POST", "/templates/add", jsonBuffer)
	if err != nil {
		c.Fatalf("Unable to build mock upload request: %s", err)
	}
	s.mockFacade.
		On("AddServiceTemplate", s.ctx.getDatastoreContext(), mock.AnythingOfType("servicetemplate.ServiceTemplate")).
		Return("", expectedError)

	restAddAppTemplate(&(s.writer), &request, s.ctx)

	s.assertServerError(c, expectedError)
}

func (s *TestWebSuite) TestRestAddAppTemplateFailsForBadForm(c *C) {
	request := s.buildRequest("POST", "/templates/add", "")

	restAddAppTemplate(&(s.writer), &request, s.ctx)

	s.assertBadRequest(c)
}

func (s *TestWebSuite) TestRestAddAppTemplateFailsForBadJson(c *C) {
	expectedError := fmt.Errorf("invalid character 't' looking for beginning of object key string")
	request, err := buildUploadRequest("POST", "/templates/add", bytes.NewBufferString("{this is not valid json}"))
	if err != nil {
		c.Fatalf("Unable to build mock upload request: %s", err)
	}

	restAddAppTemplate(&(s.writer), &request, s.ctx)

	s.assertServerError(c, expectedError)
}

// Build a multi-part form request containing a JSON service template
func buildUploadRequest(action, url string, templateJSON *bytes.Buffer) (rest.Request, error) {
	uploadBody := &bytes.Buffer{}
	writer := multipart.NewWriter(uploadBody)

	part, err := writer.CreateFormFile("tpl", "unusedFileNameValue")
	if err != nil {
		return rest.Request{}, fmt.Errorf("CreateFormFile failed: %s", err)
	}

	n, err := io.Copy(part, templateJSON)
	if err != nil {
		return rest.Request{}, fmt.Errorf("Json copy failed: %s", err)
	} else if n == 0 {
		return rest.Request{}, fmt.Errorf("Json copy failed: 0 bytes copied")
	}

	writer.Close()
	httpRequest, err := http.NewRequest(action, url, uploadBody)
	if err != nil {
		return rest.Request{}, fmt.Errorf("Could not build new request: %s", err)
	}

	httpRequest.Header.Add("Content-Type", writer.FormDataContentType())
	return rest.Request{
		httpRequest,
		map[string]string{},
	}, nil
}

func getTestTemplateJson() (*bytes.Buffer, error) {
	jsonBuffer := &bytes.Buffer{}
	encoder := json.NewEncoder(jsonBuffer)
	err := encoder.Encode(getTestTemplate())
	if err != nil {
		return nil, err
	}
	return jsonBuffer, nil
}

func getTestTemplate() servicetemplate.ServiceTemplate {
	sd := servicedefinition.ServiceDefinition{
		Name:        "testTemplateName",
		Description: "test template",
	}
	st := servicetemplate.ServiceTemplate{
		Services:    []servicedefinition.ServiceDefinition{sd},
		Name:        sd.Name,
		Version:     sd.Version,
		Description: sd.Description,
	}
	return st
}
