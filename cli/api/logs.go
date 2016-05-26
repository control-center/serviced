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

package api

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/domain/service"
	elastigocore "github.com/zenoss/elastigo/core"
	"github.com/zenoss/glog"
)

// This interface is primarily provided for unit-testing ExportLogs().
// Admittedly a very leaky abstraction around only a handful of elastigo calls.
// A better implementation would be to design an interface around all of
// the elastigo APIs which are needed by ExportLogs and the datastore package. Alternatively, maybe we need create
// a package just for interfacing withES Logstash.
// Either way that effort is outside the scope of making ExportLogs() testable.
type ExportLogDriver interface {
	// Sets the ES Logstash connection info; logstashES should be in the format hostname:port
	SetLogstashInfo(logstashES string) error

	// Returns a list of all the dates for which logs are available
	// The strings are in YYYY.MM.DD format, and in reverse chronological order.
	LogstashDays() ([]string, error)

	// Start a new search of ES logstash for a given date
	StartSearch(logstashIndex string, query string) (elastigocore.SearchResult, error)

	// Scroll to the next set of search results
	ScrollSearch(scrollID string) (elastigocore.SearchResult, error)
}

// ExportLogsConfig is the deserialized object from the command-line
type ExportLogsConfig struct {
	ServiceIDs []string
	FromDate   string
	ToDate     string
	Outfile    string
	Debug      bool
	Driver ExportLogDriver // Driver to work with logstash ES instance; if nil a default driver will be used.
}

type logExporter struct {
	ExportLogsConfig
	query                 string
	days                  []string
	tempdir               string
	parseWarningsFilename string
	parseWarningsFile     *os.File
	hostMap               map[string]host.Host
	outputFiles           []outputFileInfo
	fileIndex             map[string]map[string]int     // containerID => filename => index
}

// ExportLogs exports logs from ElasticSearch.
// serviceIds: list of services to select (includes their children). Empty slice means no filter
// from: yyyy.mm.dd (inclusive), "" means unbounded
// to: yyyy.mm.dd (inclusive), "" means unbounded
// outfile: the exported logs will tgz'd and written here. "" means "./serviced-log-export.tgz".
//
// TODO: This code is racy - creating then erasing the output file does not
// guarantee that it will be safe to write to at the end of the function
func (a *api) ExportLogs(configParam ExportLogsConfig) (err error) {
	var e error
	e = validateConfiguration(&configParam)
	if e != nil {
		return e
	}

	var exporter *logExporter
	defer func () {
		if exporter != nil {
			exporter.cleanup()
		}
	}()

	exporter, e = buildExporter(configParam, a.GetServices, a.GetHostMap)
	if e != nil {
		return e
	}

	numWarnings := 0
	foundIndexedDay := false

	glog.Infof("Starting part 1 of 3: process logstash elasticsearch results using temporary dir: %s", exporter.tempdir)
	foundIndexedDay, numWarnings, e = exporter.retrieveLogs()
	if e != nil {
		return e
	} else if !foundIndexedDay {
		return fmt.Errorf("no logstash indexes exist for the given date range %s - %s", exporter.FromDate, exporter.ToDate)
	}

	glog.Infof("Starting part 2 of 3: sort %d output files", len(exporter.outputFiles))

	indexData := []string{}
	for _, logfileIndex := range exporter.fileIndex {
		for _, i := range logfileIndex {
			filename := filepath.Join(exporter.tempdir, fmt.Sprintf("%03d.log", i))
			tmpfilename := filepath.Join(exporter.tempdir, fmt.Sprintf("%03d.log.tmp", i))
			cmd := exec.Command("sort", filename, "-uo", tmpfilename)
			if output, e := cmd.CombinedOutput(); e != nil {
				return fmt.Errorf("failed sorting %s, error: %v, output: %s", filename, e, output)
			}
			if numWarnings == 0 {
				cmd = exec.Command("mv", tmpfilename, filename)
				if output, e := cmd.CombinedOutput(); e != nil {
					return fmt.Errorf("failed moving %s %s, error: %v, output: %s", tmpfilename, filename, e, output)
				}
			} else {
				cmd = exec.Command("cp", tmpfilename, filename)
				if output, e := cmd.CombinedOutput(); e != nil {
					return fmt.Errorf("failed moving %s %s, error: %v, output: %s", tmpfilename, filename, e, output)
				}
			}
			cmd = exec.Command("sed", "s/^[0-9a-f]*\\t[0-9a-f]*\\t//", "-i", filename)
			if output, e := cmd.CombinedOutput(); e != nil {
				return fmt.Errorf("failed stripping sort prefixes config.FromDate %s, error: %v, output: %s", filename, e, output)
			}
			outputFile := exporter.outputFiles[i]
			indexData = append(indexData, fmt.Sprintf("%03d.log\t%d\t%s\t%s\t%s\t%s",
				i,
				outputFile.LineCount,
				strconv.Quote(exporter.getHostName(outputFile.HostID)),
				strconv.Quote(outputFile.HostID),
				strconv.Quote(outputFile.ContainerID),
				strconv.Quote(outputFile.LogFileName)))
		}
	}
	sort.Strings(indexData)
	indexData = append([]string{"INDEX OF LOG FILES", "File\tLine Count\tHost Name\tHost ID\tContainer ID\tOriginal Filename"}, indexData...)
	indexData = append(indexData, "")
	indexFile := filepath.Join(exporter.tempdir, "index.txt")
	e = ioutil.WriteFile(indexFile, []byte(strings.Join(indexData, "\n")), 0644)
	if e != nil {
		return fmt.Errorf("failed writing to %s: %s", indexFile, e)
	}

	glog.Infof("Starting part 3 of 3: generate tar file: %s", exporter.Outfile)

	cmd := exec.Command("tar", "-czf", exporter.Outfile, "-C", filepath.Dir(exporter.tempdir), filepath.Base(exporter.tempdir))
	if output, e := cmd.CombinedOutput(); e != nil {
		return fmt.Errorf("failed to write tgz cmd:%+v, error:%v, output:%s", cmd, e, string(output))
	}

	if numWarnings != 0 {
		glog.Warningf("warnings for log parse are included in the tar file as: %s",
			filepath.Join(filepath.Base(exporter.tempdir), filepath.Base(exporter.parseWarningsFilename)))
	}

	return nil
}

