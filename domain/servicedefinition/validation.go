// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package servicedefinition

import (
	"github.com/zenoss/glog"
	"github.com/control-center/serviced/commons"
	"github.com/control-center/serviced/validation"

	"fmt"
	"regexp"
	"strings"
)

//ValidEntity validates Host fields
func (sd *ServiceDefinition) ValidEntity() error {
	glog.V(4).Info("Validating ServiceDefinition")

	context := validationContext{make(map[string]EndpointDefinition)}
	//TODO: do servicedefinition names need to be unique?
	err := sd.validate(&context)
	return err
}

//validate ServiceDefinition configuration and any embedded ServiceDefinitions
func (sd *ServiceDefinition) validate(context *validationContext) error {
	//TODO: check name, description, config files.

	//Check MinMax settings
	if err := sd.Instances.Validate(); err != nil {
		return fmt.Errorf("service Definition %v: %v", sd.Name, err)
	}

	if err := validation.StringIn(sd.Launch, commons.AUTO, commons.MANUAL); err != nil {
		return fmt.Errorf("service definition %v: invalid launch setting %v", sd.Name, err)
	}

	//validate endpoint config
	names := make(map[string]struct{})
	for _, se := range sd.Endpoints {
		if err := se.ValidEntity(); err != nil {
			return fmt.Errorf("service definition %v: %v", sd.Name, err)
		}
		if err := context.validateVHost(se); err != nil {
			return fmt.Errorf("service definition %v: %v", sd.Name, err)
		}
		//enable unique endpoint name validation
		trimName := strings.Trim(se.Name, " ")
		if _, found := names[trimName]; found {
			return fmt.Errorf("service definition %v: Endpoint name %s not unique in service definition", sd.Name, trimName)
		}
		names[trimName] = struct{}{}
	}
	//TODO: validate LogConfigs

	return validServiceDefinitions(&sd.Services, context)
}

// validServiceDefinitions validates an array of ServiceDefinition recursively
func validServiceDefinitions(ds *[]ServiceDefinition, context *validationContext) error {
	for _, sd := range *ds {
		if err := sd.validate(context); err != nil {
			return err
		}
	}
	return nil
}

//NormalizeLaunch normalizes the launch string. Sets to commons.AUTO if empty otherwise just trims and lower cases. Does
//not check if value is valid
func (sd *ServiceDefinition) NormalizeLaunch() {
	testStr := strings.Trim(strings.ToLower(sd.Launch), " ")
	if testStr == "" {
		testStr = commons.AUTO
	}
	sd.Launch = testStr
}

//validationContext is used to keep track of things to validate in nested service definitions
type validationContext struct {
	vhosts map[string]EndpointDefinition // only care about key to test for previous definition
}

//validateVHost ensures that the VHosts in a ServiceEndpoint have not already been defined
func (vc validationContext) validateVHost(se EndpointDefinition) error {
	if len(se.VHosts) > 0 {
		for _, vhost := range se.VHosts {
			if _, found := vc.vhosts[vhost]; found {
				return fmt.Errorf("duplicate Vhost found: %v", vhost)
			}
			vc.vhosts[vhost] = se
		}
	}
	return nil
}

//ValidEntity used to make sure ServiceEndpoint is in a valid state
func (se EndpointDefinition) ValidEntity() error {
	trimName := strings.Trim(se.Name, " ")
	if trimName == "" {
		return fmt.Errorf("service definition %v: endpoint must have a name %v", se.Name, se)
	}
	if se.Purpose != "import" {
		if err := validation.ValidPort(int(se.PortNumber)); err != nil {
			return fmt.Errorf("endpoint '%s': %s", se.Name, err)
		}
		if err := applicationValidation(se.Application); err != nil {
			return fmt.Errorf("endpoint '%s': %s", se.Name, err)
		}
	}
	return se.AddressConfig.ValidEntity()
}

func applicationValidation(application string) error {
	_, err := regexp.Compile(application)
	if err != nil {
		err = fmt.Errorf("illegal application regexp %s", err)
	}
	return err
}

//ValidEntity used to make sure AddressResourceConfig is in a valid state
func (arc AddressResourceConfig) ValidEntity() error {
	//check if protocol set or port not 0
	violations := validation.NewValidationError()
	if arc.Protocol != "" || arc.Port > 0 {
		//some setting, now lets make sure they are valid
		if err := validation.ValidPort(int(arc.Port)); err != nil {
			violations.Add(fmt.Errorf("error AddressResourceConfig: %v", err))
		}

		if err := validation.StringIn(arc.Protocol, commons.TCP, commons.UDP); err != nil {
			violations.Add(fmt.Errorf("error AddressResourceConfig: invalid protocol: %v", err))
		}

	}
	if violations.HasError() {
		return violations
	}
	return nil
}

//Normalize adjusts attributes to be in an expected format, lowercases and trims certain fields
func (arc *AddressResourceConfig) Normalize() {
	testProto := strings.Trim(strings.ToLower(arc.Protocol), " ")
	arc.Protocol = testProto
}
