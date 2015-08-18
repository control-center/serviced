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

package cmd

import (
	"github.com/control-center/serviced/cli/api"
	"github.com/control-center/serviced/domain/service"
	template "github.com/control-center/serviced/domain/servicetemplate"
	"github.com/control-center/serviced/utils"

	"encoding/json"
	"errors"
	"fmt"
	"io"
	"testing"
)

const (
	NilTemplate = "NilTemplate"
)

var DefaultTemplateAPITest = TemplateAPITest{templates: DefaultTestTemplates}

var DefaultTestTemplates = []template.ServiceTemplate{
	{
		ID:          "test-template-1",
		Name:        "Alpha",
		Description: "example template 1",
	}, {
		ID:          "test-template-2",
		Name:        "Beta",
		Description: "example template 2",
	}, {
		ID:          "test-template-3",
		Name:        "Gamma",
		Description: "example template 3",
	},
}

var (
	ErrNoTemplateFound = errors.New("no templates found")
	ErrInvalidTemplate = errors.New("invalid template")
)

type TemplateAPITest struct {
	api.API
	fail      bool
	templates []template.ServiceTemplate
}

func InitTemplateAPITest(args ...string) {
	New(DefaultTemplateAPITest, utils.TestConfigReader(make(map[string]string))).Run(args)
}

func (t TemplateAPITest) GetServiceTemplates() ([]template.ServiceTemplate, error) {
	if t.fail {
		return nil, ErrInvalidTemplate
	}
	return t.templates, nil
}

func (t TemplateAPITest) GetServiceTemplate(id string) (*template.ServiceTemplate, error) {
	if t.fail {
		return nil, ErrInvalidTemplate
	}

	for i, template := range t.templates {
		if template.ID == id {
			return &t.templates[i], nil
		}
	}

	return nil, nil
}

func (t TemplateAPITest) AddServiceTemplate(r io.Reader) (*template.ServiceTemplate, error) {
	var template template.ServiceTemplate
	if err := json.NewDecoder(r).Decode(&template); err != nil {
		return nil, ErrInvalidTemplate
	} else if template.ID != "" {
		return nil, nil
	}
	return &template, nil
}

func (t TemplateAPITest) RemoveServiceTemplate(id string) error {
	if t, err := t.GetServiceTemplate(id); err != nil {
		return err
	} else if t == nil {
		return ErrNoTemplateFound
	}
	return nil
}

func (t TemplateAPITest) CompileServiceTemplate(cfg api.CompileTemplateConfig) (*template.ServiceTemplate, error) {
	if t.fail {
		return nil, ErrInvalidTemplate
	} else if cfg.Dir == NilTemplate {
		return nil, nil
	}

	tpl := template.ServiceTemplate{
		ID: fmt.Sprintf("%s-template", cfg.Dir),
	}
	return &tpl, nil
}

func (t TemplateAPITest) DeployServiceTemplate(cfg api.DeployTemplateConfig) ([]service.Service, error) {
	tpl, err := t.GetServiceTemplate(cfg.ID)
	if err != nil {
		return nil, err
	} else if tpl == nil {
		return nil, nil
	}
	s := service.Service{
		ID:     fmt.Sprintf("%s-service", cfg.ID),
		PoolID: cfg.PoolID,
	}
	return []service.Service{s}, nil
}

func TestServicedCLI_CmdTemplateList_one(t *testing.T) {
	templateID := "test-template-1"

	expected, err := DefaultTemplateAPITest.GetServiceTemplate(templateID)
	if err != nil {
		t.Fatal(err)
	}

	var actual template.ServiceTemplate
	output := pipe(InitTemplateAPITest, "serviced", "template", "list", templateID)

	if err := json.Unmarshal(output, &actual); err != nil {
		t.Fatalf("error unmarshaling resource: %s", err)
	}

	// Did you remember to update ServiceTemplate.Equals?
	if !actual.Equals(expected) {
		t.Fatalf("got:\n%+v\nwant:\n%+v", actual, expected)
	}
}

func TestServicedCLI_CmdTemplateList_all(t *testing.T) {
	expected, err := DefaultTemplateAPITest.GetServiceTemplates()
	if err != nil {
		t.Fatal(err)
	}

	var actual []*template.ServiceTemplate
	output := pipe(InitTemplateAPITest, "serviced", "template", "list", "--verbose")
	if err := json.Unmarshal(output, &actual); err != nil {
		t.Fatalf("error unmarshaling resource: %s", err)
	}

	// Did you remember to update ServiceTemplate.Equals?
	if len(actual) != len(expected) {
		t.Fatalf("got:\n%+v\nwant:\n%+v", actual, expected)
	}
	for i := range actual {
		if !actual[i].Equals(&expected[i]) {
			t.Fatalf("got:\n%+v\nwant:\n%+v", actual, expected)
		}
	}
}

func ExampleServicedCLI_CmdTemplateList() {
	// Gofmt cleans up the spaces at the end of each row
	InitTemplateAPITest("serviced", "template", "list")
}

