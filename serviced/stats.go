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

	"bufio"
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	BLKIODIR = "/sys/fs/cgroup/blkio/lxc"
	CPUDIR   = "/sys/fs/cgroup/cpuacct/lxc"
	MEMDIR   = "/sys/fs/cgroup/memory/lxc"
)

// StatsReporter is a mechanism for gathering container statistics and sending
// them to a specified destination
type StatsReporter struct {
	destination string
	username    string
	passwd      string
}

// Gather and report available container statistics every duration (d) ticks
func (sr StatsReporter) Report(d time.Duration) {
	tc := time.Tick(d)
	for t := range tc {
		go sr.reportStats(t)
	}
}

// Do the actual gathering and reporting of container statistics
func (sr StatsReporter) reportStats(t time.Time) {
	glog.V(3).Infoln("Reporting container stats at: ", t)

	memfiles := []string{"memory.stat"}
	if err := filepath.Walk(MEMDIR, sr.mkReporter("memory", memfiles, t.Unix())); err != nil {
		glog.Error("Problem reporting container memory statistics: ", err)
	}

	cpufiles := []string{"cpuacct.stat"}
	if err := filepath.Walk(CPUDIR, sr.mkReporter("cpuacct", cpufiles, t.Unix())); err != nil {
		glog.Error("Problem reporting container cpu statistics: ", err)
	}

	blkiofiles := []string{"blkio.sectors", "blkio.io_service_bytes", "blkio.io_serviced", "blkio.io_queued"}
	if err := filepath.Walk(BLKIODIR, sr.mkReporter("blkio", blkiofiles, t.Unix())); err != nil {
		glog.Error("Problem reporting container blkio statistics: ", err)
	}
}

// Create a function to gather and report the container statistics
func (sr StatsReporter) mkReporter(source string, statfiles []string, ts int64) filepath.WalkFunc {
	return func(path string, info os.FileInfo, err error) error {
		if info.IsDir() == false {
			return nil
		}

		dirname := filepath.Base(path)

		if dirname == "lxc" {
			return nil
		}

		for _, statfile := range statfiles {
			if stats, err := ioutil.ReadFile(strings.Join([]string{path, statfile}, "/")); err != nil {
				return err
			} else {
				payload := []containerStat{}
				statscanner := bufio.NewScanner(strings.NewReader(string(stats)))
				for statscanner.Scan() {
					cs := mkContainerStat(dirname, source, ts, statscanner.Text())
					payload = append(payload, cs)
				}

				if len(payload) > 0 {
					data, err := json.Marshal(payload)
					if err != nil {
						glog.V(3).Info("Couldn't marshal stats: ", err)
						return err
					}

					return sr.postStats(data)
				}
				return nil
			}
		}

		return nil
	}
}

func (sr StatsReporter) postStats(stats []byte) error {
	statsreq, err := http.NewRequest("POST", sr.destination, bytes.NewBuffer(stats))
	if err != nil {
		glog.V(3).Info("Couldn't create stats request: ", err)
		return err
	}
	statsreq.SetBasicAuth(sr.username, sr.passwd)
	statsreq.Header["User-Agent"] = []string{"Zenoss Metric Publisher"}
	statsreq.Header["Content-Type"] = []string{"application/json"}

	if glog.V(4) {
		buf := bytes.NewBuffer([]byte{})
		if logerr := statsreq.Write(buf); logerr == nil {
			glog.Info(buf.String())
		}
	}

	resp, reqerr := http.DefaultClient.Do(statsreq)
	if reqerr != nil {
		glog.V(3).Info("Couldn't post stats: ", reqerr)
		return reqerr
	}
	resp.Body.Close()

	return nil
}

type containerStat struct {
	Metric, Value string
	Timestamp     int64
	Tags          map[string]string
}

// Package up container statistics
func mkContainerStat(id, datasource string, timestamp int64, statline string) containerStat {
	statparts := strings.Split(statline, " ")

	tagmap := make(map[string]string)
	tagmap["datasource"] = datasource
	tagmap["uuid"] = id

	return containerStat{statparts[0], statparts[1], timestamp, tagmap}
}
