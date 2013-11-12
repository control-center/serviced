/*******************************************************************************
* Copyright (C) Zenoss, Inc. 2013, all rights reserved.
*
* This content is made available according to terms specified in
* License.zenoss under the directory where your Zenoss product is installed.
*
*******************************************************************************/

package main

import (
	"github.com/zenoss/glog"

	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type StatsReporter struct {
	destination string
}

func reportMemStats(path string, info os.FileInfo, err error) error {
	if info.IsDir() == false {
		return nil
	}

	if filepath.Base(path) == "lxc" {
		return nil
	}

	if stats, err := ioutil.ReadFile(strings.Join([]string{path, "memory.stat"}, "/")); err != nil {
		return err
	} else {
		fmt.Println(string(stats))
		return nil
	}
}

func (sr StatsReporter) reportStats(t time.Time) {
	glog.V(3).Infoln("Reporting container stats at: ", t)
	if err := filepath.Walk("/sys/fs/cgroup/memory/lxc", reportMemStats); err != nil {
		glog.Warning("Problem reporting container memory statistics: ", err)
	}
}

func (sr StatsReporter) Report() {
	tc := time.Tick(15 * time.Second)
	for t := range tc {
		go sr.reportStats(t)
	}
}