func ExampleServicedCLI_CmdTemplateList_fail() {
	DefaultTemplateAPITest.fail = true
	defer func() { DefaultTemplateAPITest.fail = false }()
	// Error retrieving template
	pipeStderr(InitTemplateAPITest, "serviced", "template", "list", "test-template-1")
	// Error retrieving all templates
	pipeStderr(InitTemplateAPITest, "serviced", "template", "list")

	// Output:
	// invalid template
	// invalid template
}

func ExampleServicedCLI_CmdTemplateList_err() {
	DefaultTemplateAPITest.templates = nil
	defer func() { DefaultTemplateAPITest.templates = DefaultTestTemplates }()
	// template not found
	pipeStderr(InitTemplateAPITest, "serviced", "template", "list", "test-template-0")
	// no templates found
	pipeStderr(InitTemplateAPITest, "serviced", "template", "list")

	// Output:
	// template not found
	// no templates found
}

func ExampleServicedCLI_CmdTemplateAdd() {
	InitTemplateAPITest("serviced", "template", "add")
}

func ExampleServicedCLI_CmdTemplateAdd_fail() {
	InitTemplateAPITest("serviced", "template", "add")
}

func ExampleServicedCLI_CmdTemplateAdd_err() {
	InitTemplateAPITest("serviced", "template", "add")
}

func ExampleServicedCLI_CmdTemplateRemove() {
	InitTemplateAPITest("serviced", "template", "remove", "test-template-1")

	// Output:
	// test-template-1
}

func ExampleServicedCLI_CmdTemplateRemove_usage() {
	InitTemplateAPITest("serviced", "template", "remove")

	// Output:
	// Incorrect Usage.
	//
	// NAME:
	//    remove - Remove an existing template
	//
	// USAGE:
	//    command remove [command options] [arguments...]
	//
	// DESCRIPTION:
	//    serviced template remove TEMPLATEID ...
	//
	// OPTIONS:
}

func ExampleServicedCLI_CmdTemplateRemove_err() {
	pipeStderr(InitTemplateAPITest, "serviced", "template", "remove", "test-template-0")

	// Output:
	// test-template-0: no templates found
}

func ExampleServicedCLI_CmdTemplateDeploy() {
	InitTemplateAPITest("serviced", "template", "deploy", "test-template-1", "test-pool", "deployment-id")

	// Output:
	// test-template-1-service
}

func ExampleServicedCLI_CmdTemplateDeploy_usage() {
	InitTemplateAPITest("serviced", "template", "deploy")

	// Output:
	// Incorrect Usage.
	//
	// NAME:
	//    deploy - Deploys a template's services to a pool
	//
	// USAGE:
	//    command deploy [command options] [arguments...]
	//
	// DESCRIPTION:
	//    serviced template deploy TEMPLATEID POOLID DEPLOYMENTID
	//
	// OPTIONS:
	//    --manual-assign-ips	Manually assign IP addresses
}

func ExampleServicedCLI_CmdTemplateDeploy_fail() {
	DefaultTemplateAPITest.fail = true
	defer func() { DefaultTemplateAPITest.fail = false }()
	pipeStderr(InitTemplateAPITest, "serviced", "template", "deploy", "test-template-1", "test-pool", "deployment-id")

	// Output:
	// Deploying template - please wait...
	// invalid template
}

func ExampleServicedCLI_CmdTemplateDeploy_err() {
	pipeStderr(InitTemplateAPITest, "serviced", "template", "deploy", NilTemplate, "test-pool", "deployment-id")

	// Output:
	// Deploying template - please wait...
	// received nil service definition
}

func TestServicedCLI_CmdTemplateCompile(t *testing.T) {
	dir := "/path/to/template"

	expected, err := DefaultTemplateAPITest.CompileServiceTemplate(api.CompileTemplateConfig{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}

	var actual template.ServiceTemplate
	output := pipe(InitTemplateAPITest, "serviced", "template", "compile", dir)
	if err := json.Unmarshal(output, &actual); err != nil {
		t.Fatalf("error unmarshaling resource: %s", err)
	}

	// Did you remember to update ServiceTemplate.Equals?
	if !actual.Equals(expected) {
		t.Fatalf("got:\n%+v\nwant:\n%+v", actual, expected)
	}
}

func ExampleServicedCLI_CmdTemplateCompile_usage() {
	InitTemplateAPITest("serviced", "template", "compile")

	// Output:
	// Incorrect Usage.
	//
	// NAME:
	//    compile - Convert a directory of service definitions into a template
	//
	// USAGE:
	//    command compile [command options] [arguments...]
	//
	// DESCRIPTION:
	//    serviced template compile PATH
	//
	// OPTIONS:
	//    --map 	`-map option -map option` Map a given image name to another (e.g. -map zenoss/zenoss5x:latest,quay.io/zenoss-core:alpha2)
}

func ExampleServicedCLI_CmdTemplateCompile_fail() {
	DefaultTemplateAPITest.fail = true
	defer func() { DefaultTemplateAPITest.fail = false }()
	pipeStderr(InitTemplateAPITest, "serviced", "template", "compile", "/path/to/template")

	// Output:
	// invalid template
}

func ExampleServicedCLI_CmdTemplateCompile_err() {
	pipeStderr(InitTemplateAPITest, "serviced", "template", "compile", NilTemplate)

	// Output:
	// received nil template
}
