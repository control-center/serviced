// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

// Package agent implements a service that runs on a serviced node. It is
// responsible for ensuring that a particular node is running the correct services
// and reporting the state and health of those services back to the master
// serviced.

package main

// This is here the command line arguments are parsed and executed.

import (
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/dao"

	"encoding/json"
	"fmt"
	"os"

	/*
		clientlib "github.com/zenoss/serviced/client"
		"github.com/zenoss/serviced/proxy"
	*/)

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

	var serviceTemplates map[string]*dao.ServiceTemplate
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

// Add a service template to the control plane.
func (cli *ServicedCli) CmdAddTemplate(args ...string) error {

	cmd := Subcmd("add-template", "[INPUT]", "Add a template. Use - for standard input. "+
		"[INPUT] is either a json file or template directory.")
	if err := cmd.Parse(args); err != nil {
		return err
	}
	var serviceTemplate dao.ServiceTemplate
	if len(cmd.Args()) != 1 {
		cmd.Usage()
		return nil
	}

	if cmd.Arg(0) == "-" {
		// Read from standard input
		dec := json.NewDecoder(os.Stdin)
		err := dec.Decode(&serviceTemplate)
		if err != nil {
			glog.Fatalf("Could not read ServiceTemplate from stdin: %s", err)
		}
	} else {
		// is the input a file or directory
		sdefinition, err := dao.ServiceDefinitionFromPath(cmd.Arg(0))
		if err != nil {
			glog.Fatalf("Could not read template from directory %s: %s", cmd.Arg(0), err)
		}
		serviceTemplate.Services = make([]dao.ServiceDefinition, 1)
		serviceTemplate.Services[0] = *sdefinition
		serviceTemplate.Name = sdefinition.Name
	}

	c := getClient()
	var templateId string
	err := c.AddServiceTemplate(serviceTemplate, &templateId)
	if err != nil {
		glog.Fatalf("Could not add service template:  %s", err)
	}
	fmt.Println(templateId)

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
	fmt.Println("OK")

	return nil
}

// Deploy a service template into the given pool
func (cli *ServicedCli) CmdDeployTemplate(args ...string) error {

	cmd := Subcmd("deploy-template", "[OPTIONS] TEMPLATE_ID POOL_ID DEPLOYMENT_ID", "Deploy TEMPLATE_ID into POOL_ID with a new id DEPLOYMENT_ID")
	if err := cmd.Parse(args); err != nil {
		return err
	}

	deployreq := dao.ServiceTemplateDeploymentRequest{cmd.Arg(1), cmd.Arg(0), cmd.Arg(2)}

	var unused int
	if err := getClient().DeployTemplate(deployreq, &unused); err != nil {
		glog.Fatalf("Could not deploy service template: %v", err)
	}
	fmt.Println("OK")

	return nil
}
