// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package api

import (
	"encoding/json"
	"fmt"
	elastigo "github.com/mattbaird/elastigo/api"
	"github.com/mattbaird/elastigo/core"
	"github.com/zenoss/serviced/domain/service"
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

// ExportLogs exports logs from ElasticSearch.
// serviceIds: list of services to select (includes their children). Empty slice means no filter
// from: yyyy.mm.dd (inclusive), "" means unbounded
// to: yyyy.mm.dd (inclusive), "" means unbounded
// outfile: the exported logs will tgz'd and written here. "" means "./serviced-log-export.tgz".
func (a *api) ExportLogs(serviceIds []string, from, to, outfile string) (err error) {
	var e error
	files := []*os.File{}
	fileIndex := make(map[string]map[string]int) // host => filename => index

	// make sure we can write to outfile
	if outfile == "" {
		pwd, e := os.Getwd()
		if e != nil {
			return fmt.Errorf("could not determine current directory: %s", e)
		}
		outfile = filepath.Join(pwd, "serviced-log-export.tgz")
	}
	fp, e := filepath.Abs(outfile)
	if e != nil {
		return fmt.Errorf("could not convert '%s' to an absolute path: %v", outfile, e)
	}
	outfile = filepath.Clean(fp)
	tgzfile, e := os.Create(outfile)
	if e != nil {
		return fmt.Errorf("could not create %s: %s", outfile, e)
	}
	tgzfile.Close()
	if e = os.Remove(outfile); e != nil {
		return fmt.Errorf("could not remove %s: %s", outfile, e)
	}

	// Validate and normalize the date range filter attributes "from" and "to"
	if from == "" && to == "" {
		to = time.Now().UTC().Format("2006.01.02")
		from = time.Now().UTC().AddDate(0, 0, -1).Format("2006.01.02")
	}
	if from != "" {
		if from, e = NormalizeYYYYMMDD(from); e != nil {
			return e
		}
	}
	if to != "" {
		if to, e = NormalizeYYYYMMDD(to); e != nil {
			return e
		}
	}

	query := "*"
	if len(serviceIds) > 0 {
		services, e := a.GetServices()
		if e != nil {
			return e
		}
		serviceMap := make(map[string]*service.Service)
		for _, service := range services {
			serviceMap[service.Id] = service
		}
		serviceIDMap := make(map[string]bool) //includes serviceIds, and their children as well
		for _, serviceID := range serviceIds {
			serviceIDMap[serviceID] = true
		}
		for _, service := range services {
			srvc := service
			for {
				found := false
				for _, serviceID := range serviceIds {
					if srvc.Id == serviceID {
						serviceIDMap[service.Id] = true
						found = true
						break
					}
				}
				if found || srvc.ParentServiceID == "" {
					break
				}
				srvc = serviceMap[srvc.ParentServiceID]
			}
		}
		re := regexp.MustCompile("\\A[\\w\\-]+\\z") //only letters, numbers, underscores, and dashes
		queryParts := []string{}
		for serviceID := range serviceIDMap {
			if re.FindStringIndex(serviceID) == nil {
				return fmt.Errorf("invalid service ID format: %s", serviceID)
			}
			queryParts = append(queryParts, fmt.Sprintf("\"%s\"", strings.Replace(serviceID, "-", "\\-", -1)))
		}
		query = fmt.Sprintf("service:(%s)", strings.Join(queryParts, " OR "))
	}

	// Get a temporary directory
	tempdir, e := ioutil.TempDir("", "serviced-log-export-")
	if e != nil {
		return fmt.Errorf("could not create temp directory: %s", e)
	}
	defer os.RemoveAll(tempdir)

	days, e := LogstashDays()
	if e != nil {
		return e
	}
	foundIndexedDay := false
	for _, yyyymmdd := range days {
		// Skip the indexes that are filtered out by the date range
		if (from != "" && yyyymmdd < from) || (to != "" && yyyymmdd > to) {
			continue
		} else {
			foundIndexedDay = true
		}

		logstashIndex := fmt.Sprintf("logstash-%s", yyyymmdd)
		result, e := core.SearchUri(logstashIndex, "", query, "1m", 1000)
		if e != nil {
			return fmt.Errorf("failed to search elasticsearch: %s", e)
		}
		//TODO: Submit a patch to elastigo to support the "clear scroll" api. Add a "defer" here.
		remaining := result.Hits.Total > 0
		for remaining {
			result, e = core.Scroll(false, result.ScrollId, "1m")
			hits := result.Hits.Hits
			total := len(hits)
			for i := 0; i < total; i++ {
				host, logfile, compactLines, e := parseLogSource(hits[i].Source)
				if e != nil {
					return e
				}
				if _, found := fileIndex[host]; !found {
					fileIndex[host] = make(map[string]int)
				}
				if _, found := fileIndex[host][logfile]; !found {
					index := len(files)
					filename := filepath.Join(tempdir, fmt.Sprintf("%03d.log", index))
					file, e := os.Create(filename)
					if e != nil {
						return fmt.Errorf("failed to create file %s: %s", filename, e)
					}
					defer func() {
						if e := file.Close(); e != nil && err == nil {
							err = fmt.Errorf("failed to close file '%s' cleanly: %s", filename, e)
						}
					}()
					fileIndex[host][logfile] = index
					files = append(files, file)
				}
				index := fileIndex[host][logfile]
				file := files[index]
				filename := filepath.Join(tempdir, fmt.Sprintf("%03d.log", index))
				for _, line := range compactLines {
					formatted := fmt.Sprintf("%016x\t%016x\t%s\n", line.Timestamp, line.Offset, line.Message)
					if _, e := file.WriteString(formatted); e != nil {
						return fmt.Errorf("failed writing to file %s: %s", filename, e)
					}
				}
			}
			remaining = len(hits) > 0
		}
	}
	if !foundIndexedDay {
		return fmt.Errorf("no logstash indexes exist for the given date range %s - %s", from, to)
	}

	indexData := []string{}
	for host, logfileIndex := range fileIndex {
		for logfile, i := range logfileIndex {
			filename := filepath.Join(tempdir, fmt.Sprintf("%03d.log", i))
			cmd := exec.Command("sort", filename, "-o", filename)
			if output, e := cmd.CombinedOutput(); e != nil {
				return fmt.Errorf("failed sorting %s, error: %v, output: %s", filename, e, output)
			}
			cmd = exec.Command("sed", "s/^[0-9a-f]*\\t[0-9a-f]*\\t//", "-i", filename)
			if output, e := cmd.CombinedOutput(); e != nil {
				return fmt.Errorf("failed stripping sort prefixes from %s, error: %v, output: %s", filename, e, output)
			}
			indexData = append(indexData, fmt.Sprintf("%03d.log\t%s\t%s", i, strconv.Quote(host), strconv.Quote(logfile)))
		}
	}
	sort.Strings(indexData)
	indexData = append([]string{"INDEX OF LOG FILES", "File\tHost\tOriginal Filename"}, indexData...)
	indexData = append(indexData, "")
	indexFile := filepath.Join(tempdir, "index.txt")
	e = ioutil.WriteFile(indexFile, []byte(strings.Join(indexData, "\n")), 0644)
	if e != nil {
		return fmt.Errorf("failed writing to %s: %s", indexFile, e)
	}

	cmd := exec.Command("tar", "-czf", outfile, "-C", filepath.Dir(tempdir), filepath.Base(tempdir))
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

// NormalizeYYYYMMDD matches optional non-digits, 4 digits, optional non-digits,
// 2 digits, optional non-digits, 2 digits, optional non-digits
// Returns those 8 digits formatted as "dddd.dd.dd", or error if unparseable.
func NormalizeYYYYMMDD(s string) (string, error) {
	match := yyyymmddMatcher.FindStringSubmatch(s)
	if match == nil {
		return "", fmt.Errorf("could not parse '%s' as yyyymmdd", s)
	}
	return fmt.Sprintf("%s.%s.%s", match[1], match[2], match[3]), nil
}

var yyyymmddMatcher = regexp.MustCompile("\\A[^0-9]*([0-9]{4})[^0-9]*([0-9]{2})[^0-9]*([0-9]{2})[^0-9]*\\z")

// Returns a list of all the dates with a logstash-YYYY.MM.DD index available in ElasticSearch.
// The strings are in YYYY.MM.DD format, and in reverse chronological order.
var LogstashDays = func() ([]string, error) {
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
		if trimmed := strings.TrimPrefix(index, "logstash-"); trimmed != index {
			if trimmed, e = NormalizeYYYYMMDD(trimmed); e != nil {
				trimmed = ""
			}
			result = append(result, trimmed)
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(result)))
	return result, nil
}

func truncateToMinute(nanos int64) int64 {
	return nanos / int64(time.Minute) * int64(time.Minute)
}
