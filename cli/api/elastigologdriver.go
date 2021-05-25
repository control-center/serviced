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
	"github.com/control-center/serviced/config"
	"github.com/elastic/go-elasticsearch/v7"
	"sort"
	"strings"
	"time"
)

const (
	esLogStashIndexPrefix   = "logstash-" // Prefix of the logstash index names in Elasticsearch
	esLogstashScrollTimeout =  time.Minute       // tells ES how log to keep a scroll request 'alive'
)

type elastigoLogDriver struct {
	Domain string // hostname/ip of the Logstash ES instance
	Port   string // port number of the Logstash ES instance
	es 	   *elasticsearch.Client
}

// Verify that the elastigoLogDriver implements the interface
var _ ExportLogDriver = &elastigoLogDriver{}


// Sets the ES Logstash connection info; logstashES should be in the format hostname:port
func (driver *elastigoLogDriver) SetLogstashInfo(logstashES string) error {
	parts := strings.Split(logstashES, ":")
	if len(parts) != 2 {
		return fmt.Errorf("invalid logstash-es host:port %s", config.GetOptions().LogstashES)
	}

	es, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses: []string{
			fmt.Sprintf("http://%s:%s", parts[0], parts[1]),
		},
	})

	if err != nil {
		return fmt.Errorf("Error creating the client: %s", err)
	}

	driver.Domain = parts[0]
	driver.Port = parts[1]
	driver.es = es

	return nil
}

// Returns a list of all the dates for which a logstash-YYYY.MM.DD index is available in ES logstash.
// The strings are in YYYY.MM.DD format, and in reverse chronological order.

func (driver *elastigoLogDriver) LogstashDays() ([]string, error) {

	response, e := driver.es.Cat.Indices(driver.es.Cat.Indices.WithFormat("JSON"))
	if e != nil {
		return []string{}, fmt.Errorf("couldn't fetch list of indices: %s", e)
	}

	defer response.Body.Close()

	var resMap []map[string]interface{}

	if err := json.NewDecoder(response.Body).Decode(&resMap); err != nil {
		return []string{}, fmt.Errorf("couldn't parse response (%s): %s", response, e)
	}

	result := make([]string, 0, len(resMap))
	for _, row := range resMap {
		if trimmed := strings.TrimPrefix(fmt.Sprint(row["index"]), esLogStashIndexPrefix); trimmed != row["index"] {
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

func (driver *elastigoLogDriver) StartSearch(date string, query string) (ElasticSearchResults, error) {
	logstashIndex := fmt.Sprintf("%s%s", esLogStashIndexPrefix, date)
	scanSize := 1000
	res, err := driver.es.Search(driver.es.Search.WithIndex(logstashIndex),
								 driver.es.Search.WithScroll(esLogstashScrollTimeout),
								 driver.es.Search.WithSize(scanSize),
								 driver.es.Search.WithQuery(query))

	if err != nil {
		return ElasticSearchResults{}, fmt.Errorf("Error getting response: %s", err)
	}

	defer res.Body.Close()

	if res.IsError() {
		var e map[string]interface{}
		if err := json.NewDecoder(res.Body).Decode(&e); err != nil {
			return ElasticSearchResults{}, fmt.Errorf("Error parsing the response body: %s", err)
		} else {
			// Print the response status and error information.
			return ElasticSearchResults{}, fmt.Errorf("[%s] %s: %s",
				res.Status(),
				e["error"].(map[string]interface{})["type"],
				e["error"].(map[string]interface{})["reason"],
			)
		}
	}

	var searchResults ElasticSearchResults
	if err := json.NewDecoder(res.Body).Decode(&searchResults); err != nil {
		return ElasticSearchResults{}, fmt.Errorf("Error parsing the response body: %s", err)
	}
	return searchResults, nil
}

// Scroll to the next set of search results

func (driver *elastigoLogDriver) ScrollSearch(scrollID string) (ElasticSearchResults, error) {

	res, err := driver.es.Scroll(driver.es.Scroll.WithScrollID(scrollID),
								 driver.es.Scroll.WithScroll(esLogstashScrollTimeout))

	if err != nil {
		return ElasticSearchResults{}, fmt.Errorf("Error getting response: %s", err)
	}

	defer res.Body.Close()

	if res.IsError() {
		var e map[string]interface{}
		if err = json.NewDecoder(res.Body).Decode(&e); err != nil {
			return ElasticSearchResults{}, fmt.Errorf("Error parsing the response body: %s", err)
		} else {
			// Print the response status and error information.
			return ElasticSearchResults{}, fmt.Errorf("[%s] %s: %s",
				res.Status(),
				e["error"].(map[string]interface{})["type"],
				e["error"].(map[string]interface{})["reason"],
			)
		}
	}

	var scrollResults ElasticSearchResults

	if err = json.NewDecoder(res.Body).Decode(&scrollResults); err != nil {
		return ElasticSearchResults{}, fmt.Errorf("Error parsing the response body: %s", err)
	}

	return scrollResults, nil
}

type ElasticSearchResults struct {
	Took     int            `json:"took,omitempty"`
	TimedOut bool           `json:"timed_out,omitempty"`
	Shards   map[string]int `json:"_shards,omitempty"`
	ScrollID string         `json:"_scroll_id,omitempty"`
	Hits     ExternalHit    `json:"hits,omitempty"`
}

type Total struct {
	Value    int    `json:"value"`
	Relation string `json:"relation"`
}

type InternalHit struct {
	Index       string                 `json:"_index"`
	Type        string                 `json:"_type"`
	Id          string                 `json:"_id"`
	Version     int                    `json:"_version,omitempty"`
	PrimaryTerm int                    `json:"_primary_term,omitempty"`
	SeqNo       int                    `json:"_seq_no,omitempty"`
	Score       float64                `json:"_score"`
	Source      *json.RawMessage       `json:"_source"`
}

type ExternalHit struct {
	Total    Total          `json:"total"`
	MaxScore float64        `json:"max_score"`
	Hits     []InternalHit    `json:"hits,omitempty"`
}
