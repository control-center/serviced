package main

import (
	"errors"
	"fmt"

	"github.com/zenoss/glog"
)

// CmdAction attaches to service(s) and performs the predefined action
func (cli *ServicedCli) CmdAction(args ...string) error {
	cmd := Subcmd("action", "ACTION", "attach to service(s) and perform the predefined action")

	var pattern string
	cmd.StringVar(&pattern, "regexp", "", "use REGEXP to match service")

	if err := cmd.Parse(args); err != nil {
		return err
	}

	if len(cmd.Args()) < 1 {
		return errors.New(fmt.Sprintf("missing action to perform\n"))
	}

	action := cmd.Args()[0]

	controlPlane := getClient()
	if err := controlPlane.Action(action, pattern); err != nil {
		glog.Errorf("Received an error: %s", err)
		return err
	} else {
		fmt.Printf("Successfully performed action:%s\n", action)
	}

	return nil
}
