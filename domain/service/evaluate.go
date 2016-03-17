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

package service

import (
	"github.com/zenoss/glog"

	"bytes"
	"runtime"
	"strings"
	"text/template"
)

func parent(gs GetService) func(s *runtimeContext) (*runtimeContext, error) {
	rc := &runtimeContext{}
	return func(svc *runtimeContext) (*runtimeContext, error) {
		s, err := gs(svc.ParentServiceID)
		if err != nil {
			return rc, err
		}
		return &runtimeContext{s, 0}, nil
	}
}

func child(fc FindChildService) func(s *runtimeContext, childName string) (*runtimeContext, error) {
	rc := &runtimeContext{}
	return func(svc *runtimeContext, childName string) (*runtimeContext, error) {
		s, err := fc(svc.ID, childName)
		if err != nil {
			return rc, err
		}
		return &runtimeContext{s, 0}, nil
	}
}

func flattenContext(svc Service, gs GetService, prefix string, ctx *map[string]interface{}) error {
	if svc.ParentServiceID != "" {
		parent, err := gs(svc.ParentServiceID)
		if err != nil {
			return err
		}
		err = flattenContext(parent, gs, prefix, ctx)
		if err != nil {
			return err
		}
	}
	if svc.Context != nil {
		for k, v := range svc.Context {
			if strings.HasPrefix(k, prefix) {
				(*ctx)[strings.TrimPrefix(k, prefix)] = v
			}
		}
	}
	return nil
}

func getContext(gs GetService) func(s *runtimeContext, key string) (interface{}, error) {
	return func(s *runtimeContext, key string) (interface{}, error) {
		ctx := make(map[string]interface{})
		err := flattenContext(s.Service, gs, "", &ctx)
		if err != nil {
			glog.Errorf("Flattening context for %s (%s): %s", s.Name, s.ID, err)
		}

		return ctx[key], err
	}
}

func context(gs GetService) func(s *runtimeContext) (map[string]interface{}, error) {
	return func(s *runtimeContext) (map[string]interface{}, error) {
		ctx := make(map[string]interface{})
		err := flattenContext(s.Service, gs, "", &ctx)
		if err != nil {
			glog.Errorf("Flattening context for %s (%s): %s", s.Name, s.ID, err)
		}
		return ctx, err
	}
}

func contextFilter(gs GetService) func(s *runtimeContext, prefix string) (map[string]interface{}, error) {
	return func(s *runtimeContext, prefix string) (map[string]interface{}, error) {
		ctx := make(map[string]interface{})
		err := flattenContext(s.Service, gs, prefix, &ctx)
		if err != nil {
			glog.Errorf("Flattening context for %s (%s prefix:%s): %s", s.Name, s.ID, prefix, err)
		}
		return ctx, err
	}
}

// EvaluateActionsTemplate parses and evaluates the Actions string of a service.
func (service *Service) EvaluateActionsTemplate(gs GetService, fc FindChildService, instanceID int) (err error) {
	for key, value := range service.Actions {
		err, result := service.evaluateTemplate(gs, fc, instanceID, value)
		if err != nil {
			return err
		}

		if result != "" {
			service.Actions[key] = result
		}
	}
	return
}

// EvaluateHostnameTemplate parses and evaluates the Hostname string of a service.
func (service *Service) EvaluateHostnameTemplate(gs GetService, fc FindChildService, instanceID int) (err error) {
	err, result := service.evaluateTemplate(gs, fc, instanceID, service.Hostname)
	if err == nil {
		service.Hostname = result
	}
	return
}

// EvaluateVolumesTemplate parses and evaluates the ResourcePath string in
// volumes of a service
func (service *Service) EvaluateVolumesTemplate(gs GetService, fc FindChildService, instanceID int) (err error) {
	for i, vol := range service.Volumes {
		err, result := service.evaluateTemplate(gs, fc, instanceID, vol.ResourcePath)
		if err != nil {
			return err
		}

		if result != "" {
			vol.ResourcePath = result
		}
		service.Volumes[i] = vol
	}
	return
}

// EvaluateStartupTemplate parses and evaluates the StartUp string of a service.
func (service *Service) EvaluateStartupTemplate(gs GetService, fc FindChildService, instanceID int) (err error) {
	err, result := service.evaluateTemplate(gs, fc, instanceID, service.Startup)
	if err == nil && result != "" {
		service.Startup = result
	}
	return
}

