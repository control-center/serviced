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

package container

import (
	"github.com/control-center/serviced/stats"
	"github.com/zenoss/glog"

	"io/ioutil"
	"path"
	"strconv"
	"strings"
	"time"
	"os"
	"bufio"
)

// statReporter perically collects statistics at the given
// interval until the closing channel closes
func statReporter(statsUrl string, interval time.Duration) {

	tick := time.Tick(interval)
	for {
		select {
		case t := <-tick:
			collect(t, statsUrl)
		}
	}
}

var eth0StatsDir = "/sys/devices/virtual/net/eth0/statistics"
var procNetFiles = map[string]string{
	"tcp": "/proc/net/tcp",
	"udp": "/proc/net/udp",
	"raw": "/proc/net/raw",
}

func collect(ts time.Time, statsUrl string) {
	// TODO: At some point we can look at refactoring this to use the
	// 'serviced metric' code

	// collect eth0 statistics
	netStats, err := readInt64Stats(eth0StatsDir)
	if err != nil {
		glog.Errorf("Could not collect eth0 stats: %s", err)
		return
	}

	// convert netStats to samples
	samples := make([]stats.Sample, len(netStats))
	now := ts.Unix()
	tags := map[string]string{"component": "eth0"}
	i := 0
	for name, value := range netStats {
		samples[i] = stats.Sample{
			Metric:    "net." + name,
			Value:     strconv.FormatInt(value, 10),
			Timestamp: now,
			Tags:      tags,
		}
		i++
	}

	// collect open connection statistics
	for proto, procFile := range procNetFiles {
		conns, err := getOpenConnections(procFile)
		if err != nil {
			glog.Errorf("Could not collect open connection information: %s", err)
			return
		}

		sample := stats.Sample{
			Metric:    "net.open_connections." + proto,
			Value:     strconv.FormatInt(int64(conns), 10),
			Timestamp: now,
			Tags:      map[string]string{"protocol": proto},
		}
		samples = append(samples, sample)
	}


	glog.V(4).Infof("posting samples: %+v", samples)
	if err := stats.Post(statsUrl, samples); err != nil {
		glog.Errorf("could not post stats: %s", err)
	}
}

// Read all the files in a directory that contain integers and return a
// map of those values
func readInt64Stats(dir string) (results map[string]int64, err error) {
	finfos, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	results = make(map[string]int64)
	for _, finfo := range finfos {
		if finfo.IsDir() {
			continue
		}
		fname := path.Join(dir, finfo.Name())
		data, err := ioutil.ReadFile(fname)
		if err != nil {
			return nil, err
		}
		i, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
		if err != nil {
			return nil, err
		}
		results[finfo.Name()] = i
	}
	return results, nil
}

func getOpenConnections(fileLoc string) (openConns int, err error) {
	file, err := os.Open(fileLoc)
	if err != nil {
		return -1, err
	}
	defer file.Close()

	openConns = 0
	scanner := bufio.NewScanner(file)
	scanner.Scan() // Skip the first line of headers
	for scanner.Scan() {
		splitString := strings.Fields(scanner.Text())
		if len(splitString) > 4 {
			if splitString[3] != "06" {
				openConns++
			}
		} else {
			glog.Errorf("Unable to read open connection information from %s", fileLoc)
		}
	}
	return openConns, scanner.Err()
}