func validateConfiguration(config *ExportLogsConfig) error {
	if config.Driver == nil {
		config.Driver = &elastigoLogDriver{}
	}

	err := config.Driver.SetLogstashInfo(options.LogstashES)
	if err != nil {
		return err
	}

	// make sure we can write to outfile
	if config.Outfile == "" {
		pwd, e := os.Getwd()
		if e != nil {
			return fmt.Errorf("could not determine current directory: %s", e)
		}
		now := time.Now().UTC()
		// time.RFC3339 = "2006-01-02T15:04:05Z07:00"
		nowString := strings.Replace(now.Format(time.RFC3339), ":", "", -1)
		config.Outfile = filepath.Join(pwd, fmt.Sprintf("serviced-log-export-%s.tgz", nowString))
	}
	fp, e := filepath.Abs(config.Outfile)
	if e != nil {
		return fmt.Errorf("could not convert '%s' to an absolute path: %v", config.Outfile, e)
	}
	config.Outfile = filepath.Clean(fp)
	tgzfile, e := os.Create(config.Outfile)
	if e != nil {
		return fmt.Errorf("could not create %s: %s", config.Outfile, e)
	}
	tgzfile.Close()
	if e = os.Remove(config.Outfile); e != nil {
		return fmt.Errorf("could not remove %s: %s", config.Outfile, e)
	}

	// Validate and normalize the date range filter attributes "from" and "to"
	if config.FromDate == "" && config.ToDate == "" {
		config.ToDate = time.Now().UTC().Format("2006.01.02")
		config.FromDate = time.Now().UTC().AddDate(0, 0, -1).Format("2006.01.02")
	}
	if config.FromDate != "" {
		if config.FromDate, e = NormalizeYYYYMMDD(config.FromDate); e != nil {
			return e
		}
	}
	if config.ToDate != "" {
		if config.ToDate, e = NormalizeYYYYMMDD(config.ToDate); e != nil {
			return e
		}
	}

	if config.Debug {
		glog.Infof("Normalized date range: %s - %s", config.FromDate, config.ToDate)
	}
	return nil
}

