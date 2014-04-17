package main

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/dao"
)

// runServiceCommand attaches to a service state container and executes an arbitrary bash command
func attachContainerAndRun(state *dao.ServiceState, command string) ([]byte, error) {
	if state.DockerId == "" {
		return []byte{}, fmt.Errorf("the DockerId is empty for state:%+v", state)
	}

	exeMap, err := exePaths([]string{"sudo", "nsinit"})
	if err != nil {
		return []byte{}, err
	}

	nsInitRoot := "/var/lib/docker/execdriver/native" // has container.json

	attachCmd := fmt.Sprintf("cd %s/%s && %s exec %s", nsInitRoot, state.DockerId,
		exeMap["nsinit"], command)
	fullCmd := []string{exeMap["sudo"], "--", "/bin/bash", "-c", attachCmd}
	glog.V(2).Infof("ServiceId: %s, Command: %s", state.ServiceId, strings.Join(fullCmd, " "))
	cmd := exec.Command(fullCmd[0], fullCmd[1:]...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		glog.Errorf("Error running command: '%s' for serviceId.%d:%s output: %s err: %s", command, state.InstanceId, state.ServiceId, output, err)
		return output, err
	}
	glog.V(1).Infof("Successfully ran command: '%s' for serviceId.%d:%s  output: %s", command, state.InstanceId, state.ServiceId, output)
	return output, nil
}

// CmdAction attaches to service(s) and performs the predefined action
func (cli *ServicedCli) CmdAction(args ...string) error {
	// Check the args
	cmd := Subcmd("action", "SERVICEID ACTION", "attach to service instances and perform the predefined action")

	var instance = -1
	cmd.IntVar(&instance, "instance", instance, "instance specifier, default -1=ALL")

	if err := cmd.Parse(args); err != nil {
		return err
	}

	if len(cmd.Args()) < 2 {
		cmd.Usage()
		return fmt.Errorf("missing action to perform\n")
	}

	serviceID := cmd.Arg(0)
	action := cmd.Arg(1)

	// Get the associated service
	cp := getClient()
	service, err := getService(&cp, serviceID)
	if err != nil {
		glog.Fatalf("error while acquiring service - error: %s", err)
	} else if service == nil {
		glog.Fatalf("no service found for serviceID: %s", serviceID)
	}

	// Evaluate service Actions for templates
	if err := service.EvaluateActionsTemplate(cp); err != nil {
		glog.Fatalf("error evaluating service:%s Actions:%+v  error:%s", service.Id, service.Actions, err)
	}

	// Parse the command
	command, ok := service.Actions[action]
	if !ok {
		glog.Infof("service: %+v", service)
		glog.Fatalf("cannot access action: %s", action)
	}

	// Perform the action on all service states of this service
	var states []*dao.ServiceState
	if err := cp.GetServiceStates(service.Id, &states); err != nil {
		glog.Fatalf("unable to retrieve service states for serviceID:%s error:%s", service.Id, err)
	}

	if len(states) < 1 {
		glog.Fatalf("unable to find service states for serviceID:%s", service.Id)
	}

	for _, state := range states {
		if instance >= 0 {
			if state.InstanceId != instance {
				continue
			}
		}

		output, err := attachContainerAndRun(state, command)
		if err != nil {
			return err
		}
		fmt.Printf("%s", string(output))
	}

	// TODO: ask control plane to perform action
	//if err := cp.Action(action, serviceID); err != nil {
	//	glog.Errorf("Received an error: %s", err)
	//	return err
	//} else {
	//	fmt.Printf("Successfully performed action:%s\n", action)
	//}

	return nil
}
