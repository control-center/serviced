package cmd

import (
	"encoding/json"
	"errors"
	"io"

	"github.com/zenoss/serviced/cli/api"
	service "github.com/zenoss/serviced/dao"
	template "github.com/zenoss/serviced/dao"
)

var DefaultTemplateAPITest = TemplateAPITest{templates: DefaultTestTemplates}

var DefaultTestTemplates = []*template.ServiceTemplate{
	{
		Id:          "test-template-1",
		Name:        "Alpha",
		Description: "example template 1",
	}, {
		Id:          "test-template-2",
		Name:        "Beta",
		Description: "example template 2",
	}, {
		Id:          "test-template-3",
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
	templates []*template.ServiceTemplate
}

func InitTemplateAPITest(args ...string) {
	New(DefaultTemplateAPITest).Run(args)
}

func (t TemplateAPITest) GetServiceTemplates() ([]*template.ServiceTemplate, error) {
	return t.templates, nil
}

func (t TemplateAPITest) GetServiceTemplate(id string) (*template.ServiceTemplate, error) {
	for i, template := range t.templates {
		if template.Id == id {
			return t.templates[i], nil
		}
	}

	return nil, ErrNoTemplateFound
}

func (t TemplateAPITest) AddServiceTemplate(r io.Reader) (*template.ServiceTemplate, error) {
	var template template.ServiceTemplate
	if err := json.NewDecoder(r).Decode(&template); err != nil {
		return nil, ErrInvalidTemplate
	}

	return &template, nil
}

func (t TemplateAPITest) RemoveServiceTemplate(id string) error {
	for _, template := range t.templates {
		if template.Id == id {
			return nil
		}
	}

	return ErrNoTemplateFound
}

func (t TemplateAPITest) CompileServiceTemplate(cfg api.CompileTemplateConfig) (*template.ServiceTemplate, error) {
	return nil, nil
}

func (t TemplateAPITest) DeployServiceTemplate(cfg api.DeployTemplateConfig) (*service.Service, error) {
	return nil, nil
}

func ExampleServicedCli_cmdTemplateList() {
	InitTemplateAPITest("serviced", "template", "list", "--verbose")

	// Output:
	// [
	//    {
	//      "Id": "test-template-1",
	//      "Name": "Alpha",
	//      "Description": "example template 1",
	//      "Services": null,
	//      "ConfigFiles": null
	//    },
	//    {
	//      "Id": "test-template-2",
	//      "Name": "Beta",
	//      "Description": "example template 2",
	//      "Services": null,
	//      "ConfigFiles": null
	//    },
	//    {
	//      "Id": "test-template-3",
	//      "Name": "Gamma",
	//      "Description": "example template 3",
	//      "Services": null,
	//      "ConfigFiles": null
	//    }
	//  ]
}

func ExampleServicedCli_cmdTemplateAdd() {
	InitTemplateAPITest("serviced", "template", "add")
}

func ExampleServicedCli_cmdTemplateRemove() {
	InitTemplateAPITest("serviced", "template", "remove", "test-template-1", "test-template-0")

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

func ExampleServicedCli_cmdTemplateDeploy() {
	InitTemplateAPITest("serviced", "template", "deploy")
}

func ExampleServicedCli_cmdTemplateCompile() {
	InitTemplateAPITest("serviced", "template", "compile")
}
