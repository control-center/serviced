// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package api

import (
	"encoding/json"
	"fmt"
	"github.com/mattbaird/elastigo/core"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Export logs
func (a *api) ExportLogs(yyyymmdd, dirpath string) error {
	var e error
	indexName := "logstash-*"
	if yyyymmdd != "" {
		// 4 digits, optional non-digits, 2 digits, optional non-digits, 2 digits.
		re := regexp.MustCompile("\\A([0-9]{4})[^0-9]*([0-9]{2})[^0-9]*([0-9]{2})\\z")
		match := re.FindStringSubmatch(yyyymmdd)
		if match == nil {
			return fmt.Errorf("could not parse '%s' as yyyymmdd", yyyymmdd)
		}
		indexName = fmt.Sprintf("logstash-%s.%s.%s", match[1], match[2], match[3])
	}

	if dirpath == "" {
		dirpath, e = os.Getwd()
		if e != nil {
			return fmt.Errorf("could not determine current directory: %s", e)
		}
	}
	if fp, e := filepath.Abs(dirpath); e != nil {
		return fmt.Errorf("could not convert '%s' to an absolute path: %v", dirpath, e)
	} else {
		dirpath = filepath.Clean(fp)
	}

	tempdir, e := ioutil.TempDir("", "serviced-log-export-")
	if e != nil {
		return fmt.Errorf("could create temp directory: %s", e)
	}
	defer os.RemoveAll(tempdir)

	result, e := core.SearchUri(indexName, "", "*", "1m", 1000)
	if e != nil {
		return fmt.Errorf("failed to search elasticsearch: %s", e)
	}
	//TODO: Submit a patch to elastigo to support the "clear scroll" api. Add a "defer" here.

	hostFileLines := make(map[string]map[string]sortableLogLines)
	remaining := result.Hits.Total > 0
	for remaining {
		result, e = core.Scroll(false, result.ScrollId, "1m")
		hits := result.Hits.Hits
		total := len(hits)
		for i := 0; i < total; i++ {
			host, file, compactLines, e := parseLogSource(hits[i].Source)
			if e != nil {
				return e
			}
			for _, line := range compactLines {
				fileLines, found := hostFileLines[host]
				if !found {
					fileLines = make(map[string]sortableLogLines)
					hostFileLines[host] = fileLines
				}
				fileLines[file] = append(fileLines[file], line)
			}
		}
		remaining = len(hits) > 0
	}
	writeLogFile := func(index int, lines sortableLogLines) error {
		lines.Sort()
		filename := filepath.Join(tempdir, fmt.Sprintf("%d.log", index))
		file, e := os.Create(filename)
		if e != nil {
			return fmt.Errorf("failed to create file %s: %s", filename, e)
		}
		defer file.Close() //TODO: perhaps worry about error on close?
		for _, line := range lines {
			if _, e := file.WriteString(line.Message); e != nil {
				return fmt.Errorf("failed writing to file %s: %s", filename, e)
			}
			if _, e := file.WriteString("\n"); e != nil {
				return fmt.Errorf("failed writing to file %s: %s", filename, e)
			}
		}
		return nil
	}
	i := 0
	indexData := []string{"INDEX OF LOG FILES", "File\tHost\tOriginal Filename"}
	for host, fileLines := range hostFileLines {
		for file, lines := range fileLines {
			if e := writeLogFile(i, lines); e != nil {
				return e
			}
			indexData = append(indexData, fmt.Sprintf("%d.log\t%s\t%s", i, strconv.Quote(host), strconv.Quote(file)))
			i++
		}
	}
	indexData = append(indexData, "")
	indexFile := filepath.Join(tempdir, "index.txt")
	e = ioutil.WriteFile(indexFile, []byte(strings.Join(indexData, "\n")), 0644)
	if e != nil {
		return fmt.Errorf("failed writing to %s: %s", indexFile, e)
	}

	tgzfile := filepath.Join(dirpath, "serviced-log-export.tgz")

	cmd := exec.Command("tar", "-czf", tgzfile, "-C", filepath.Dir(tempdir), filepath.Base(tempdir))
	if output, e := cmd.CombinedOutput(); e != nil {
		return fmt.Errorf("failed to write tgz cmd:%+v, error:%v, output:%s", cmd, e, string(output))
	}
	return nil
}

type logLine struct {
	Host      string    `json:"host"`
	File      string    `json:"file"`
	Timestamp time.Time `json:"@timestamp"`
	Offset    string    `json:"offset"`
	Message   string    `json:"message"`
}

type logMultiLine struct {
	Host      string    `json:"host"`
	File      string    `json:"file"`
	Timestamp time.Time `json:"@timestamp"`
	Offset    []string  `json:"offset"`
	Message   string    `json:"message"`
}

type compactLogLine struct {
	Timestamp int64 //nanoseconds since the epoch, truncated at the minute to hide jitter
	Offset    int64
	Message   string
}

var newline = regexp.MustCompile("\\r?\\n")

// return: host, file, lines, error
func parseLogSource(source []byte) (string, string, []compactLogLine, error) {

	var line logLine
	if e := json.Unmarshal(source, &line); e == nil {
		offset, e := strconv.ParseInt(line.Offset, 10, 64)
		if e != nil {
			return "", "", nil, fmt.Errorf("failed to parse offset \"%s\" in \"%s\": %s", line.Offset, source, e)
		}
		compactLine := compactLogLine{
			Timestamp: truncateToMinute(line.Timestamp.UnixNano()),
			Offset:    offset,
			Message:   line.Message,
		}
		return line.Host, line.File, []compactLogLine{compactLine}, nil
	}
	var multiLine logMultiLine
	if e := json.Unmarshal(source, &multiLine); e != nil {
		return "", "", nil, fmt.Errorf("failed to parse JSON \"%s\": %s", source, e)
	}
	messages := newline.Split(multiLine.Message, -1)
	if len(messages) != len(multiLine.Offset) {
		return "", "", nil, fmt.Errorf("offsets do not correspond with lines: %s", source)
	}
	timestamp := truncateToMinute(multiLine.Timestamp.UnixNano())
	compactLines := make([]compactLogLine, len(messages))
	for i, offsetString := range multiLine.Offset {
		offset, e := strconv.ParseInt(offsetString, 10, 64)
		if e != nil {
			return "", "", nil, fmt.Errorf("failed to parse offset \"%s\" in \"%s\": %s", offsetString, source, e)
		}
		compactLines = append(compactLines, compactLogLine{
			Timestamp: timestamp,
			Offset:    offset,
			Message:   messages[i],
		})
	}
	return multiLine.Host, multiLine.File, compactLines, nil
}

func truncateToMinute(nanos int64) int64 {
	return nanos / int64(time.Minute) * int64(time.Minute)
}

type sortableLogLines []compactLogLine

func (this sortableLogLines) Len() int {
	return len(this)
}

func (this sortableLogLines) Swap(i, j int) {
	this[i], this[j] = this[j], this[i]
}

func (this sortableLogLines) Less(i, j int) bool {
	cmp := this[i].Timestamp - this[j].Timestamp
	if cmp < 0 {
		return true
	}
	if cmp > 0 {
		return false
	}

	if this[i].Offset < this[j].Offset {
		return true
	}

	return false
}

func (this sortableLogLines) Sort() {
	sort.Sort(this)
}