func buildExporter(configParam ExportLogsConfig, getServices func()([]service.Service, error), getHostMap func()(map[string]host.Host, error) ) (exporter *logExporter, err error) {
	exporter = &logExporter{ExportLogsConfig: configParam}
	exporter.fileIndex = make(map[string]map[string]int)
	exporter.query, err = exporter.buildQuery(getServices)
	if err != nil {
		return nil, fmt.Errorf("Could not build query: %s", err)
	}

	exporter.days, err = exporter.Driver.LogstashDays()
	if err != nil {
		return nil, fmt.Errorf("could not determine range of days in the logstash repo: %s", err)
	} else if exporter.Debug {
		glog.Infof("Found %d days of logs in logstash", len(exporter.days))
	}

	// Get a temporary directory
	exporter.tempdir, err = ioutil.TempDir("", "serviced-log-export-")
	if err != nil {
		return nil, fmt.Errorf("could not create temp directory: %s", err)
	}

	// create a file to hold parse warnings
	exporter.parseWarningsFilename = filepath.Join(exporter.tempdir, "warnings.log")
	exporter.parseWarningsFile, err = os.Create(exporter.parseWarningsFilename)
	if err != nil {
		return nil, fmt.Errorf("failed to create file %s: %s", exporter.parseWarningsFilename, err)
	}

	exporter.hostMap, err = getHostMap()
	if err != nil {
		return nil, fmt.Errorf("failed to get list of host: %s", err)
	}

	return exporter, nil
}
func (exporter *logExporter) cleanup() {
	// Close all of the temporary files when we're done
	for _, outputFile := range exporter.outputFiles {
		if e := outputFile.File.Close(); e != nil {
			glog.Errorf("failed to close file '%s' cleanly: %s", outputFile.Name, e)
		}
	}

	if exporter.parseWarningsFile != nil {
		if e := exporter.parseWarningsFile.Close(); e != nil {
			glog.Errorf("failed to close file '%s' cleanly: %s", exporter.parseWarningsFilename, e)
		}
	}

	if exporter.tempdir != "" {
		defer os.RemoveAll(exporter.tempdir)
	}
}

