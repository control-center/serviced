// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package main

import (
	"encoding/json"
	"fmt"
	"github.com/mattbaird/elastigo/core"
	"github.com/zenoss/glog"
	"sort"
	// "strconv"
	"time"
)

type logLine struct {
	Host      string    `json:"host"`
	File      string    `json:"file"`
	Timestamp time.Time `json:"@timestamp"`
	// Offset    string    `json:"offset"`
	Message string `json:"message"`
}

type compactLogLine struct {
	Host      int
	File      int
	Timestamp int64
	// Offset    int64
	Message string
}

type sortableLogLines struct {
	Lines             []compactLogLine
	HostIndices       map[string]int
	FileIndices       map[string]int
	SortedHostIndices []int
	SortedFileIndices []int
}

func newSortableLogLines() sortableLogLines {
	return sortableLogLines{
		Lines:       []compactLogLine{},
		HostIndices: make(map[string]int),
		FileIndices: make(map[string]int),
	}
}

// Not thread safe
func (this *sortableLogLines) NewCompactLogLine(line *logLine) compactLogLine {
	host, found := this.HostIndices[line.Host]
	if !found {
		host = len(this.HostIndices)
		this.HostIndices[line.Host] = host
	}
	file, found := this.FileIndices[line.File]
	if !found {
		file = len(this.FileIndices)
		this.FileIndices[line.File] = file
	}
	// offset, e := strconv.ParseInt(line.Offset, 10, 64)
	// if e != nil {
	// 	glog.Fatalf("Failed to parse int %s: %s", line.Offset, e)
	// }
	return compactLogLine{
		Host:      host,
		File:      file,
		Timestamp: line.Timestamp.UnixNano(), //TODO: too jittery, truncate at minute.
		// Offset:    offset,
		Message: line.Message,
	}
}

// Not thread safe
func (this *sortableLogLines) Add(line *logLine) {
	this.Lines = append(this.Lines, this.NewCompactLogLine(line))
}

func (this sortableLogLines) Len() int {
	return len(this.Lines)
}

func (this sortableLogLines) Swap(i, j int) {
	this.Lines[i], this.Lines[j] = this.Lines[j], this.Lines[i]
}

func (this sortableLogLines) Less(i, j int) bool {
	//TODO: get Offset working
	cmp := this.SortedHostIndices[this.Lines[i].Host] - this.SortedHostIndices[this.Lines[j].Host]
	if cmp < 0 {
		return true
	}
	if cmp > 0 {
		return false
	}

	cmp = this.SortedFileIndices[this.Lines[i].File] - this.SortedFileIndices[this.Lines[j].File]
	if cmp < 0 {
		return true
	}
	if cmp > 0 {
		return false
	}

	cmp64 := this.Lines[i].Timestamp - this.Lines[j].Timestamp
	if cmp64 < 0 {
		return true
	}
	// if cmp64 > 0 {
	// 	return false
	// }

	// if this.Lines[i].Offset < this.Lines[j].Offset {
	// 	return true
	// }

	return false
}

func (this sortableLogLines) Sort() {
	hosts := newStringIntPairs(this.HostIndices)
	files := newStringIntPairs(this.FileIndices)
	sort.Sort(hosts)
	sort.Sort(files)
	this.SortedHostIndices = hosts.Ints
	this.SortedFileIndices = files.Ints
	sort.Sort(this)
}

type stringIntPairs struct {
	Strings []string
	Ints    []int
}

func newStringIntPairs(m map[string]int) stringIntPairs {
	result := stringIntPairs{
		Strings: make([]string, len(m)),
		Ints:    make([]int, len(m)),
	}
	j := 0
	for s, i := range m {
		result.Strings[j] = s
		result.Ints[j] = i
		j++
	}
	return result
}

func (this stringIntPairs) Len() int {
	return len(this.Strings)
}

func (this stringIntPairs) Swap(i, j int) {
	this.Strings[i], this.Strings[j] = this.Strings[j], this.Strings[i]
	this.Ints[i], this.Ints[j] = this.Ints[j], this.Ints[i]
}

func (this stringIntPairs) Less(i, j int) bool {
	if this.Strings[i] < this.Strings[j] {
		return true
	}
	if this.Strings[i] > this.Strings[j] {
		return false
	}
	return this.Ints[i] < this.Ints[j]
}

func (cli *ServicedCli) CmdLogs(args ...string) error {
	cmd := Subcmd("logs", "", "Export logs")
	if err := cmd.Parse(args); err != nil {
		return nil
	}
	result, e := core.SearchUri("logstash-*", "", "*", "1m", 1000)
	if e != nil {
		glog.Fatalf("Failed to search elasticsearch: %s", e)
	}
	//TODO: Submit a patch to elastigo to support the "clear scroll" api. Add a "defer" here.

	lines := newSortableLogLines()
	remaining := result.Hits.Total > 0
	for remaining {
		result, e = core.Scroll(false, result.ScrollId, "1m")
		hits := result.Hits.Hits
		total := len(hits)
		for i := 0; i < total; i += 1 {
			var line logLine
			if e := json.Unmarshal(hits[i].Source, &line); e != nil {
				glog.Errorf("JSON: %s", hits[i].Source)
				glog.Fatalf("TODO: your head asplode: %s", e)
				return e
			}
			lines.Add(&line)
		}
		remaining = len(hits) > 0
	}

	fmt.Printf("Lines: %d", len(lines.Lines))
	lines.Sort()
	fmt.Printf("Lines: %d", len(lines.Lines))

	for _, line := range lines.Lines {
		//TODO: detect when transitioning to a new host or file, and either
		// write a nice big header and a delimiter (if we're streaming to stdout),
		// or start a new file.
		fmt.Println(line.Message) //TODO: support dumping files instead of stdout.
	}
	return nil
}
