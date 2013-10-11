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

/*
	clientlib "github.com/zenoss/serviced/client"
	"github.com/zenoss/serviced/proxy"
*/
)

// List the service templates associated with the control plane.
func (cli *ServicedCli) CmdTemplates(args ...string) error {
	cmd := Subcmd("templates", "[OPTIONS]", "List templates")
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

	for id, template := range serviceTemplates {
		if t, err := json.MarshalIndent(template, " ", " "); err == nil {
			fmt.Printf("%s: %s\n", id, t)
		}
	}

	return err
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
	c := getClient()
	err = c.AddServiceTemplate(serviceTemplate, &unused)
	if err != nil {
		glog.Fatalf("Could not read add service template:  %s", err)
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

	glog.Infof("Service template %s removed", cmd.Arg(0))
	return nil
}

// Deploy a service template into the given pool
func (cli *ServicedCli) CmdDeployTemplate(args ...string) error {

	cmd := Subcmd("deploy-template", "[OPTIONS] TEMPLATE_ID POOL_ID", "Deploy TEMPLATE_ID into POOL_ID")
	if err := cmd.Parse(args); err != nil {
		return err
	}
	return nil
}