func (exporter *logExporter) buildQuery(getServices func()([]service.Service, error)) (string, error) {
	query := "*"
	if len(exporter.ServiceIDs) > 0 {
		services, e := getServices()
		if e != nil {
			return "", e
		}
		serviceMap := make(map[string]service.Service)
		for _, service := range services {
			serviceMap[service.ID] = service
		}
		serviceIDMap := make(map[string]bool) //includes serviceIds, and their children as well
		for _, serviceID := range exporter.ServiceIDs {
			serviceIDMap[serviceID] = true
		}
		for _, service := range services {
			srvc := service
			for {
				found := false
				for _, serviceID := range exporter.ServiceIDs {
					if srvc.ID == serviceID {
						serviceIDMap[service.ID] = true
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
				return "", fmt.Errorf("invalid service ID format: %s", serviceID)
			}
			queryParts = append(queryParts, fmt.Sprintf("\"%s\"", strings.Replace(serviceID, "-", "\\-", -1)))
		}
		// sort the query parts for predictable testability of the query string
		sort.Sort(sort.StringSlice(queryParts))

		query = fmt.Sprintf("service:(%s)", strings.Join(queryParts, " OR "))
	}
	if exporter.Debug {
		glog.Infof("Looking for services based on this query: %s", query)
	}
	return query, nil
}

func (exporter *logExporter) retrieveLogs() (foundIndexedDay bool, numWarnings int, e error) {
	numWarnings = 0
	foundIndexedDay = false
	for _, yyyymmdd := range exporter.days {
		// Skip the indexes that are filtered out by the date range
		if (exporter.FromDate != "" && yyyymmdd < exporter.FromDate) || (exporter.ToDate != "" && yyyymmdd > exporter.ToDate) {
			continue
		} else {
			foundIndexedDay = true
		}

		if exporter.Debug {
			glog.Infof("Querying logstash for day %s", yyyymmdd)
		}

		result, e := exporter.Driver.StartSearch(yyyymmdd, exporter.query)
		if e != nil {
			return foundIndexedDay, numWarnings, fmt.Errorf("failed to search elasticsearch for day %s: %s", yyyymmdd, e)
		}

		if exporter.Debug {
			glog.Infof("Found %d log messages for %s", result.Hits.Total, yyyymmdd)
		}

		//TODO: Submit a patch to elastigo to support the "clear scroll" api. Add a "defer" here.
		remaining := result.Hits.Total > 0
		for remaining {
			result, e = exporter.Driver.ScrollSearch(result.ScrollId)
			if e != nil {
				return foundIndexedDay, numWarnings, e
			}
			hits := result.Hits.Hits
			total := len(hits)
			for i := 0; i < total; i++ {
				message, e := parseLogSource(hits[i].Source)
				if e != nil {
					return foundIndexedDay, numWarnings, e
				}

				if _, found := exporter.fileIndex[message.ContainerID]; !found {
					exporter.fileIndex[message.ContainerID] = make(map[string]int)
				}
				// add a new tempfile
				if _, found := exporter.fileIndex[message.ContainerID][message.LogFileName]; !found {
					index := len(exporter.outputFiles)
					filename := filepath.Join(exporter.tempdir, fmt.Sprintf("%03d.log", index))
					file, e := os.Create(filename)
					if e != nil {
						return foundIndexedDay, numWarnings, fmt.Errorf("failed to create file %s: %s", filename, e)
					}
					exporter.fileIndex[message.ContainerID][message.LogFileName] = index
					outputFile := outputFileInfo{
						File:        file,
						Name:        filename,
						HostID:      message.HostID,
						ContainerID: message.ContainerID,
						LogFileName: message.LogFileName,
					}
					exporter.outputFiles = append(exporter.outputFiles, outputFile)
					if exporter.Debug {
						glog.Infof("Writing messages for ContainerID=%s Application Log=%s", outputFile.ContainerID, outputFile.LogFileName)
					}
				}
				index := exporter.fileIndex[message.ContainerID][message.LogFileName]
				exporter.outputFiles[index].LineCount += len(message.Lines)
				file := exporter.outputFiles[index].File
				filename := filepath.Join(exporter.tempdir, fmt.Sprintf("%03d.log", index))
				for _, line := range message.Lines {
					formatted := fmt.Sprintf("%016x\t%016x\t%s\n", line.Timestamp, line.Offset, line.Message)
					if _, e := file.WriteString(formatted); e != nil {
						return foundIndexedDay, numWarnings, fmt.Errorf("failed writing to file %s: %s", filename, e)
					}
				}
				if len(message.Warnings) > 0 {
					if _, e := exporter.parseWarningsFile.WriteString(message.Warnings); e != nil {
						return foundIndexedDay, numWarnings, fmt.Errorf("failed writing to file %s: %s", exporter.parseWarningsFilename, e)
					}
					numWarnings++
				}
			}
			remaining = len(hits) > 0
		}
	}
	return foundIndexedDay, numWarnings, nil
}

// Lookup a hostID in the map.
func (exporter *logExporter) getHostName(hostID string) string {
	if hostID == "" {
		// Probably a message that was logged before we added the 'ccWorkerID' field
		return ""
	} else if host, ok := exporter.hostMap[hostID]; !ok {
		// either the CC worker node was retired or
		// the hostID of the worker node was changed after we had captured some logs
		return "unknown"
	} else {
		return host.Name
	}
}

// NOTE: the logstash field named 'host' is hard-coded in logstash to be the value from `hostname`, only when
//       executed inside a docker container, the value is actually the container ID, not the name of docker host.
//       In later releases of CC, we added the field 'ccWorkerID' to have the hostID of the docker host. This
//       means that some installations may have older log messages with no 'ccWorkerID' field.
type logSingleLine struct {
	HostID      string    `json:"ccWorkerID"`
	ContainerID string    `json:"host"`
	File        string    `json:"file"`
	Timestamp   time.Time `json:"@timestamp"`
	Offset      string    `json:"offset"`
	Message     string    `json:"message"`
}

type logMultiLine struct {
	HostID      string    `json:"ccWorkerID"`
	ContainerID string    `json:"host"`
	File        string    `json:"file"`
	Timestamp   time.Time `json:"@timestamp"`
	Offset      []string  `json:"offset"`
	Message     string    `json:"message"`
}

type compactLogLine struct {
	Timestamp int64 //nanoseconds since the epoch, truncated at the minute to hide jitter
	Offset    uint64
	Message   string
}

// Represents a set of log messages from a since instance of a single application log
type parsedMessage struct {
	HostID      string              // The ID of the CC worker node that hosted ContainerID
	ContainerID string              // The original Container ID of the application log file
	LogFileName string              // The name of the application log file
	Lines       []compactLogLine    // One or more lines of log messages from the file.
	Warnings    string              // Warnings from the our logstash results parser
}

// Represents one file output by this routine
type outputFileInfo struct {
	File        *os.File            // The temporary file on disk holding the log messages
	Name        string              // The name of the temporary file on disk holding the log mesasges
	HostID      string              // The ID of the CC worker node that hosted ContainerID
	ContainerID string              // The original Container ID of the application log file
	LogFileName string              // The name of the application log file
	LineCount   int                 // number of message lines in the application log file
}

var newline = regexp.MustCompile("\\r?\\n")

// convertOffsets converts a list of strings into a list of uint64s
func convertOffsets(offsets []string) ([]uint64, error) {
	result := make([]uint64, len(offsets))
	for i, offsetString := range offsets {
		offset, e := strconv.ParseUint(offsetString, 10, 64)
		if e != nil {
			return result, fmt.Errorf("failed to parse offset[%d] \"%s\" in \"%s\": %s", i, offsetString, offsets, e)
		}
		result[i] = offset
	}

	return result, nil
}

// uint64sAreSorted returns true if input values are sorted in increasing order - mimics sort.IntsAreSorted()
func uint64sAreSorted(values []uint64) bool {
	if len(values) == 0 {
		return true
	}

	previousValue := values[0]
	for _, value := range values {
		if value < previousValue {
			return false
		}
		previousValue = value
	}
	return true
}

// getMinValue returns the minimum value in an array of uint64
func getMinValue(values []uint64) uint64 {
	result := uint64(math.MaxUint64)
	for _, value := range values {
		if value < result {
			result = value
		}
	}
	return result
}

// generateOffsets uses the minimum offset in the array as a base returns an array of offsets where
// each offset is the base + index
func generateOffsets(messages []string, offsets []uint64) []uint64 {
	result := make([]uint64, len(messages))
	minOffset := getMinValue(offsets)
	if minOffset == uint64(math.MaxUint64) {
		minOffset = 0
	}
	for i, _ := range result {
		result[i] = minOffset + uint64(i)
	}
	return result
}

// return: containerID, file, lines, error
func parseLogSource(source []byte) (*parsedMessage, error) {
	warnings := ""

	// attempt to unmarshal into singleLine
	var line logSingleLine
	if e := json.Unmarshal(source, &line); e == nil {
		offset := uint64(0)
		if len(line.Offset) != 0 {
			var e error
			offset, e = strconv.ParseUint(line.Offset, 10, 64)
			if e != nil {
				return nil, fmt.Errorf("failed to parse offset \"%s\" in \"%s\": %s", line.Offset, source, e)
			}
		}
		compactLine := compactLogLine{
			Timestamp: truncateToMinute(line.Timestamp.UnixNano()),
			Offset:    offset,
			Message:   line.Message,
		}
		message := &parsedMessage{
			HostID:      line.HostID,
			ContainerID: line.ContainerID,
			LogFileName: line.File,
			Lines:       []compactLogLine{compactLine},
		}
		return message, nil
	}

	// attempt to unmarshal into multiLine
	var multiLine logMultiLine
	if e := json.Unmarshal(source, &multiLine); e != nil {
		return nil, fmt.Errorf("failed to parse JSON \"%s\": %s", source, e)
	}

	// build offsets - list of uint64
	offsets, e := convertOffsets(multiLine.Offset)
	if e != nil {
		return nil, fmt.Errorf("failed to parse JSON \"%s\": %s", source, e)
	}

	// verify number of lines in message against number of offsets
	messages := newline.Split(multiLine.Message, -1)
	if len(offsets)+1 == len(messages) {
		warnings += fmt.Sprintf(
			"number of offsets for %s:%s (numLines:%d numOffsets:%d) is one less than number of lines: %s\n",
			multiLine.ContainerID, multiLine.File, len(messages), len(offsets), source)
		numLines := len(messages)
		if numLines > 1 {
			lastOffset := uint64(len(messages[numLines-2])) + offsets[numLines-2]
			offsets = append(offsets, lastOffset)
		}
	} else if len(offsets) > len(messages) {
		warnings += fmt.Sprintf(
			"number of offsets for %s:%s (numLines:%d numOffsets:%d) is greater than number of lines: %s\n",
			multiLine.ContainerID, multiLine.File, len(messages), len(multiLine.Offset), source)
		offsets = offsets[0:len(messages)]
	} else if len(offsets) < len(messages) {
		warnings += fmt.Sprintf(
			"number of offsets for %s:%s (numLines:%d numOffsets:%d) is less than number of lines: %s\n",
			multiLine.ContainerID, multiLine.File, len(messages), len(multiLine.Offset), source)
		offsets = generateOffsets(messages, offsets)
		warnings += fmt.Sprintf("new offsets: %v", offsets)
	}

	// deal with offsets that are not sorted in increasing order
	if !uint64sAreSorted(offsets) {
		warnings = fmt.Sprintf("offsets are not sorted: %v\n", offsets)
		offsets = generateOffsets(messages, offsets)
		warnings = fmt.Sprintf("new offsets: %v\n", offsets)
	}

	// build compactLines
	timestamp := truncateToMinute(multiLine.Timestamp.UnixNano())
	compactLines := make([]compactLogLine, len(messages))
	for i, offset := range offsets {
		compactLines = append(compactLines, compactLogLine{
			Timestamp: timestamp,
			Offset:    offset,
			Message:   messages[i],
		})
	}

	message := &parsedMessage{
		HostID:      multiLine.HostID,
		ContainerID: multiLine.ContainerID,
		LogFileName: multiLine.File,
		Lines:       compactLines,
		Warnings:    warnings,
	}
	return message, nil
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


func truncateToMinute(nanos int64) int64 {
	return nanos / int64(time.Minute) * int64(time.Minute)
}
