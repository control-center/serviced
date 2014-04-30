package cmd

import (
	"errors"
	"io"

	service "github.com/zenoss/serviced/dao"
	template "github.com/zenoss/serviced/dao"
	"github.com/zenoss/serviced/cli/api"
)

var DefaultTemplateAPITest = TemplateAPITest{}

var DefaultTestTemplates = []*template.ServiceTemplate{}

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
	return nil, nil
}

func (t TemplateAPITest) GetServiceTemplate(id string) (*template.ServiceTemplate, error) {
	return nil, nil
}

func (t TemplateAPITest) AddServiceTemplate(r io.Reader) (*template.ServiceTemplate, error) {
	return nil, nil
}

func (t TemplateAPITest) RemoveServiceTemplate(id string) error {
	return nil
}

func (t TemplateAPITest) CompileServiceTemplate(cfg api.CompileTemplateConfig) (*template.ServiceTemplate, error) {
	return nil, nil
}

func (t TemplateAPITest) DeployServiceTemplate(cfg api.DeployTemplateConfig) (*service.Service, error) {
	return nil, nil
}

func ExampleServicedCli_cmdTemplateList() {
	InitTemplateAPITest("serviced", "template", "list")
}

func ExampleServicedCli_cmdTemplateAdd() {
	InitTemplateAPITest("serviced", "template", "add")
}

func ExampleServicedCli_cmdTemplateRemove() {
	InitTemplateAPITest("serviced", "template", "remove")
	InitTemplateAPITest("serviced", "template", "rm")
}

func ExampleServicedCli_cmdTemplateDeploy() {
	InitTemplateAPITest("serviced", "template", "deploy")
}

func ExampleServicedCli_cmdTemplateCompile() {
	InitTemplateAPITest("serviced", "template", "compile")
}
