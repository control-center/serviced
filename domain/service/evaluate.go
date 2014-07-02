// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package service

import (
	"github.com/zenoss/glog"

	"bytes"
	"encoding/json"
	"text/template"
)

func parent(gs GetService) func(s Service) (Service, error) {
	return func(svc Service) (Service, error) {
		return gs(svc.ParentServiceID)
	}
}
func context() func(s Service) (map[string]interface{}, error) {
	return func(s Service) (map[string]interface{}, error) {
		ctx := make(map[string]interface{})
		err := json.Unmarshal([]byte(s.Context), &ctx)
		if err != nil {
			glog.Errorf("Error unmarshal service context ID=%s: %s -> %s", s.ID, s.Context, err)
		}
		return ctx, err
	}
}

// EvaluateActionsTemplate parses and evaluates the Actions string of a service.
func (service *Service) EvaluateActionsTemplate(gs GetService) (err error) {
	for key, value := range service.Actions {
		result := service.evaluateTemplate(gs, value)
		if result != "" {
			service.Actions[key] = result
		}
	}
	return
}

// EvaluateStartupTemplate parses and evaluates the StartUp string of a service.
func (service *Service) EvaluateStartupTemplate(gs GetService) (err error) {

	result := service.evaluateTemplate(gs, service.Startup)
	if result != "" {
		service.Startup = result
	}

	return
}

// EvaluateRunsTemplate parses and evaluates the Runs string of a service.
func (service *Service) EvaluateRunsTemplate(gs GetService) (err error) {
	for key, value := range service.Runs {
		result := service.evaluateTemplate(gs, value)
		if result != "" {
			service.Runs[key] = result
		}
	}
	return
}

// evaluateTemplate takes a control plane client and template string and evaluates
// the template using the service as the context. If the template is invalid or there is an error
// then an empty string is returned.
func (service *Service) evaluateTemplate(gs GetService, serviceTemplate string) string {
	functions := template.FuncMap{
		"parent":       parent(gs),
		"context":      context(),
		"percentScale": percentScale,
		"bytesToMB":    bytesToMB,
	}

	glog.V(3).Infof("Evaluating template string %v", serviceTemplate)
	// parse the template
	t := template.Must(template.New("ServiceDefinitionTemplate").Funcs(functions).Parse(serviceTemplate))

	// evaluate it
	var buffer bytes.Buffer
	err := t.Execute(&buffer, service)
	if err == nil {
		return buffer.String()
	}

	// something went wrong, warn them
	glog.Warningf("Evaluating template %v produced the following error %v ", serviceTemplate, err)
	return ""
}

// EvaluateLogConfigTemplate parses and evals the Path, Type and all the values for the tags of the log
// configs. This happens for each LogConfig on the service.
func (service *Service) EvaluateLogConfigTemplate(gs GetService) (err error) {
	// evaluate the template for the LogConfig as well as the tags
	for i, logConfig := range service.LogConfigs {
		// Path
		result := service.evaluateTemplate(gs, logConfig.Path)
		if result != "" {
			service.LogConfigs[i].Path = result
		}
		// Type
		result = service.evaluateTemplate(gs, logConfig.Type)
		if result != "" {
			service.LogConfigs[i].Type = result
		}

		// Tags
		for j, tag := range logConfig.LogTags {
			result = service.evaluateTemplate(gs, tag.Value)
			if result != "" {
				service.LogConfigs[i].LogTags[j].Value = result
			}
		}
	}
	return
}

// EvaluateConfigFilesTemplate parses and evals the Filename and Content. This happens for each
// ConfigFile on the service.
func (service *Service) EvaluateConfigFilesTemplate(gs GetService) (err error) {
	glog.V(3).Infof("Evaluating Config Files for %s", service.ID)
	for key, configFile := range service.ConfigFiles {
		glog.V(3).Infof("Evaluating Config File: %v", key)
		// Filename
		result := service.evaluateTemplate(gs, configFile.Filename)
		if result != "" {
			configFile.Filename = result
		}
		// Content
		result = service.evaluateTemplate(gs, configFile.Content)
		if result != "" {
			configFile.Content = result
		}
		service.ConfigFiles[key] = configFile
	}
	return
}

func percentScale(x uint64, percentage float64) uint64 {
	return uint64(round(float64(x) * percentage))
}

func bytesToMB(x uint64) uint64 {
	return uint64(round(float64(x) / (1048576.0))) // 1024.0 * 1024
}

// round value - convert to int64
func round(value float64) int64 {
	if value < 0.0 {
		value -= 0.5
	} else {
		value += 0.5
	}
	return int64(value)
}

// EvaluateEndpointTemplates parses and evaluates the "ApplicationTemplate" property
// of each of the service endpoints for this service.
func (service *Service) EvaluateEndpointTemplates(gs GetService) (err error) {
	functions := template.FuncMap{
		"parent":       parent(gs),
		"context":      context(),
		"percentScale": percentScale,
		"bytesToMB":    bytesToMB,
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

// Fill in all the templates in the ServiceDefinitions
func (service *Service) Evaluate(getSvc GetService) error {
	if err := service.EvaluateEndpointTemplates(getSvc); err != nil {
		return err
	}
	if err := service.EvaluateLogConfigTemplate(getSvc); err != nil {
		return err
	}
	if err := service.EvaluateConfigFilesTemplate(getSvc); err != nil {
		return err
	}
	if err := service.EvaluateStartupTemplate(getSvc); err != nil {
		return err
	}
	if err := service.EvaluateRunsTemplate(getSvc); err != nil {
		return err
	}
	if err := service.EvaluateActionsTemplate(getSvc); err != nil {
		return err
	}

	return nil
}
