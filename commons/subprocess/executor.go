// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

//Package subprocess defines an executor service for running parallel subprocesses
package subprocess

import (
	"bytes"
	"os/exec"
	"time"

	"github.com/zenoss/glog"
)

//Result defines a command to run and store's it's execution results
type Result struct {
	Command string       //command to execute in parallel
	Args    []string     //arguments for command
	Stdout  bytes.Buffer //command's captured stdout
	Stderr  bytes.Buffer //command's captured stderr
	Error   error        //command final error status

	complete bool //are we done?
}

//get execute's the command defined in the result struct and stores it's execution results
func (result *Result) get() {
	result.Stdout = bytes.Buffer{}
	result.Stderr = bytes.Buffer{}
	cmd := exec.Command(result.Command, result.Args...)
	cmd.Stdout = &result.Stdout
	cmd.Stderr = &result.Stderr
	result.Error = cmd.Run()
}

//Executor supports running parallel subprocesses
type Executor struct {
	Results []Result // results of running commands
}

//Submit queues a job for execution
func (executor *Executor) Submit(Command string, Args ...string) {
	result := Result{
		Command:  Command,
		Args:     Args,
		complete: false,
	}
	executor.Results = append(executor.Results, result)
}

//Execute spawns all submitted jobs for execution and waits for their completion
func (executor *Executor) Execute() {
	//spawn parallel pull requests
	resultsChannel := make(chan int)
	for i := range executor.Results {
		result := &executor.Results[i]
		glog.Infof("Executing Command: Cmd:%s Args:%s", result.Command, result.Args)
		go func(ch chan int, id int, result *Result) {
			result.get()
			ch <- id
		}(resultsChannel, i, result)
	}

	//wait for completion
	for {
		select {
		case i := <-resultsChannel:
			result := &executor.Results[i]
			if result.Error == nil {
				glog.Infof("Command completed, Cmd:%s Args:%s", result.Command, result.Args)
			} else {
				glog.Errorf("Command completed, Cmd:%s Args:%s", result.Command, result.Args)
				glog.Error(" w/error: ", result.Error)
				glog.Error(" w/stdout: ", result.Stdout.String())
				glog.Error(" w/stderr: ", result.Stderr.String())
			}
			result.complete = true
			//TODO, wait time can be configurable
		case <-time.After(10 * time.Second):
		}

		//print banner for users if there's a pull request pending, otherwise break
		complete := true
		for i := range executor.Results {
			result := &executor.Results[i]
			if result.complete == false {
        glog.Infof("Pending command completion: Cmd:%s Args:%s", result.Command, result.Args)
				complete = false
			}
		}

		if complete {
			break
		}
	}
}
