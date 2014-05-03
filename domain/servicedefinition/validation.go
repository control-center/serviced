// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package servicedefinition

import (
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/commons"
	"github.com/zenoss/serviced/validation"

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

	//make sure launch attribute is valid and normalize case to the constants
	err := sd.NormalizeLaunch()
	if err != nil {
		return fmt.Errorf("service definition %v: %v", sd.Name, err)
	}

	//validate endpoint config
	names := make(map[string]struct{})
	for _, se := range sd.Endpoints {
		if err = se.ValidEntity(); err != nil {
			return fmt.Errorf("Service Definition %v: %v", sd.Name, err)
		}
		if err = context.validateVHost(se); err != nil {
			return fmt.Errorf("Service Definition %v: %v", sd.Name, err)
		}
		//enable unique endpoint name validation
		trimName := strings.Trim(se.Name, " ")
		if _, found := names[trimName]; found {
			return fmt.Errorf("Service Definition %v: Endpoint name %s not unique in service definition", sd.Name, trimName)
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

//normalizeLaunch validates and normalizes the launch string. Set the normalized value of launch that matches
// constants. If string is not a valid value returns error. Launch must be empty, "auto", or "manual", if it's empty
// default it to "AUTO"
func (sd *ServiceDefinition) NormalizeLaunch() error {
	testStr := sd.Launch
	if testStr == "" {
		testStr = commons.AUTO
	} else {
		testStr = strings.Trim(strings.ToLower(testStr), " ")
		if testStr != commons.AUTO && testStr != commons.MANUAL {
			return fmt.Errorf("invalid launch setting (%s)", sd.Launch)
		}
	}
	sd.Launch = testStr
	return nil
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
				return fmt.Errorf("Duplicate Vhost found: %v", vhost)
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
		return fmt.Errorf("Service Definition %v: Endpoint must have a name %v", se.Name, se)
	}
	if se.Purpose != "import" {
		if err := validation.ValidPort(int(se.PortNumber)); err != nil {
			return fmt.Errorf("Endpoint '%s': %s", se.Name, err)
		}
		if err := applicationValidation(se.Application); err != nil {
			return fmt.Errorf("Endpoint '%s': %s", se.Name, err)
		}
	}
	return se.AddressConfig.ValidEntity()
}

func applicationValidation(application string) error {
	_, err := regexp.Compile(application)
	if err != nil {
		err = fmt.Errorf("Illegal application regexp %s", err)
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
			violations.Add(fmt.Errorf("AddressResourceConfig: %v", err))
		}

		if err := validation.StringIn(arc.Protocol, commons.TCP, commons.UDP); err != nil {
			violations.Add(fmt.Errorf("AddressResourceConfig: invalid protocol: %v", err))
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

