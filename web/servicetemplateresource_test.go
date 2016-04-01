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

func (s *TestWebSuite) TestRestGetAppTemplates(c *C) {
	expectedTemplates := []servicetemplate.ServiceTemplate{
		{ID: "template1"},
		{ID: "template2"},
	}
	expectedResult := map[string]servicetemplate.ServiceTemplate{
		"template1": expectedTemplates[0],
		"template2": expectedTemplates[1],
	}
	request := s.buildRequest("GET", "/templates", "")
	s.mockFacade.
		On("GetServiceTemplates", s.ctx.getDatastoreContext()).
		Return(expectedResult, nil)

	restGetAppTemplates(&(s.writer), &request, s.ctx)

	c.Assert(s.recorder.Code, Equals, http.StatusOK)
	actualResult := map[string]servicetemplate.ServiceTemplate{}
	s.getResult(c, &actualResult)
	s.assertMapKeys(c, actualResult, expectedResult)
	for templateID, template := range actualResult {
		c.Assert(template.ID, Equals, expectedResult[templateID].ID)
	}
}

func (s *TestWebSuite) TestRestGetAppTemplatesFails(c *C) {
	expectedError := fmt.Errorf("mock GetServiceTemplates failed")
	request := s.buildRequest("GET", "/templates", "")
	s.mockFacade.
		On("GetServiceTemplates", s.ctx.getDatastoreContext()).
		Return(nil, expectedError)

	restGetAppTemplates(&(s.writer), &request, s.ctx)

	s.assertServerError(c, expectedError)
}

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

func (s *TestWebSuite) TestRestRemoveAppTemplate(c *C) {
	templateID := "someTemplateID"
	request := s.buildRequest("DELETE", "/templates/someTemplateID", "")
	request.PathParams["templateId"] = templateID
	s.mockFacade.
		On("RemoveServiceTemplate", s.ctx.getDatastoreContext(), templateID).
		Return(nil)

	restRemoveAppTemplate(&(s.writer), &request, s.ctx)

	c.Assert(s.recorder.Code, Equals, http.StatusOK)
	s.assertSimpleResponse(c, templateID, servicesLinks())
}

func (s *TestWebSuite) TestRestRemoveAppTemplateFails(c *C) {
	expectedError := fmt.Errorf("mock RemoveServiceTemplate failed")
	templateID := "someTemplateID"
	request := s.buildRequest("DELETE", "/templates/someTemplateID", "")
	request.PathParams["templateId"] = templateID
	s.mockFacade.
		On("RemoveServiceTemplate", s.ctx.getDatastoreContext(), templateID).
		Return(expectedError)

	restRemoveAppTemplate(&(s.writer), &request, s.ctx)

	s.assertServerError(c, expectedError)
}

func (s *TestWebSuite) TestRestRemoveAppTemplateFailsForInvalidURL(c *C) {
	request := s.buildRequest("DELETE", "/templates/%zzz", "")
	request.PathParams["templateId"] = "%zzz"

	restRemoveAppTemplate(&(s.writer), &request, s.ctx)

	s.assertBadRequest(c)
}

func (s *TestWebSuite) TestRestDeployAppTemplate(c *C) {
	expectedResult := []string{
		"someTenantID",
	}
	payload := servicetemplate.ServiceTemplateDeploymentRequest{
		PoolID:       "somePoolID",
		TemplateID:   "someTemplateID",
		DeploymentID: "someDeploymentID",
	}
	jsonPayload, err := json.Marshal(&payload)
	if err != nil {
		c.Fatalf("Failed to marshall JSON: %s", err)
	}
	request := s.buildRequest("GET", "/templates/deploy", string(jsonPayload))
	s.mockFacade.
		On("DeployTemplate", s.ctx.getDatastoreContext(), payload.PoolID, payload.TemplateID, payload.DeploymentID).
		Return(expectedResult, nil)
	s.mockFacade.
		On("AssignIPs", s.ctx.getDatastoreContext(), mock.AnythingOfType("addressassignment.AssignmentRequest")).
		Return(nil)

	restDeployAppTemplate(&(s.writer), &request, s.ctx)

	c.Assert(s.recorder.Code, Equals, http.StatusOK)
	s.assertSimpleResponse(c, expectedResult[0], servicesLinks())
}

