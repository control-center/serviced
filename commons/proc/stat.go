// Copyright 2014 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package proc

import (
	"github.com/zenoss/glog"

	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

var procDir = "/proc/"

// ProcStat is a parsed represenation of /proc/PID/stat information.
type ProcStat struct {
	Pid      int    // The process ID.
	Filename string // The filename of the executable, in parentheses. This is visible whether or not the executable is swapped out.
	State    string // One character from the string "RSDZTW" where R is running, S is sleeping in an interruptible wait, D is waiting in uninterruptible disk sleep, Z is zombie, T is traced or stopped
	Ppid     int    // The PID of the parent.
	Pgrp     int    // The process group ID of the process.
	Session  int    // The session ID of the process.
}

var MalformedStatErr = fmt.Errorf("malformed stat file")

func parseInt(str string, val *int) error {
	i, err := strconv.ParseInt(str, 0, 64)
	if err != nil {
		return err
	}
	*val = int(i)
	return nil
}

// GetAllPids finds all processes under the /proc directory
func GetAllPids() ([]int, error) {
	finfos, err := ioutil.ReadDir(procDir)
	if err != nil {
		return nil, err
	}
	var pids []int
	for _, finfo := range finfos {
		if !finfo.IsDir() {
			continue
		}
		if pid, err := strconv.ParseInt(finfo.Name(), 0, 32); err == nil {
			pids = append(pids, int(pid))
		}
	}
	return pids, nil
}

// ReapZombies will call wait on zombie pids in order to reap them.
func ReapZombies() {
	pids, err := GetAllPids()
	if err != nil {
		return
	}
	var done sync.WaitGroup
	for _, pid := range pids {
		stat, err := GetProcStat(pid)
		glog.V(8).Infof("found pid %d procstat %s", pid, err)
		if err != nil || stat.State != "Z" {
			continue
		}
		done.Add(1)
		go func(p int) {
			defer done.Done()
			process, err := os.FindProcess(p)
			if err != nil || process == nil {
				return
			}
			process.Wait()
		}(pid)
	}
	done.Wait()
}

// KillGroup will send a SIGTERM to all the processes in the pgrp processs group. If the processes don't
// shutdown within 10 seconds
func KillGroup(pgrp int, timeout time.Duration) error {
	pids, err := GetAllPids()
	if err != nil {
		return err
	}
	timedout := make(chan struct{})
	var done sync.WaitGroup
	for _, pid := range pids {
		stat, err := GetProcStat(pid)
		glog.V(8).Infof("found pid %d procstat %s", pid, err)
		if err != nil || stat.Pgrp != pgrp {
			continue
		}
		done.Add(1)
		go func(p int) {
			defer done.Done()
			process, err := os.FindProcess(p)
			if err != nil {
				return
			}
			if process == nil {
				return
			}
			glog.V(8).Infof("sending %d SIGTERM", process.Pid)
			process.Signal(syscall.SIGTERM)
			exited := make(chan error)
			go func(e chan error) {
				glog.V(8).Infof("process %d exited", process.Pid)
				_, err := process.Wait()
				e <- err
			}(exited)
			select {
			case <-exited:
			case <-timedout:
				glog.V(8).Infof("process %d sigterm timedout, sending SIGKILL", process.Pid)
				process.Signal(syscall.SIGKILL)
			}
		}(pid)
	}
	go func() {
		time.Sleep(timeout)
		close(timedout)
	}()
	done.Wait()
	return nil
}

// GetProcStat gets the /proc/pid/stat info for the given pid.
func GetProcStat(pid int) (*ProcStat, error) {
	stat := ProcStat{}

	// read in the stat
	data, err := ioutil.ReadFile(fmt.Sprintf(procDir+"%d/stat", pid))
	if err != nil {
		return nil, err
	}
	fields := strings.Fields(string(data))

	// the first field contains the PID
	if len(fields) < 1 {
		return nil, fmt.Errorf("less than 1 field")
	}
	if err := parseInt(fields[0], &stat.Pid); err != nil {
		return nil, err
	}

	// The next couple of fields will contain the process file name enclosed by ()
	// so look for the last ) starting at the end
	i := 0
	for j := len(fields) - 1; j >= 0; j-- {
		if strings.HasSuffix(fields[j], ")") {
			i = j
			break
		}
	}
	if i < 1 {
		return nil, fmt.Errorf("could not find process name")
	}
	stat.Filename = fields[1]
	for j := 2; j <= i; j++ {
		stat.Filename = stat.Filename + " " + fields[j]
	}

	if (len(fields) - i) < 5 {
		return nil, MalformedStatErr
	}

	// the next field is the process state flag
	i++
	stat.State = fields[i]

	// the remaining fields are integers
	i++
	if err := parseInt(fields[i], &stat.Ppid); err != nil {
		return nil, MalformedStatErr
	}
	i++
	if err := parseInt(fields[i], &stat.Pgrp); err != nil {
		return nil, MalformedStatErr
	}
	i++
	if err := parseInt(fields[i], &stat.Session); err != nil {
		return nil, MalformedStatErr
	}
	return &stat, nil
}
