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
	"archive/tar"
	"compress/gzip"
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

	"github.com/Sirupsen/logrus"
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/volume"
	elastigocore "github.com/zenoss/elastigo/core"
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
	// A list of one or more serviced IDs to export logs for (including children)
	ServiceIDs []string

	// In the format yyyy.mm.dd (inclusive), "" means unbounded
	FromDate string

	// In the format yyyy.mm.dd (inclusive), "" means unbounded
	ToDate string

	// Name of the compressed tar file containing all of the exported logs. If not specified, defaults to
	// "./serviced-log-export-<TIMESTAMP>.tgz" where <TIMESTAMP> is an RFC3339-like string (e.g. 2016-06-02T143843Z)
	OutFileName string

	// Set to true to default more verbose logging
	Debug bool

	// Driver to work with logstash ES instance; if nil a default driver will be used. Primarily used for testing.
	Driver ExportLogDriver

	// the file opened for OutFileName
	outFile *os.File
}

// logExporter is used internally to manage various operations required to export the application logs. It serves
// as a container for all of the intermediate data used during the export operation.
type logExporter struct {
	ExportLogsConfig

	// The ES-logstash query string
	query string

	// A list of dates for which ES-logstash has logs from any service
	days []string

	// The fully-qualified path to the temporary directory used to accummulate the application logs
	tempdir string

	// The name of the file used to store the collected set of parser warnings
	parseWarningsFilename string
	parseWarningsFile     *os.File

	// A map of hostid to host info used to populate the index file on completion of the export
	hostMap map[string]host.Host

	// A list of the output files created by the export process
	outputFiles []outputFileInfo

	// A list of services used to populate the index file on completion of the export
	serviceMap map[string]service.Service
}

