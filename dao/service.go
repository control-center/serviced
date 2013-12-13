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

func (service *Service) EvaluateContext(cp ControlPlane) (err error) {
	parent := func(s Service) (value Service, err error) {
		err = cp.GetService(s.ParentServiceId, &value)
		return
	}

	context := func(s Service) (ctx map[string]interface{}, err error) {
		err = json.Unmarshal([]byte(s.Context), &ctx)
		if err != nil {
			glog.Errorf("Error unmarshal service context Id=%s: %s -> %s", s.Id, s.Context, err)
		}
		return
	}

	functions := template.FuncMap{
		"parent":  parent,
		"context": context,
	}

	t := template.Must(template.New(service.Name).Funcs(functions).Parse(service.Startup))

	var buffer bytes.Buffer
	if err = t.Execute(&buffer, service); err == nil {
		service.Startup = buffer.String()
	}
	return
}
