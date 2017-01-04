// Copyright 2016 The Serviced Authors.
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

// Package stats collects serviced metrics and posts them to the TSDB.
package stats

import (
	"github.com/control-center/serviced/logging"

	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

var (
	statsReqUserAgent   = []string{"Zenoss Metric Publisher"}
	statsReqContentType = []string{"application/json"}
	plog                = logging.PackageLogger()
)

type gatherStatsFunc func(time.Time) []Sample
type updateStatsFunc func()

// StatsReporterInterface declares functions that may be useful for unit tests
type StatsReporterInterface interface {
	Close()
	gatherStatsFunc()
	updateStatsFunc()
}

// statsReporter collects and posts stats to the TSDB
type statsReporter struct {
	destination     string
	closeChannel    chan struct{}
	updateStatsFunc updateStatsFunc
	gatherStatsFunc gatherStatsFunc
}

// Sample is a single metric measurement
type Sample struct {
	Metric    string            `json:"metric"`
	Value     string            `json:"value"`
	Timestamp int64             `json:"timestamp"`
	Tags      map[string]string `json:"tags"`
}

// Updates the default registry, fills out the metric consumer format, and posts
// the data to the TSDB. Stops when close signal is received on closeChannel.
func (sr *statsReporter) report(d time.Duration) {
	tc := time.Tick(d)
	plog.Infof("collecting internal metrics at %s intervals", d)
	for {
		select {
		case _ = <-sr.closeChannel:
			plog.Info("Ceasing stat reporting.")
			return
		case t := <-tc:
			sr.updateStatsFunc()
			stats := sr.gatherStatsFunc(t)
			err := Post(sr.destination, stats)
			if err != nil {
				plog.Errorf("Error reporting stats: %v", err)
			}
		}
	}
}

// Close shuts down the reporting goroutine.
func (sr *statsReporter) Close() {
	close(sr.closeChannel)

}

// Post sends the list of stats to the TSDB.
func Post(destination string, stats []Sample) error {
	payload := map[string][]Sample{"metrics": stats}
	data, err := json.Marshal(payload)
	if err != nil {
		plog.Warningf("Couldn't marshal stats: ", err)
		return err
	}
	statsreq, err := http.NewRequest("POST", destination, bytes.NewBuffer(data))
	if err != nil {
		plog.Warningf("Couldn't create stats request: ", err)
		return err
	}
	statsreq.Header["User-Agent"] = statsReqUserAgent
	statsreq.Header["Content-Type"] = statsReqContentType
	resp, reqerr := http.DefaultClient.Do(statsreq)
	if reqerr != nil {
		plog.Warningf("Couldn't post container stats: %s", reqerr)
		return reqerr
	}
	defer resp.Body.Close()
	if !strings.Contains(resp.Status, "200 OK") {
		plog.Warningf("Post for container stats failed: %s", resp.Status)
		return nil
	}
	return nil
}