// EvaluateRunsTemplate parses and evaluates the Runs string of a service.
func (service *Service) EvaluateRunsTemplate(gs GetService, fc FindChildService) (err error) {
	for key, value := range service.Runs {
		err, result := service.evaluateTemplate(gs, fc, 0, value)
		if err != nil {
			return err
		}
		if result != "" {
			service.Runs[key] = result
		}
	}
	for key, value := range service.Commands {
		err, result := service.evaluateTemplate(gs, fc, 0, value.Command)
		if err != nil {
			return err
		}
		if result != "" {
			value.Command = result
			service.Commands[key] = value
		}
	}
	return
}

// evaluateTemplate takes a control center client and template string and evaluates
// the template using the service as the context. If the template is invalid or there is an error
// then an empty string is returned.
func (service *Service) evaluateTemplate(gs GetService, fc FindChildService, instanceID int, serviceTemplate string) (err error, result string) {
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(runtime.Error); ok {
				panic(r)
			}
			err = r.(error)
			result = ""
		}
	}()

	functions := template.FuncMap{
		"parent":        parent(gs),
		"child":         child(fc),
		"context":       context(gs),
		"getContext":    getContext(gs),
		"contextFilter": contextFilter(gs),
		"percentScale":  percentScale,
		"bytesToMB":     bytesToMB,
		"plus":          plus,
		"uintToInt":     uintToInt,
		"each":          each,
	}

	// parse the template
	t := template.Must(template.New("ServiceDefinitionTemplate").Funcs(functions).Parse(serviceTemplate))

	// evaluate it
	var buffer bytes.Buffer
	ctx := newRuntimeContext(service, instanceID)
	err = t.Execute(&buffer, ctx)
	if err == nil {
		result = buffer.String()
		return
	}

	// something went wrong, warn them
	glog.Warningf("Evaluating template %v produced the following error %v ", serviceTemplate, err)
	return
}

// EvaluateLogConfigTemplate parses and evals the Path, Type and all the values for the tags of the log
// configs. This happens for each LogConfig on the service.
func (service *Service) EvaluateLogConfigTemplate(gs GetService, fc FindChildService, instanceID int) (err error) {
	// evaluate the template for the LogConfig as well as the tags
	for i, logConfig := range service.LogConfigs {
		// Path
		err, result := service.evaluateTemplate(gs, fc, instanceID, logConfig.Path)
		if err != nil {
			return err
		}
		if result != "" {
			service.LogConfigs[i].Path = result
		}

		// Type
		err, result = service.evaluateTemplate(gs, fc, instanceID, logConfig.Type)
		if err != nil {
			return err
		}
		if result != "" {
			service.LogConfigs[i].Type = result
		}

		// Tags
		for j, tag := range logConfig.LogTags {
			err, result = service.evaluateTemplate(gs, fc, instanceID, tag.Value)
			if err != nil {
				return err
			}
			if result != "" {
				service.LogConfigs[i].LogTags[j].Value = result
			}
		}
	}
	return
}

// EvaluateConfigFilesTemplate parses and evals the Filename and Content. This happens for each
// ConfigFile on the service.
func (service *Service) EvaluateConfigFilesTemplate(gs GetService, fc FindChildService, instanceID int) (err error) {
	glog.V(3).Infof("Evaluating Config Files for %s:%d", service.ID, instanceID)
	for key, configFile := range service.ConfigFiles {
		// Filename
		err, result := service.evaluateTemplate(gs, fc, instanceID, configFile.Filename)
		if err != nil {
			return err
		}
		if result != "" {
			configFile.Filename = result
		}
		// Content
		err, result = service.evaluateTemplate(gs, fc, instanceID, configFile.Content)
		if err != nil {
			return err
		}
		if result != "" {
			configFile.Content = result
		}
		service.ConfigFiles[key] = configFile
	}
	return
}

// EvaluatePrereqsTemplate parses and evals the Script field for each Prereq.
func (service *Service) EvaluatePrereqsTemplate(gs GetService, fc FindChildService, instanceID int) (err error) {
	glog.V(3).Infof("Evaluating Prereq scripts for %s:%d", service.ID, instanceID)
	for i, prereq := range service.Prereqs {
		err, result := service.evaluateTemplate(gs, fc, instanceID, prereq.Script)
		if err != nil {
			return err
		}
		if result != "" {
			prereq.Script = result
			service.Prereqs[i] = prereq
		}
	}
	return
}

