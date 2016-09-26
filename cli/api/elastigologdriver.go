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

package api

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/control-center/serviced/config"
	elastigo "github.com/zenoss/elastigo/api"
	elastigocore "github.com/zenoss/elastigo/core"
)

const (
	esLogStashIndexPrefix   = "logstash-" // Prefix of the logstash index names in Elasticsearch
	esLogstashScrollTimeout = "1m"        // tells ES how log to keep a scroll request 'alive'
)

type elastigoLogDriver struct {
	Domain string // hostname/ip of the Logstash ES instance
	Port   string // port number of the Logstash ES instance
}

// Verify that the elastigoLogDriver implements the interface
var _ ExportLogDriver = &elastigoLogDriver{}

// Sets the ES Logstash connection info; logstashES should be in the format hostname:port
func (driver *elastigoLogDriver) SetLogstashInfo(logstashES string) error {
	parts := strings.Split(logstashES, ":")
	if len(parts) != 2 {
		return fmt.Errorf("invalid logstash-es host:port %s", config.GetOptions().LogstashES)
	}
	elastigo.Domain = parts[0]
	elastigo.Port = parts[1]

	driver.Domain = elastigo.Domain
	driver.Port = elastigo.Port
	return nil
}

// Returns a list of all the dates for which a logstash-YYYY.MM.DD index is available in ES logstash.
// The strings are in YYYY.MM.DD format, and in reverse chronological order.
func (driver *elastigoLogDriver) LogstashDays() ([]string, error) {
	response, e := elastigo.DoCommand("GET", "/_aliases", nil)
	if e != nil {
		return []string{}, fmt.Errorf("couldn't fetch list of indices: %s", e)
	}
	var aliasMap map[string]interface{}
	if e = json.Unmarshal(response, &aliasMap); e != nil {
		return []string{}, fmt.Errorf("couldn't parse response (%s): %s", response, e)
	}
	result := make([]string, 0, len(aliasMap))
	for index := range aliasMap {
		if trimmed := strings.TrimPrefix(index, esLogStashIndexPrefix); trimmed != index {
			if trimmed, e = NormalizeYYYYMMDD(trimmed); e != nil {
				trimmed = ""
			}
			result = append(result, trimmed)
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(result)))
	return result, nil
}

// Start a new search of ES logstash for a given date
// params:
//   @date:  specifies the date to search against in the format YYYYMMDD
//   @query: valid string in lucene search syntax
//
// For more info on the details of the ES search params, see
// https://www.elastic.co/guide/en/elasticsearch/reference/0.90/search-request-search-type.html
func (driver *elastigoLogDriver) StartSearch(date string, query string) (elastigocore.SearchResult, error) {
	logstashIndex := fmt.Sprintf("%s%s", esLogStashIndexPrefix, date)
	scanSize := 1000
	return elastigocore.SearchUri(logstashIndex, "", query, esLogstashScrollTimeout, scanSize)
}

// Scroll to the next set of search results
func (driver *elastigoLogDriver) ScrollSearch(scrollID string) (elastigocore.SearchResult, error) {
	prettyPrint := false
	return elastigocore.Scroll(prettyPrint, scrollID, esLogstashScrollTimeout)
}
