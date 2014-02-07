/*******************************************************************************
* Copyright (C) Zenoss, Inc. 2014, all rights reserved.
*
* This content is made available according to terms specified in
* License.zenoss under the directory where your Zenoss product is installed.
*
*******************************************************************************/

package dao

import (
	"github.com/zenoss/serviced/commons"

	"fmt"
	"strings"
)

// Validate ensure that a ServiceTemplate has valid values
func (t *ServiceTemplate) Validate() error {
	//TODO: check name, description, config files.
	context := validationContext{make(map[string]ServiceEndpoint)}
	//TODO: do servicedefinition names need to be unique?
	err := validServiceDefinitions(&t.Services, &context)
	return err
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

//validate ServiceDefinition configuration and any embedded ServiceDefinitions
func (d *ServiceDefinition) validate(context *validationContext) error {

	//Check MinMax settings
	if err := d.Instances.validate(); err != nil {
		return fmt.Errorf("Service Definition %v: %v", d.Name, err)
	}

	//make sure launch attribute is valid and normalize case to the constants
	err := d.normalizeLaunch()
	if err != nil {
		return fmt.Errorf("Service Definition %v: %v", d.Name, err)
	}

	//validate endpoint config
	//	names := make(map[string]interface{})
	for _, se := range d.Endpoints {
		err = se.validate()
		if err == nil {
			err = context.validateVHost(se)
		}
		if err != nil {
			return fmt.Errorf("Service Definition %v: %v", d.Name, err)
		}
		// TODO: uncomment to unique enable endpoint name validation
		/*
			trimName := strings.Trim(se.Name, "")
			if _, found := names[trimName]; found {
				return fmt.Errorf("Service Definition %v: Endpoint name not unique in service definition %v", d.Name, trimName)
			}
			names[trimName] = struct{}
		*/
	}
	//TODO: validate LogConfigs

	return validServiceDefinitions(&d.Services, context)
}

//normalizeLaunch validates and normalizes the launch string. Set the normalized value of launch that matches
// constants. If string is not a valid value returns error. Launch must be empty, "auto", or "manual", if it's empty
// default it to "AUTO"
func (sd *ServiceDefinition) normalizeLaunch() error {
	testStr := sd.Launch
	if testStr == "" {
		testStr = commons.AUTO
	} else {
		testStr = strings.Trim(strings.ToLower(testStr), " ")
		if testStr != commons.AUTO && testStr != commons.MANUAL {
			return fmt.Errorf("Invalid launch setting (%s)", sd.Launch)
		}
	}
	sd.Launch = testStr
	return nil
}

func (se ServiceEndpoint) validate() error {
	err := portValidation(se.PortNumber)
	if err != nil {
		trimName := strings.Trim(se.Name, "")
		if "" == trimName {
			return fmt.Errorf("Service Definition %v: Endpoint must have a name %v", se.Name, se)
		}
		err = se.AddressConfig.normalize()
	}
	return err
}

func portValidation(port uint16) error {
	if port == 0 {
		return fmt.Errorf("Invalid port number %v", port)
	}
	return nil
}

func (arc AddressResourceConfig) normalize() error {
	//check if protocol set or port not 0
	if arc.Protocol != "" || arc.Port > 0 {
		//some setting, now lets make sure they are valid
		if err := portValidation(arc.Port); err != nil {
			return fmt.Errorf("AddressResourceConfig: %v", err)
		}
		testProto := strings.Trim(strings.ToLower(arc.Protocol), " ")
		if testProto != commons.TCP && testProto != commons.UDP {
			return fmt.Errorf("AddressResourceConfig: Protocol must be one of %v or %v; found %v", commons.TCP, commons.UDP, testProto)
		}
		//use the sanitized version
		arc.Protocol = testProto
	}
	return nil
}

//validate ensure that the values in min max are valid. Max >= Min >=0  returns error otherwise
func (minmax *MinMax) validate() error {
	// Instances["min"] and Instances["max"] must be positive
	if minmax.Min < 0 || minmax.Max < 0 {
		return fmt.Errorf("Instances constraints must be positive: Min=%v; Max=%v", minmax.Min, minmax.Max)
	}

	// If "min" and "max" are both declared Instances["min"] < Instances["max"]
	if minmax.Max != 0 && minmax.Min > minmax.Max {
		return fmt.Errorf("Minimum instances larger than maximum instances: Min=%v; Max=%v", minmax.Min, minmax.Max)
	}
	return nil
}

//valicationContext is used to keep track of things to validate in nested service definitions
type validationContext struct {
	vhosts map[string]ServiceEndpoint // only care about key to test for previous definition
}

//validateVHost ensures that the VHosts in a ServiceEndpoint have not already been defined
func (vc validationContext) validateVHost(se ServiceEndpoint) error {
	if len(se.VHosts) > 0 {
		for _, vhost := range se.VHosts {
			if _, found := vc.vhosts[vhost]; found {
				return fmt.Errorf("Duplicate Vhost found: %v", vhost)
			} else {
				vc.vhosts[vhost] = se
			}
		}
	}
	return nil
}

func (a *AddressAssignment) Validate() error {
	return nil
}
