package dao

import (
	"bytes"
	"encoding/json"
	"github.com/zenoss/glog"
	"text/template"
)

func (a *Service) Equals(b *Service) bool {
	if a.Id != b.Id {
		return false
	}
	if a.Name != b.Name {
		return false
	}
	if a.Context != b.Context {
		return false
	}
	if a.Startup != b.Startup {
		return false
	}
	if a.Description != b.Description {
		return false
	}
	if a.Instances != b.Instances {
		return false
	}
	if a.ImageId != b.ImageId {
		return false
	}
	if a.PoolId != b.PoolId {
		return false
	}
	if a.DesiredState != b.DesiredState {
		return false
	}
	if a.Launch != b.Launch {
		return false
	}
	if a.Hostname != b.Hostname {
		return false
	}
	if a.Privileged != b.Privileged {
		return false
	}
	if a.ParentServiceId != b.ParentServiceId {
		return false
	}
	if a.CreatedAt != b.CreatedAt {
		return false
	}
	if a.UpdatedAt != b.CreatedAt {
		return false
	}
	return true
}

func parent(cp ControlPlane) func(s Service) (value Service, err error) {
	return func(s Service) (value Service, err error) {
		err = cp.GetService(s.ParentServiceId, &value)
		return
	}
}

func context(cp ControlPlane) func(s Service) (ctx map[string]interface{}, err error) {
	return func(s Service) (ctx map[string]interface{}, err error) {
		err = json.Unmarshal([]byte(s.Context), &ctx)
		if err != nil {
			glog.Errorf("Error unmarshal service context Id=%s: %s -> %s", s.Id, s.Context, err)
		}
		return
	}
}

// EvaluateActionsTemplate parses and evaluates the Actions string of a service.
func (service *Service) EvaluateActionsTemplate(cp ControlPlane) (err error) {
	for key, value := range service.Actions {
		result := service.evaluateTemplate(cp, value)
		if result != "" {
			service.Actions[key] = result
		}
	}
	return
}

// EvaluateStartupTemplate parses and evaluates the StartUp string of a service.
func (service *Service) EvaluateStartupTemplate(cp ControlPlane) (err error) {

	result := service.evaluateTemplate(cp, service.Startup)
	if result != "" {
		service.Startup = result
	}

	return
}

// evaluateTemplate takes a control plane client and template string and evaluates
// the template using the service as the context. If the template is invalid or there is an error
// then an empty string is returned.
func (service *Service) evaluateTemplate(cp ControlPlane, serviceTemplate string) string {
	functions := template.FuncMap{
		"parent":  parent(cp),
		"context": context(cp),
	}
	// parse the template
	t := template.Must(template.New("ServiceDefinitionTemplate").Funcs(functions).Parse(serviceTemplate))

	// evaluate it
	var buffer bytes.Buffer
	err := t.Execute(&buffer, service)
	if err == nil {
		return buffer.String()
	}

	// something went wrong, warn them
	glog.Warning("Evaluating template %s produced the following error %s ", serviceTemplate, err)
	return ""
}

// EvaluateLogConfigTemplate parses and evals the Path, Type and all the values for the tags of the log
// configs. This happens for each LogConfig on the service.
func (service *Service) EvaluateLogConfigTemplate(cp ControlPlane) (err error) {
	// evaluate the template for the LogConfig as well as the tags

	for i, logConfig := range service.LogConfigs {
		// Path
		result := service.evaluateTemplate(cp, logConfig.Path)
		if result != "" {
			service.LogConfigs[i].Path = result
		}
		// Type
		result = service.evaluateTemplate(cp, logConfig.Type)
		if result != "" {
			service.LogConfigs[i].Type = result
		}

		// Tags
		for j, tag := range logConfig.LogTags {
			result = service.evaluateTemplate(cp, tag.Value)
			if result != "" {
				service.LogConfigs[i].LogTags[j].Value = result
			}
		}
	}
	return
}

// EvaluateEndpointTemplates parses and evaluates the "ApplicationTemplate" property
// of each of the service endpoints for this service.
func (service *Service) EvaluateEndpointTemplates(cp ControlPlane) (err error) {
	functions := template.FuncMap{
		"parent":  parent(cp),
		"context": context(cp),
	}

	for i, ep := range service.Endpoints {
		if ep.Application != "" && ep.ApplicationTemplate == "" {
			ep.ApplicationTemplate = ep.Application
			service.Endpoints[i].ApplicationTemplate = ep.Application
		}
		if ep.ApplicationTemplate != "" {
			t := template.Must(template.New(service.Name).Funcs(functions).Parse(ep.ApplicationTemplate))
			var buffer bytes.Buffer
			if err = t.Execute(&buffer, service); err == nil {
				service.Endpoints[i].Application = buffer.String()
			} else {
				return
			}
		}
	}
	return
}
