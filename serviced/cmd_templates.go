/*******************************************************************************
* Copyright (C) Zenoss, Inc. 2013, all rights reserved.
*
* This content is made available according to terms specified in
* License.zenoss under the directory where your Zenoss product is installed.
*
*******************************************************************************/

package main

// This is here the command line arguments are parsed and executed.

import (
	"encoding/json"
	"fmt"
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced"
	"os"
	"strings"

/*
	clientlib "github.com/zenoss/serviced/client"
	"github.com/zenoss/serviced/proxy"
*/
)

// List the service templates associated with the control plane.
func (cli *ServicedCli) CmdTemplates(args ...string) error {
	cmd := Subcmd("templates", "[OPTIONS]", "List templates")

	var verbose bool
	cmd.BoolVar(&verbose, "verbose", false, "Show JSON representation for each template")

	var raw bool
	cmd.BoolVar(&raw, "raw", false, "Don't show header line")

	if err := cmd.Parse(args); err != nil {
		return err
	}

	c := getClient()

	var serviceTemplates map[string]*serviced.ServiceTemplate
	var unused int
	err := c.GetServiceTemplates(unused, &serviceTemplates)
	if err != nil {
		glog.Fatalf("Could not get list of templates: %s", err)
	}

	if verbose == false {
		outfmt := "%-36s %-16s %-32.32s\n"

		if raw == false {
			fmt.Printf("%-36s %-16s %-32s\n", "TEMPLATE ID", "NAME", "DESCRIPTION")
		} else {
			outfmt = "%s|%s|%s\n"
		}

		for id, t := range serviceTemplates {
			fmt.Printf(outfmt, id, t.Name, t.Description)
		}
	} else {
		for id, template := range serviceTemplates {
			if t, err := json.MarshalIndent(template, " ", " "); err == nil {
				if verbose {
					fmt.Printf("%s: %s\n", id, t)
				}
			}
		}
	}

	return err
}

func validServiceDefinition(d *serviced.ServiceDefinition) error {
	// Instances["min"] and Instances["max"] must be positive
	if d.Instances.Min < 0 || d.Instances.Max < 0 {
		return fmt.Errorf("Instances constrains must be positive")
	}

	// If "min" and "max" are both declared Instances["min"] < Instances["max"]
	if d.Instances.Max != 0 && d.Instances.Min > d.Instances.Max {
		return fmt.Errorf("Minimum instances larger than maximum instances")
	}

	// Launch must be empty, "auto", or "manual", if it's empty default it to "AUTO"
	if d.Launch == "" {
		d.Launch = serviced.AUTO
	} else {
		launch := strings.Trim(strings.ToLower(d.Launch), " ")
		if launch != serviced.AUTO && launch != serviced.MANUAL {
			return fmt.Errorf("Invalid launch setting (%s)", d.Launch)
		} else {
			// trim and lower the value of Launch
			d.Launch = launch
		}
	}

	return validServiceDefinitions(&d.Services)
}

func validServiceDefinitions(ds *[]serviced.ServiceDefinition) error {
	for i, _ := range *ds {
		if err := validServiceDefinition(&(*ds)[i]); err != nil {
			return err
		}
	}

	return nil
}

func validTemplate(t *serviced.ServiceTemplate) error {
	return validServiceDefinitions(&t.Services)
}

// Add a service template to the control plane.
func (cli *ServicedCli) CmdAddTemplate(args ...string) error {

	cmd := Subcmd("add-template", "[OPTIONS]", "Add a template")
	if err := cmd.Parse(args); err != nil {
		return err
	}
	var serviceTemplate serviced.ServiceTemplate
	var unused int

	dec := json.NewDecoder(os.Stdin)

	err := dec.Decode(&serviceTemplate)
	if err != nil {
		glog.Fatalf("Could not read ServiceTemplate from stdin: %s", err)
	}

	if err := validTemplate(&serviceTemplate); err != nil {
		return err
	} else {
		c := getClient()
		err = c.AddServiceTemplate(serviceTemplate, &unused)
		if err != nil {
			glog.Fatalf("Could not read add service template:  %s", err)
		}
	}

	return nil
}

// Remove a service template associated with the control plane.
func (cli *ServicedCli) CmdRemoveTemplate(args ...string) error {

	cmd := Subcmd("remove-template", "[OPTIONS]", "Remove a service template")
	if err := cmd.Parse(args); err != nil {
		return err
	}

	if len(cmd.Args()) != 1 {
		cmd.Usage()
		return nil
	}

	var unused int
	if err := getClient().RemoveServiceTemplate(cmd.Arg(0), &unused); err != nil {
		glog.Fatalf("Could not remove service template: %v", err)
	}

	return nil
}

// Deploy a service template into the given pool
func (cli *ServicedCli) CmdDeployTemplate(args ...string) error {

	cmd := Subcmd("deploy-template", "[OPTIONS] TEMPLATE_ID POOL_ID", "Deploy TEMPLATE_ID into POOL_ID")
	if err := cmd.Parse(args); err != nil {
		return err
	}

	deployreq := serviced.ServiceTemplateDeploymentRequest{cmd.Arg(1), cmd.Arg(0)}

	var unused int
	if err := getClient().DeployTemplate(deployreq, &unused); err != nil {
		glog.Fatalf("Could not deploy service template: %v", err)
	}

	return nil
}