// ExportLogs exports logs from ElasticSearch.
func (a *api) ExportLogs(configParam ExportLogsConfig) (err error) {
	var e error
	e = validateConfiguration(&configParam)
	if e != nil {
		return e
	}

	var exporter *logExporter
	defer func() {
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

	log := log.WithFields(logrus.Fields{
		"tmpdir": exporter.tempdir,
	})
	log.Info("Starting part 1 of 3: Processing Logstash Elastic results")
	foundIndexedDay, numWarnings, e = exporter.retrieveLogs()
	if e != nil {
		return e
	} else if !foundIndexedDay {
		return fmt.Errorf("no logstash indexes exist for the given date range %s - %s", exporter.FromDate, exporter.ToDate)
	}

	log.WithFields(logrus.Fields{
		"numfiles": len(exporter.outputFiles),
	}).Info("Starting part 2 of 3: Sort output files")

	indexData := []string{}
	for i, outputFile := range exporter.outputFiles {
		if e := exporter.organizeAndGroomLogFile(i, numWarnings); e != nil {
			return e
		}

		indexData = append(indexData, fmt.Sprintf("%03d.log\t%d\t%s\t%s\t%s\t%s\t%s\t%s",
			i,
			outputFile.LineCount,
			strconv.Quote(exporter.getHostName(outputFile.HostID)),
			strconv.Quote(outputFile.HostID),
			strconv.Quote(outputFile.ContainerID),
			strconv.Quote(exporter.getServiceName(outputFile.ServiceID)),
			strconv.Quote(outputFile.ServiceID),
			strconv.Quote(outputFile.LogFileName)))
	}
	sort.Strings(indexData)
	indexData = append([]string{"INDEX OF LOG FILES", "File\tLine Count\tHost Name\tHost ID\tContainer ID\tService Name\tService ID\tOriginal Filename"}, indexData...)
	indexData = append(indexData, "")
	indexFile := filepath.Join(exporter.tempdir, "index.txt")
	e = ioutil.WriteFile(indexFile, []byte(strings.Join(indexData, "\n")), 0644)
	if e != nil {
		return fmt.Errorf("failed writing to %s: %s", indexFile, e)
	}

	log.WithFields(logrus.Fields{
		"outfile": exporter.OutFileName,
	}).Info("Starting part 3 of 3: Generate tar file")
	gz := gzip.NewWriter(exporter.outFile)
	defer gz.Close()
	tarfile := tar.NewWriter(gz)
	defer tarfile.Close()
	if e := volume.ExportDirectory(tarfile, exporter.tempdir, filepath.Base(exporter.tempdir)); e != nil {
		return fmt.Errorf("failed to write tgz %s: %s", exporter.OutFileName, e)
	}

	if numWarnings != 0 {
		log.WithFields(logrus.Fields{
			"warnfile": filepath.Join(filepath.Base(exporter.tempdir), filepath.Base(exporter.parseWarningsFilename)),
		}).Info("Warnings for log parse are included in the tar file")
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
	if config.OutFileName == "" {
		pwd, e := os.Getwd()
		if e != nil {
			return fmt.Errorf("could not determine current directory: %s", e)
		}
		now := time.Now().UTC()
		// time.RFC3339 = "2006-01-02T15:04:05Z07:00"
		nowString := strings.Replace(now.Format(time.RFC3339), ":", "", -1)
		config.OutFileName = filepath.Join(pwd, fmt.Sprintf("serviced-log-export-%s.tgz", nowString))
	}
	fp, e := filepath.Abs(config.OutFileName)
	if e != nil {
		return fmt.Errorf("could not convert '%s' to an absolute path: %v", config.OutFileName, e)
	}
	config.OutFileName = filepath.Clean(fp)

	// Create the file in exclusive mode to avoid race with a concurrent invocation
	// of the same command on the same node
	config.outFile, e = os.OpenFile(config.OutFileName, os.O_RDWR|os.O_CREATE|os.O_TRUNC|os.O_EXCL, 0666)
	if e != nil {
		return fmt.Errorf("Could not create backup for %s: %s", config.OutFileName, e)
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
		log.WithFields(logrus.Fields{
			"from": config.FromDate,
			"to":   config.ToDate,
		}).Info("Normalized date range")
	}
	return nil
}

// Builds an instance of logExporter to use for the current export operation.
func buildExporter(configParam ExportLogsConfig, getServices func() ([]service.Service, error), getHostMap func() (map[string]host.Host, error)) (exporter *logExporter, err error) {
	exporter = &logExporter{ExportLogsConfig: configParam}
	exporter.query, err = exporter.buildQuery(getServices)
	if err != nil {
		return nil, fmt.Errorf("Could not build query: %s", err)
	}

	exporter.days, err = exporter.Driver.LogstashDays()
	if err != nil {
		return nil, fmt.Errorf("could not determine range of days in the logstash repo: %s", err)
	} else if exporter.Debug {
		log.WithFields(logrus.Fields{
			"days": len(exporter.days),
		}).Info("Found logs")
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

	// get services and build up servicemap
	exporter.serviceMap, err = buildServiceMap(getServices)
	if err != nil {
		return nil, fmt.Errorf("could not build service map: %s", err)
	}

	return exporter, nil
}

func buildServiceMap(getServices func() ([]service.Service, error)) (map[string]service.Service, error) {
	result := make(map[string]service.Service)
	if serviceArray, err := getServices(); err != nil {
		return nil, fmt.Errorf("failed to get list of services: %s", err)
	} else {
		for _, svc := range serviceArray {
			result[svc.ID] = svc
		}
	}
	return result, nil
}

// Responsible for cleaning up all of the files/directories created during the export
func (exporter *logExporter) cleanup() {
	// Close all of the temporary files
	for _, outputFile := range exporter.outputFiles {
		if e := outputFile.File.Close(); e != nil {
			log.WithFields(logrus.Fields{
				"outputfile": outputFile.Name,
			}).WithError(e).Error("Failed to close file cleanly")
		}
	}

	if exporter.parseWarningsFile != nil {
		if e := exporter.parseWarningsFile.Close(); e != nil {
			log.WithFields(logrus.Fields{
				"warningsfile": exporter.parseWarningsFilename,
			}).WithError(e).Error("Failed to close file cleanly")
		}
	}

	// close the tar file
	if exporter.outFile != nil {
		exporter.outFile.Close()
	}

	if exporter.tempdir != "" {
		os.RemoveAll(exporter.tempdir)
	}
}

// Builds an ES-logstash query string based on the list of service IDs requested.
func (exporter *logExporter) buildQuery(getServices func() ([]service.Service, error)) (string, error) {
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
		log.WithFields(logrus.Fields{
			"query": query,
		}).Info("Looking for services")
	}
	return query, nil
}

// Retrieve log data from ES-logstash, writing the log messages to separate files based on each unique
// combination of containerID and log-file-name. Depending on the lifespan of the container and the date range
// defined by exporter.days, some of the files created by this method may have messages from multiple dates.
func (exporter *logExporter) retrieveLogs() (foundIndexedDay bool, numWarnings int, e error) {
	numWarnings = 0
	foundIndexedDay = false
	// fileIndex is a map of containerID => map of app log filename => index into exporter.outputFiles
	// for the info about that container/app-log instance
	fileIndex := make(map[string]map[string]int)
	for _, yyyymmdd := range exporter.days {
		// Skip the indexes that are filtered out by the date range
		if (exporter.FromDate != "" && yyyymmdd < exporter.FromDate) || (exporter.ToDate != "" && yyyymmdd > exporter.ToDate) {
			continue
		} else {
			foundIndexedDay = true
		}

		if exporter.Debug {
			log.WithFields(logrus.Fields{
				"date": yyyymmdd,
			}).Info("Querying logstash for given date")
		}

		result, e := exporter.Driver.StartSearch(yyyymmdd, exporter.query)
		if e != nil {
			return foundIndexedDay, numWarnings, fmt.Errorf("failed to search elasticsearch for day %s: %s", yyyymmdd, e)
		}

		if exporter.Debug {
			log.WithFields(logrus.Fields{
				"date":      yyyymmdd,
				"totalmsgs": result.Hits.Total,
			}).Info("Found log messages")
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

				if _, found := fileIndex[message.ContainerID]; !found {
					fileIndex[message.ContainerID] = make(map[string]int)
				}
				// add a new tempfile
				if _, found := fileIndex[message.ContainerID][message.LogFileName]; !found {
					index := len(exporter.outputFiles)
					filename := filepath.Join(exporter.tempdir, fmt.Sprintf("%03d.log", index))
					file, e := os.Create(filename)
					if e != nil {
						return foundIndexedDay, numWarnings, fmt.Errorf("failed to create file %s: %s", filename, e)
					}
					fileIndex[message.ContainerID][message.LogFileName] = index
					outputFile := outputFileInfo{
						File:        file,
						Name:        filename,
						HostID:      message.HostID,
						ContainerID: message.ContainerID,
						LogFileName: message.LogFileName,
						ServiceID:   message.ServiceID,
					}
					exporter.outputFiles = append(exporter.outputFiles, outputFile)
					if exporter.Debug {
						log.WithFields(logrus.Fields{
							"containerid": outputFile.ContainerID,
							"logfile":     outputFile.LogFileName,
						}).Info("Writing messages")
					}
				}
				index := fileIndex[message.ContainerID][message.LogFileName]
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

// Organize and cleanup the log messages retrieved from logstash.
//
// The data returned by ES-logstash is not necessarily ordered by time.  The output files created by retrieveLogs()
// are formatted as "<LogstashTimestamp> <offset> <logMessage>". This method will sort the output file content by
// timestamp and offset, and then truncate those fields from the output file such that all that remains are the
// application log messages ordered from oldest first to newest last.
func (exporter *logExporter) organizeAndGroomLogFile(fileIndex, numWarnings int) error {
	filename := filepath.Join(exporter.tempdir, fmt.Sprintf("%03d.log", fileIndex))
	tmpfilename := filepath.Join(exporter.tempdir, fmt.Sprintf("%03d.log.tmp", fileIndex))
	cmd := exec.Command("sort", filename, "-uo", tmpfilename)
	if output, e := cmd.CombinedOutput(); e != nil {
		return fmt.Errorf("failed sorting %s, error: %v, output: %s", filename, e, output)
	}
	if numWarnings == 0 {
		if e := os.Rename(tmpfilename, filename); e != nil {
			return fmt.Errorf("failed moving %s %s, error: %s", tmpfilename, filename, e)
		}
	} else {
		cmd = exec.Command("cp", tmpfilename, filename)
		if output, e := cmd.CombinedOutput(); e != nil {
			return fmt.Errorf("failed copying %s %s, error: %v, output: %s", tmpfilename, filename, e, output)
		}
	}

	// Remove the logstash timestamp and offset fields from the output file, so that all that remains is the
	// message reported by the application (which may or may not include an application-level timestamp).
	cmd = exec.Command("sed", "s/^[0-9a-f]*\\t[0-9a-f]*\\t//", "-i", filename)
	if output, e := cmd.CombinedOutput(); e != nil {
		return fmt.Errorf("failed stripping sort prefixes config.FromDate %s, error: %v, output: %s", filename, e, output)
	}
	return nil
}

// Finds a host name based on a hostID.
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

func (exporter *logExporter) getServiceName(serviceID string) string {
	if serviceID == "" {
		return ""
	} else if service, ok := exporter.serviceMap[serviceID]; !ok {
		return "unknown"
	} else {
		return service.Name
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
	ServiceID   string    `json:"service"`
}

type logMultiLine struct {
	HostID      string    `json:"ccWorkerID"`
	ContainerID string    `json:"host"`
	File        string    `json:"file"`
	Timestamp   time.Time `json:"@timestamp"`
	Offset      []string  `json:"offset"`
	Message     string    `json:"message"`
	ServiceID   string    `json:"service"`
}

type compactLogLine struct {
	Timestamp int64 //nanoseconds since the epoch, truncated at the minute to hide jitter
	Offset    uint64
	Message   string
}

// Represents a set of log messages from a since instance of a single application log
type parsedMessage struct {
	HostID      string           // The ID of the CC worker node that hosted ContainerID
	ContainerID string           // The original Container ID of the application log file
	LogFileName string           // The name of the application log file
	Lines       []compactLogLine // One or more lines of log messages from the file.
	Warnings    string           // Warnings from the our logstash results parser
	ServiceID   string           // The service ID of the service
}

// Represents one file output by this routine
type outputFileInfo struct {
	File        *os.File // The temporary file on disk holding the log messages
	Name        string   // The name of the temporary file on disk holding the log mesasges
	HostID      string   // The ID of the CC worker node that hosted ContainerID
	ContainerID string   // The original Container ID of the application log file
	LogFileName string   // The name of the application log file
	LineCount   int      // number of message lines in the application log file
	ServiceID   string   // The service ID of the service
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
			ServiceID:   line.ServiceID,
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
		ServiceID:   multiLine.ServiceID,
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
