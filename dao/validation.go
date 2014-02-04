/*******************************************************************************
* Copyright (C) Zenoss, Inc. 2014, all rights reserved.
*
* This content is made available according to terms specified in
* License.zenoss under the directory where your Zenoss product is installed.
*
*******************************************************************************/

package dao

import (
	"fmt"
	"strings"
	"github.com/zenoss/serviced/commons"
)

// Validate ensure that a ServiceTemplate has valid values
func (t *ServiceTemplate) Validate() error {
	//TODO check name, description, config files.
	context := validationContext{make(map[string]ServiceEndpoint)}
	//TODO do servicedefinition names need to be unique?
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
	for _, se := range d.Endpoints {
		err := se.validate()
		if err == nil {
			err = context.validateVHost(se)
		}
		if err != nil {
			return fmt.Errorf("Service Definition %v: %v", d.Name, err)
		}

	}
	//TODO validate LogConfigs

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
	//TODO validate more ServiceEndpoint fields
	return se.AddressConfig.normalize()
}

func (arc AddressResourceConfig) normalize() error {
	//check if protocol set or port not 0
	if arc.Protocol != "" || arc.Port > 0 || arc.Port < 0 {
		//some setting, now lets make sure they are valid
		if arc.Port > 65535 || arc.Port < 1 {
			return fmt.Errorf("AddressResourceConfig: Invalid port number %v", arc.Port)
		}
		testProto := strings.Trim(strings.ToLower(arc.Protocol), " ")
		if testProto != commons.TCP && testProto != commons.UDP{
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