// EvaluateHealthCheckTemplate parses and evals the Script field for each HealthCheck.
func (service *Service) EvaluateHealthCheckTemplate(gs GetService, fc FindChildService, instanceID int) (err error) {
	glog.V(3).Infof("Evaluating HealthCheck scripts for %s:%d", service.ID, instanceID)
	for key, healthcheck := range service.HealthChecks {
		err, result := service.evaluateTemplate(gs, fc, instanceID, healthcheck.Script)
		if err != nil {
			return err
		}
		if result != "" {
			healthcheck.Script = result
			service.HealthChecks[key] = healthcheck
		}
	}
	return
}

func percentScale(x uint64, percentage float64) uint64 {
	return uint64(round(float64(x) * percentage))
}

func bytesToMB(x uint64) uint64 {
	return uint64(round(float64(x) / (1048576.0))) // 1024.0 * 1024
}

func each(n int) []int {
	r := make([]int, n)
	for i := 0; i < n; i++ {
		r[i] = i
	}
	return r
}

func plus(a, b int) int {
	return a + b
}

func uintToInt(a uint64) int {
    return int(a)    
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
func (service *Service) EvaluateEndpointTemplates(gs GetService, fc FindChildService) (err error) {
	for i, ep := range service.Endpoints {
		//cache the application template (assumes this is called after service creation from svc definition)
		if ep.Application != "" && ep.ApplicationTemplate == "" {
			ep.ApplicationTemplate = ep.Application
			service.Endpoints[i].ApplicationTemplate = ep.Application
		}

		if ep.ApplicationTemplate != "" {
			err, result := service.evaluateTemplate(gs, fc, 0, ep.ApplicationTemplate)
			if err != nil {
				return err
			}
			if result != "" {
				service.Endpoints[i].Application = result
			}
		}
	}
	return
}

// runtimeContext wraps a service and adds extra fields for template evaluation.
type runtimeContext struct {
	Service
	InstanceID int
}

// Allow templates to access Service.RAMCommitment.Value via {{.RAMCommitment}}
func (service *runtimeContext) RAMCommitment() uint64 {
	return service.Service.RAMCommitment.Value
}

// newRuntimeContext wraps a given Service with a runtimeContext, adding any
// extra attributes passed in.
func newRuntimeContext(svc *Service, instanceID int) *runtimeContext {
	return &runtimeContext{
		*svc,
		instanceID,
	}
}

// Evaluate evaluates all the fields of the Service that we care about, using
// a runtimeContext with the current Service embedded, and adding instanceID
// as an extra attribute.
func (service *Service) Evaluate(getSvc GetService, findChild FindChildService, instanceID int) (err error) {
	if err = service.EvaluateEndpointTemplates(getSvc, findChild); err != nil {
		glog.Errorf("%+v", err)
		return err
	}
	if err = service.EvaluateLogConfigTemplate(getSvc, findChild, instanceID); err != nil {
		glog.Errorf("%+v", err)
		return err
	}
	if err = service.EvaluateConfigFilesTemplate(getSvc, findChild, instanceID); err != nil {
		glog.Errorf("%+v", err)
		return err
	}
	if err = service.EvaluateStartupTemplate(getSvc, findChild, instanceID); err != nil {
		glog.Errorf("%+v", err)
		return err
	}
	if err = service.EvaluateRunsTemplate(getSvc, findChild); err != nil {
		glog.Errorf("%+v", err)
		return err
	}
	if err = service.EvaluateActionsTemplate(getSvc, findChild, instanceID); err != nil {
		glog.Errorf("%+v", err)
		return err
	}
	if err = service.EvaluateHostnameTemplate(getSvc, findChild, instanceID); err != nil {
		glog.Errorf("%+v", err)
		return err
	}
	if err = service.EvaluateVolumesTemplate(getSvc, findChild, instanceID); err != nil {
		glog.Errorf("%+v", err)
		return err
	}
	if err = service.EvaluatePrereqsTemplate(getSvc, findChild, instanceID); err != nil {
		glog.Errorf("%+v", err)
		return err
	}
	if err = service.EvaluateHealthCheckTemplate(getSvc, findChild, instanceID); err != nil {
		glog.Errorf("%+v", err)
		return err
	}

	return nil
}