func (s *TestWebSuite) TestRestDeployAppTemplateFails(c *C) {
	expectedError := fmt.Errorf("mock DeployTemplate failed")
	request := s.buildRequest("GET", "/templates/deploy", `{"DeploymentID": "someID"}`)
	s.mockFacade.
		On("DeployTemplate", s.ctx.getDatastoreContext(), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string")).
		Return(nil, expectedError)

	restDeployAppTemplate(&(s.writer), &request, s.ctx)

	s.assertServerError(c, expectedError)
}

func (s *TestWebSuite) TestRestDeployAppTemplateFailsForBadJSON(c *C) {
	request := s.buildRequest("POST", "/templates/deploy", "{this is not valid json}")

	restDeployAppTemplate(&(s.writer), &request, s.ctx)

	s.assertBadRequest(c)
}

func (s *TestWebSuite) TestRestDeployAppTemplateStatus(c *C) {
	expectedResult := "ok"
	deploymentID := "someDeploymentID"
	requestJSON := `{"DeploymentID": "` + deploymentID + `"}`
	request := s.buildRequest("POST", "/templates/deploy/status", requestJSON)
	s.mockFacade.
		On("DeployTemplateStatus", deploymentID).
		Return(expectedResult, nil)

	restDeployAppTemplateStatus(&(s.writer), &request, s.ctx)

	c.Assert(s.recorder.Code, Equals, http.StatusOK)
	s.assertSimpleResponse(c, expectedResult, servicesLinks())
}

func (s *TestWebSuite) TestRestDeployAppTemplateStatusFails(c *C) {
	expectedError := fmt.Errorf("mock DeployTemplateStatus failed")
	deploymentID := "someDeploymentID"
	requestJSON := `{"DeploymentID": "` + deploymentID + `"}`
	request := s.buildRequest("POST", "/templates/deploy/status", requestJSON)
	s.mockFacade.
		On("DeployTemplateStatus", deploymentID).
		Return(nil, expectedError)

	restDeployAppTemplateStatus(&(s.writer), &request, s.ctx)

	s.assertAltServerError(c, expectedError)
}

func (s *TestWebSuite) TestRestDeployAppTemplateStatusFailsForBadJSON(c *C) {
	request := s.buildRequest("POST", "/templates/deploy/status", "{this is not valid json}")

	restDeployAppTemplateStatus(&(s.writer), &request, s.ctx)

	s.assertBadRequest(c)
}

func (s *TestWebSuite) TestRestDeployAppTemplateActive(c *C) {
	expectedResult := make([]map[string]string, 1)
	request := s.buildRequest("GET", "/templates/deploy/active", "")
	s.mockFacade.
		On("DeployTemplateActive").
		Return(expectedResult, nil)

	restDeployAppTemplateActive(&(s.writer), &request, s.ctx)

	c.Assert(s.recorder.Code, Equals, http.StatusOK)
	var actualResult []map[string]string
	s.getResult(c, &actualResult)
	c.Assert(actualResult, DeepEquals, expectedResult)
}

func (s *TestWebSuite) TestRestDeployAppTemplateActiveFails(c *C) {
	expectedError := fmt.Errorf("mock DeployTemplateActive failed")
	request := s.buildRequest("GET", "/templates/deploy/active", "")
	s.mockFacade.
		On("DeployTemplateActive").
		Return(nil, expectedError)

	restDeployAppTemplateActive(&(s.writer), &request, s.ctx)

	s.assertAltServerError(c, expectedError)
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
