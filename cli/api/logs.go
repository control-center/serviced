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
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/control-center/serviced/config"
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/volume"
)

type UnknownElasticStructError struct {
	UnknownData   string
	UnknownStruct string
}

func (e *UnknownElasticStructError) Error() string {
	return fmt.Sprintf("Expected elastic %s, got: %s", e.UnknownStruct, e.UnknownData)
}

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
	StartSearch(logstashIndex string, query string) (ElasticSearchResults, error)

	// Scroll to the next set of search results
	ScrollSearch(scrollID string) (ElasticSearchResults, error)
}

// The export process produces 1 or more <nnn>.log files containing the exported log messages. The values of
// ExportGroup control which messages are grouped together in a single <nnn>.log file:
//   GroupByContainer   Each <nnn>.log contains messages for the same unique combination of container ID and
//                      application log file name.
//   GroupByDay         Each <nnn>.log contains messages for the same calendar day.
//   GroupByService     Each <nnn>.log contains messages for the same logical service.
type ExportGroup int

const (
	GroupByContainerID ExportGroup = iota
	GroupByDay
	GroupByService
)

// ExportLogsConfig is the deserialized object from the command-line
type ExportLogsConfig struct {
	// A list of one or more serviced IDs to export logs for
	// (includes all child services unless ExcludeChildren is true)
	ServiceIDs []string

	// A list of one or more application log file names to export
	FileNames []string

	// In the format yyyy.mm.dd (inclusive), "" means unbounded
	FromDate string

	// In the format yyyy.mm.dd (inclusive), "" means unbounded
	ToDate string

	// Name of the compressed tar file containing all of the exported logs. If not specified, defaults to
	// "./serviced-log-export-<TIMESTAMP>.tgz" where <TIMESTAMP> is an RFC3339-like string (e.g. 2016-06-02T143843Z)
	OutFileName string

	// Set to true to default more verbose logging
	Debug bool

	// Defines which messages are grouped together in each output file
	GroupBy ExportGroup

	// Set to true to exclude child services
	ExcludeChildren bool

	// Driver to work with logstash ES instance; if nil a default driver will be used. Primarily used for testing.
	Driver ExportLogDriver

	// the file opened for OutFileName
	outFile *os.File
}

// logExporter is used internally to manage various operations required to export the application logs. It serves
// as a container for all of the intermediate data used during the export operation.
type logExporter struct {
	ExportLogsConfig

	// The timestamp for when the export started
	startTime time.Time

	// The string representation of startTime suitable for file/directory names
	timeLabel string

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
	serviceMap map[string]service.ServiceDetails

	// An index of information about outputFiles. Handles variations in group-by requirements.
	fileIndex FileIndex

	// The min/max dates found in logstash
	minDateFound string
	maxDateFound string
}

func (eg ExportGroup) String() string {
	switch eg {
	case GroupByContainerID:
		return "container"
	case GroupByDay:
		return "day"
	case GroupByService:
		return "service"
	default:
		return "undefined"
	}
}

func ExportGroupFromString(value string) ExportGroup {
	switch value {
	case "container":
		return GroupByContainerID
	case "day":
		return GroupByDay
	case "service":
		return GroupByService
	default:
		return -1
	}
}

// ExportLogs exports logs from ElasticSearch.
func (a *api) ExportLogs(configParam ExportLogsConfig) (err error) {

	startTime := time.Now().UTC()
	// time.RFC3339 = "2006-01-02T15:04:05Z07:00"
	timeLabel := strings.Replace(startTime.Format(time.RFC3339), ":", "", -1)

	e := validateConfiguration(&configParam, timeLabel)
	if e != nil {
		return e
	}

	var exporter *logExporter
	defer func() {
		if exporter != nil {
			exporter.cleanup()
		}
	}()

	exporter, e = buildExporter(configParam, startTime, timeLabel, a.GetAllServiceDetails, a.GetHostMap)
	if e != nil {
		return e
	}

	numWarnings := 0
	foundIndexedDay := false

	log := log.WithFields(logrus.Fields{
		"tmpdir": exporter.tempdir,
		"from":   exporter.FromDate,
		"to":     exporter.ToDate,
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
		indexData = append(indexData, exporter.fileIndex.GetFileIndexData(
			i,
			outputFile,
			exporter.getHostName(outputFile.HostID),
			exporter.getServiceName(outputFile.ServiceID)))
	}
	sort.Strings(indexData)

	servicesDesc := make([]string, 0)
	for _, id := range configParam.ServiceIDs {
		servicesDesc = append(servicesDesc, fmt.Sprintf("%s (%s)", exporter.getServiceName(id), id))
	}
	indexData = append([]string{
		"LOG EXPORT SUMMARY",
		fmt.Sprintf("       Export Ran On: %s", exporter.startTime.Format(time.RFC1123)),
		fmt.Sprintf("Available Date Range: %s - %s", exporter.days[len(exporter.days)-1], exporter.days[0]),
		fmt.Sprintf("     Requested Dates: %s - %s", configParam.FromDate, configParam.ToDate),
		fmt.Sprintf("      Exported Dates: %s - %s", exporter.minDateFound, exporter.maxDateFound),
		fmt.Sprintf("          Grouped By: %s", configParam.GroupBy),
		fmt.Sprintf("Requested Service(s): %s", strings.Join(servicesDesc, ", ")),
		fmt.Sprintf("Child Svcs Excluded?: %s", strconv.FormatBool(configParam.ExcludeChildren)),
		fmt.Sprintf("   Requested File(s): %s", strings.Join(configParam.FileNames, ", ")),
		"",
		"INDEX OF LOG FILES",
		exporter.fileIndex.GetIndexHeader()},
		indexData...)
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

func validateConfiguration(cfg *ExportLogsConfig, timeLabel string) error {
	if cfg.Driver == nil {
		cfg.Driver = &elastigoLogDriver{}
	}

	err := cfg.Driver.SetLogstashInfo(config.GetOptions().LogstashES)
	if err != nil {
		return err
	}

	// make sure we can write to outfile
	if cfg.OutFileName == "" {
		pwd, e := os.Getwd()
		if e != nil {
			return fmt.Errorf("could not determine current directory: %s", e)
		}
		cfg.OutFileName = filepath.Join(pwd, fmt.Sprintf("serviced-log-export-%s.tgz", timeLabel))
	}
	_, e := filepath.Abs(cfg.OutFileName)
	if e != nil {
		return fmt.Errorf("could not convert '%s' to an absolute path: %v", cfg.OutFileName, e)
	}

	// Create the file in exclusive mode to avoid race with a concurrent invocation
	// of the same command on the same node
	cfg.outFile, e = os.OpenFile(cfg.OutFileName, os.O_RDWR|os.O_CREATE|os.O_TRUNC|os.O_EXCL, 0666)
	if e != nil {
		return fmt.Errorf("Could not create backup for %s: %s", cfg.OutFileName, e)
	}

	// Validate and normalize the date range filter attributes "from" and "to"
	if cfg.FromDate == "" && cfg.ToDate == "" {
		cfg.ToDate = time.Now().UTC().Format("2006.01.02")
		cfg.FromDate = time.Now().UTC().AddDate(0, 0, -1).Format("2006.01.02")
	}
	if cfg.FromDate != "" {
		if cfg.FromDate, e = NormalizeYYYYMMDD(cfg.FromDate); e != nil {
			return e
		}
	}
	if cfg.ToDate != "" {
		if cfg.ToDate, e = NormalizeYYYYMMDD(cfg.ToDate); e != nil {
			return e
		}
	}

	if cfg.Debug {
		log.WithFields(logrus.Fields{
			"from": cfg.FromDate,
			"to":   cfg.ToDate,
		}).Info("Normalized date range")
	}
	return nil
}

// Builds an instance of logExporter to use for the current export operation.
func buildExporter(configParam ExportLogsConfig, startTime time.Time, timeLabel string, getServices func() ([]service.ServiceDetails, error), getHostMap func() (map[string]host.Host, error)) (exporter *logExporter, err error) {
	exporter = &logExporter{
		ExportLogsConfig: configParam,
		startTime:        startTime,
		timeLabel:        timeLabel,
	}

	exporter.query, err = exporter.buildQuery(getServices)
	if err != nil {
		return nil, fmt.Errorf("Could not build query: %s", err)
	}

	exporter.days, err = exporter.Driver.LogstashDays()
	if err != nil {
		return nil, fmt.Errorf("could not determine range of days in the logstash repo: %s", err)
	} else if exporter.Debug {
		log.WithFields(logrus.Fields{
			"ndays": len(exporter.days),
			"to":    exporter.days[0],
			"from":  exporter.days[len(exporter.days)-1],
		}).Info("Found logs")
	}

	// Get a temporary directory
	exporter.tempdir, err = ioutil.TempDir("", "serviced-log-export-")
	if err != nil {
		return nil, fmt.Errorf("could not create temp directory: %s", err)
	}
	exporter.tempdir = filepath.Join(exporter.tempdir, fmt.Sprintf("serviced-log-export-%s", exporter.timeLabel))
	err = os.Mkdir(exporter.tempdir, 0700)
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

func buildServiceMap(getServices func() ([]service.ServiceDetails, error)) (map[string]service.ServiceDetails, error) {
	result := make(map[string]service.ServiceDetails)
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
func (exporter *logExporter) buildQuery(getServices func() ([]service.ServiceDetails, error)) (string, error) {
	query := "*"
	if len(exporter.ServiceIDs) > 0 {
		services, e := getServices()
		if e != nil {
			return "", e
		}
		serviceMap := make(map[string]service.ServiceDetails)
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
				if found || srvc.ParentServiceID == "" || exporter.ExcludeChildren {
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

		query = fmt.Sprintf("fields.service:(%s)", strings.Join(queryParts, " OR "))
	}

	if len(exporter.FileNames) > 0 {
		for i, filename := range exporter.FileNames {
			exporter.FileNames[i] = strconv.Quote(filename)
		}
		fileQuery := fmt.Sprintf("file:(%s)", strings.Join(exporter.FileNames, " OR "))
		if len(exporter.ServiceIDs) > 0 {
			query = fmt.Sprintf("%s AND %s", query, fileQuery)
		} else {
			query = fileQuery
		}
	}

	if exporter.Debug {
		log.WithFields(logrus.Fields{
			"servicecount": len(exporter.ServiceIDs),
			"filecount":    len(exporter.FileNames),
			"query":        query,
		}).Info("Looking for log messages")
	}
	return query, nil
}

// Retrieve log data from ES-logstash, writing the log messages to separate files based on the group-by criteria.
func (exporter *logExporter) retrieveLogs() (foundIndexedDay bool, numWarnings int, e error) {
	numWarnings = 0
	foundIndexedDay = false
	// fileIndex is a map of parsedMessage to an index into exporter.outputFiles
	exporter.fileIndex = NewFileIndex(exporter.GroupBy)
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
		log := log.WithFields(logrus.Fields{
			"date": yyyymmdd,
		})
		log.Info("Retrieving Logstash Elastic results for one day")

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
		remaining := result.Hits.Total.Value > 0
		for remaining {
			result, e = exporter.Driver.ScrollSearch(result.ScrollID)
			if e != nil {
				return foundIndexedDay, numWarnings, e
			}
			hits := result.Hits.Hits
			for _, hit := range hits {
				message, e := parseLogSource(yyyymmdd, *hit.Source)
				if e != nil {
					return foundIndexedDay, numWarnings, e
				}

				// Ignore log messages sent directly from serviced and serviced-controller; only
				// export logs from the applications themselves.  Note that filebeat is hard-coded
				// to set the _type property to "log" and we only use filebeat for messages from the
				// application services.
				if message.Type != "log" {
					continue
				}

				if len(exporter.minDateFound) == 0 || yyyymmdd < exporter.minDateFound {
					exporter.minDateFound = yyyymmdd
				}
				if len(exporter.maxDateFound) == 0 || yyyymmdd > exporter.maxDateFound {
					exporter.maxDateFound = yyyymmdd
				}

				// add a new tempfile
				if _, found := exporter.fileIndex.FindIndexForMessage(message); !found {
					index := len(exporter.outputFiles)
					filename := filepath.Join(exporter.tempdir, exporter.fileIndex.GetFileName(index, message))
					file, e := os.Create(filename)
					if e != nil {
						return foundIndexedDay, numWarnings, fmt.Errorf("failed to create file %s: %s", filename, e)
					}
					exporter.fileIndex.AddIndexForMessage(index, message)
					outputFile := outputFileInfo{
						File:        file,
						Name:        filename,
						HostID:      message.HostID,
						ContainerID: message.ContainerID,
						LogFileName: message.LogFileName,
						ServiceID:   message.ServiceID,
					}
					exporter.outputFiles = append(exporter.outputFiles, outputFile)
				}
				index, _ := exporter.fileIndex.FindIndexForMessage(message)
				exporter.outputFiles[index].LineCount += len(message.Lines)
				file := exporter.outputFiles[index].File
				filename := exporter.outputFiles[index].Name
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

type FileIndex interface {
	FindIndexForMessage(message *parsedMessage) (index int, found bool)
	AddIndexForMessage(index int, message *parsedMessage)
	GetFileIndexData(index int, outputFile outputFileInfo, hostName, serviceName string) string
	GetIndexHeader() string
	GetFileName(index int, message *parsedMessage) string
}

func NewFileIndex(groupBy ExportGroup) FileIndex {
	var fileIndex FileIndex
	switch groupBy {
	case GroupByDay:
		fileIndex = NewDateFileIndex()
	case GroupByService:
		fileIndex = NewServiceFileIndex()
	case GroupByContainerID:
		fileIndex = NewContainerFileIndex()
	}
	return fileIndex
}

// ContainerFileIndex maintains an index of separate files based on each unique combination of container ID and
// application log file name. Depending on the lifespan of the container and the date range
// defined by exporter.days, some of the files created by this method may have messages from multiple dates.
type ContainerFileIndex struct {
	fileIndex     map[string]map[string]int
	outputPattern string
}

func NewContainerFileIndex() *ContainerFileIndex {
	return &ContainerFileIndex{
		fileIndex:     make(map[string]map[string]int),
		outputPattern: "%-9.9s  %-11.11s  %-36.36s  %-10.10s  %-13.13s  %-20.20s  %-26.26s  %s",
	}
}

func (cfi *ContainerFileIndex) FindIndexForMessage(message *parsedMessage) (index int, found bool) {
	if _, found := cfi.fileIndex[message.ContainerID]; !found {
		cfi.fileIndex[message.ContainerID] = make(map[string]int)
	}
	index, found = cfi.fileIndex[message.ContainerID][message.LogFileName]
	return index, found
}

func (cfi *ContainerFileIndex) AddIndexForMessage(index int, message *parsedMessage) {
	cfi.fileIndex[message.ContainerID][message.LogFileName] = index
}

func (cfi *ContainerFileIndex) GetFileIndexData(index int, outputFile outputFileInfo, hostName, serviceName string) string {
	return fmt.Sprintf(cfi.outputPattern,
		filepath.Base(outputFile.Name),
		strconv.Itoa(outputFile.LineCount),
		hostName,
		outputFile.HostID,
		outputFile.ContainerID,
		strconv.Quote(serviceName),
		outputFile.ServiceID,
		outputFile.LogFileName)
}

func (cfi *ContainerFileIndex) GetIndexHeader() string {
	return fmt.Sprintf(cfi.outputPattern,
		"File",
		"Line Count",
		"Host Name",
		"Host ID",
		"Container ID",
		"Service Name",
		"Service ID",
		"Original Filename")
}

func (cfi *ContainerFileIndex) GetFileName(index int, message *parsedMessage) string {
	return fmt.Sprintf("%04d.log", index)
}

// DateFileIndex maintains an index of separate files based on date. Therefore, a single output file will contain
// messages from multiple services.
type DateFileIndex struct {
	fileIndex     map[string]int
	outputPattern string
}

func NewDateFileIndex() *DateFileIndex {
	return &DateFileIndex{
		fileIndex:     make(map[string]int),
		outputPattern: "%-15.15s  %11.11s  %s",
	}
}

func (dfi *DateFileIndex) FindIndexForMessage(message *parsedMessage) (index int, found bool) {
	index, found = dfi.fileIndex[message.Date]
	return index, found
}

func (dfi *DateFileIndex) AddIndexForMessage(index int, message *parsedMessage) {
	dfi.fileIndex[message.Date] = index
}

func (dfi *DateFileIndex) GetFileIndexData(index int, outputFile outputFileInfo, hostName, serviceName string) string {
	return fmt.Sprintf(dfi.outputPattern,
		filepath.Base(outputFile.Name),
		strconv.Itoa(outputFile.LineCount),
		dfi.findDateForIndex(index))
}

func (dfi *DateFileIndex) GetIndexHeader() string {
	return fmt.Sprintf(dfi.outputPattern, "File", "Line Count", "Date")
}

func (dfi *DateFileIndex) GetFileName(index int, message *parsedMessage) string {
	return fmt.Sprintf("%s.log", message.Date)
}

func (dfi *DateFileIndex) findDateForIndex(index int) string {
	var date string
	for key, value := range dfi.fileIndex {
		if value == index {
			date = key
			break
		}
	}
	return date
}

// ServiceFileIndex maintains an index of separate files based on service id. Therefore, a single output file will
// contain messages for all instances of single service.
type ServiceFileIndex struct {
	fileIndex     map[string]int
	outputPattern string
}

func NewServiceFileIndex() *ServiceFileIndex {
	return &ServiceFileIndex{
		fileIndex:     make(map[string]int),
		outputPattern: "%-30.30s  %11.11s  %-20.20s  %s",
	}
}

func (sfi *ServiceFileIndex) FindIndexForMessage(message *parsedMessage) (index int, found bool) {
	index, found = sfi.fileIndex[message.ServiceID]
	return index, found
}

func (sfi *ServiceFileIndex) AddIndexForMessage(index int, message *parsedMessage) {
	sfi.fileIndex[message.ServiceID] = index
}

func (sfi *ServiceFileIndex) GetFileIndexData(index int, outputFile outputFileInfo, hostName, serviceName string) string {
	return fmt.Sprintf(sfi.outputPattern,
		filepath.Base(outputFile.Name),
		strconv.Itoa(outputFile.LineCount),
		strconv.Quote(serviceName),
		outputFile.ServiceID)
}

func (sfi *ServiceFileIndex) GetIndexHeader() string {
	return fmt.Sprintf(sfi.outputPattern, "File", "Line Count", "Service Name", "Service ID")
}

func (sfi *ServiceFileIndex) GetFileName(index int, message *parsedMessage) string {
	return fmt.Sprintf("%s.log", message.ServiceID)
}

// Organize and cleanup the log messages retrieved from logstash.
//
// The data returned by ES-logstash is not necessarily ordered by time.  The output files created by retrieveLogs()
// are formatted as "<LogstashTimestamp> <offset> <logMessage>". This method will sort the output file content by
// timestamp and offset, and then truncate those fields from the output file such that all that remains are the
// application log messages ordered from oldest first to newest last.
func (exporter *logExporter) organizeAndGroomLogFile(fileIndex, numWarnings int) error {
	filename := exporter.outputFiles[fileIndex].Name
	logName := filepath.Base(filename)
	tmpfilename := filepath.Join(exporter.tempdir, fmt.Sprintf("%s.tmp", logName))

	// Use the -u option to filter out duplicates.
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

// beatProps are properties added to each message by filebeat itself
type beatProps struct {
	Name     interface{} `json:"name"`
	Hostname interface{} `json:"hostname"` // Note this is actually the docker container id
	Version  string      `json:"version"`
}

// fieldProps are properties added to each message by our container controller; see container/logstash.go
type fieldProps struct {
	CCWorkerID  interface{} `json:"ccWorkerID"` // Note this is actually the host ID of the CC host
	Type        interface{} `json:"type"`       // This is the 'type' from the LogConfig in the service def
	Service     interface{} `json:"service"`    // This is the service id
	Instance    interface{} `json:"instance"`   // This is the service instance id
	HostIPs     interface{} `json:"hostips"`    // space-separated list of host-ips from the container
	PoolID      interface{} `json:"poolid"`
	ServicePath interface{} `json:"servicepath"` // Fully qualified path to the service
}

func IDsToString(ids interface{}) string {
	var idsStr string
	switch ids.(type) {
	case []string:
		idsStr = strings.Join(ids.([]string)[:], "_")
	case string:
		idsStr = ids.(string)
	default:
		idsStr = ""
	}
	return idsStr
}

// logSingleLine represents the data returned from elasticsearch for a single-line log message
type logSingleLine struct {
	File      string      `json:"file"`
	Timestamp time.Time   `json:"@timestamp"`
	Offset    json.Number `json:"offset"`
	Message   string      `json:"message"`
	Fields    fieldProps  `json:"fields"`
	FileBeat  beatProps   `json:"beat"`

	// This is the 'type' set by the logger. The values will vary depending on the source logger:
	// 1. Application log messages forwarded by filebeat will have the value "log".
	// 2. Messages forwarded directly from serviced via the glog library will have values starting with "serviced-".
	// 3. Messages forwarded directly from serviced-controller via the glog library will have values starting
	//    with "controller-"
	Type string `json:"type"`
}

// logSingleLine represents the data returned from elasticsearch for a multi-line log message
type logMultiLine struct {
	Type      string     `json:"type"` // see note above for logSingleLine.Type
	File      string     `json:"file"`
	Timestamp time.Time  `json:"@timestamp"`
	Offsets   []uint64   `json:"offset"`
	Messages  []string   `json:"message"`
	Fields    fieldProps `json:"fields"`
	FileBeat  beatProps  `json:"beat"`
}

type logMultiLineGeneric struct {
	Type      string      `json:"type"` // see note above for logSingleLine.Type
	File      string      `json:"file"`
	Timestamp time.Time   `json:"@timestamp"`
	Offsets   interface{} `json:"offset"`
	Messages  interface{} `json:"message"`
	Fields    fieldProps  `json:"fields"`
	FileBeat  beatProps   `json:"beat"`
}

type compactLogLine struct {
	Timestamp int64 // nanoseconds since the epoch, truncated at the minute to hide jitter
	Offset    uint64
	Message   string
}

// Represents a set of log messages from a since instance of a single application log
type parsedMessage struct {
	Date        string           // The date for the message in the format YYYYMMDD
	HostID      string           // The ID of the CC worker node that hosted ContainerID
	ContainerID string           // The original Container ID of the application log file
	LogFileName string           // The name of the application log file
	Lines       []compactLogLine // One or more lines of log messages from the file.
	Warnings    string           // Warnings from the our logstash results parser
	ServiceID   string           // The service ID of the service
	Type        string           // The type of the source log file.
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

// The ccworkerid is a number on some systems that didn't have
// the filter mutate, and the hostid was a numeric value.  We need
// to handle both cases.
func convertWorkerID(id interface{}) (string, error) {
	if id == nil {
		return "", nil
	}

	switch id.(type) {
	case int:
		num := id.(int)
		return fmt.Sprintf("%v", num), nil
	case uint64:
		num := id.(uint64)
		return fmt.Sprintf("%v", num), nil
	case float64:
		num := id.(float64)
		return fmt.Sprintf("%.0f", num), nil
	case json.Number:
		return string(id.(json.Number)), nil
	case string:
		return id.(string), nil
	case []string:
		return strings.Join(id.([]string)[:], "_"), nil
	case []interface{}:
		datalist := id.([]interface{})
		ids := make([]string, len(datalist))
		for i, item := range datalist {
			ids[i] = fmt.Sprint(item)
		}
		return strings.Join(ids[:], "_"), nil
	}

	// An unexpected type. We'll get the type name for the error, but this
	// will fail the export.. so it only uses reflect once.
	name := reflect.TypeOf(id)
	return "", &UnknownElasticStructError{fmt.Sprintf("%v", name), "ccworkerid"}
}

// Convert the interface{} to uint64 from uint64/float64/json.Number
func convertOffset(offset interface{}) (uint64, error) {
	switch offset.(type) {
	case uint64:
		return offset.(uint64), nil
	case float64:
		return uint64(offset.(float64)), nil
	case json.Number:
		return strconv.ParseUint(string(offset.(json.Number)), 10, 64)
	}

	// An unexpected type. We'll get the type name for the error, but this
	// will fail the export.. so it only uses reflect once.
	name := reflect.TypeOf(offset)
	return 0, &UnknownElasticStructError{fmt.Sprintf("%v", name), "offset"}
}

// Converts the generic interface{} offset to []uint64
func convertGenericOffsets(data interface{}) ([]uint64, error) {
	switch data.(type) {
	case uint64, float64, json.Number:
		if value, err := convertOffset(data); err != nil {
			return nil, err
		} else {
			return []uint64{value}, nil
		}
	case []json.Number:
		interfaces := data.([]json.Number)
		offsets := make([]uint64, len(interfaces))
		for i, offset := range interfaces {
			if value, err := convertOffset(offset); err != nil {
				return nil, err
			} else {
				offsets[i] = value
			}
		}
		return offsets, nil
	case []uint64:
		return data.([]uint64), nil
	case []float64:
		interfaces := data.([]float64)
		offsets := make([]uint64, len(interfaces))
		for i, offset := range interfaces {
			if value, err := convertOffset(offset); err != nil {
				return nil, err
			} else {
				offsets[i] = value
			}
		}
		return offsets, nil
	case []interface{}: // The array may not be typed.
		interfaces := data.([]interface{})
		offsets := make([]uint64, len(interfaces))
		for i, offset := range interfaces {
			if value, err := convertOffset(offset); err != nil {
				return nil, err
			} else {
				offsets[i] = value
			}
		}
		return offsets, nil
	}

	// An unexpected type. We'll get the type name for the error, but this
	// will fail the export.. so it only uses reflect once.
	name := reflect.TypeOf(data)
	return nil, &UnknownElasticStructError{fmt.Sprintf("%v", name), "offset"}
}

// convert a single message
func convertMessage(data interface{}) (string, error) {
	switch data.(type) {
	case string:
		return data.(string), nil
	}

	// An unexpected type. We'll get the type name for the error, but this
	// will fail the export.. so it only uses reflect once.
	name := reflect.TypeOf(data)
	return "", &UnknownElasticStructError{fmt.Sprintf("%v", name), "message"}
}

// Converts the generic interface{} message to an array of strings.
func convertGenericMessages(data interface{}) ([]string, error) {
	switch t := data.(type) {
	case string:
		return newline.Split(t, -1), nil
	case []string:
		return t, nil
	case []interface{}:
		// Untyped array.
		datalist := data.([]interface{})
		messages := make([]string, len(datalist))
		for i, message := range datalist {
			if value, err := convertMessage(message); err != nil {
				return nil, err
			} else {
				messages[i] = value
			}
		}
		return messages, nil
	}

	// An unexpected type. We'll get the type name for the error, but this
	// will fail the export.. so it only uses reflect once.
	name := reflect.TypeOf(data)
	return nil, &UnknownElasticStructError{fmt.Sprintf("%v", name), "messages"}
}

// Converts the elastic log into a generic type and converts that into a
// common format to use in the export.
func convertMultiLineSource(source []byte) (logMultiLine, error) {
	var multiLine logMultiLine
	var multiLineGeneric logMultiLineGeneric

	if err := json.Unmarshal(source, &multiLineGeneric); err == nil {
		multiLine.Type = multiLineGeneric.Type
		multiLine.File = multiLineGeneric.File
		multiLine.Timestamp = multiLineGeneric.Timestamp
		multiLine.Fields = multiLineGeneric.Fields
		multiLine.FileBeat = multiLineGeneric.FileBeat

		// Try to get the offsets and messages.
		if offsets, err := convertGenericOffsets(multiLineGeneric.Offsets); err != nil {
			return logMultiLine{}, err
		} else {
			multiLine.Offsets = offsets
		}
		if messages, err := convertGenericMessages(multiLineGeneric.Messages); err != nil {
			return logMultiLine{}, err
		} else {
			multiLine.Messages = messages
		}

		return multiLine, nil
	} else {
		return logMultiLine{}, err
	}
}

// return: containerID, file, lines, error
func parseLogSource(date string, source []byte) (*parsedMessage, error) {
	warnings := ""

	// attempt to unmarshal into singleLine
	var line logSingleLine

	if e := json.Unmarshal(source, &line); e == nil {
		offset := uint64(0)
		if len(line.Offset) != 0 {
			var e error
			offset, e = strconv.ParseUint(string(line.Offset), 10, 64)
			if e != nil {
				return nil, fmt.Errorf("failed to parse offset \"%s\" in \"%s\": %s", line.Offset, source, e)
			}
		}
		compactLine := compactLogLine{
			Timestamp: truncateToMinute(line.Timestamp.UnixNano()),
			Offset:    offset,
			Message:   line.Message,
		}

		// Get the ccWorkerID. This can be a number in older systems, so we have to parse it out.
		ccWorkerID, err := convertWorkerID(line.Fields.CCWorkerID)
		if err != nil {
			return nil, fmt.Errorf("failed to parse ccWorkerID from %v in \"%s\": %s", line.Fields.CCWorkerID, source, err)
		}

		message := &parsedMessage{
			Date:        date,
			Type:        line.Type,
			HostID:      ccWorkerID,
			ContainerID: IDsToString(line.FileBeat.Hostname),
			LogFileName: line.File,
			Lines:       []compactLogLine{compactLine},
			ServiceID:   IDsToString(line.Fields.Service),
		}
		return message, nil
	}

	// attempt to unmarshal into multiLine
	multiLine, err := convertMultiLineSource(source)
	if err != nil {
		return nil, fmt.Errorf("failed to parse multiLine log from JSON \"%s\": %s", source, err)
	}

	// build offsets - list of uint64
	offsets := multiLine.Offsets

	// verify number of lines in message against number of offsets
	messages := multiLine.Messages
	if len(offsets)+1 == len(messages) {
		warnings += fmt.Sprintf(
			"number of offsets for %s:%s (numLines:%d numOffsets:%d) is one less than number of lines: %s\n",
			multiLine.FileBeat.Hostname, multiLine.File, len(messages), len(offsets), source)
		numLines := len(messages)
		if numLines > 1 {
			lastOffset := uint64(len(messages[numLines-2])) + offsets[numLines-2]
			offsets = append(offsets, lastOffset)
		}
	} else if len(offsets) > len(messages) {
		warnings += fmt.Sprintf(
			"number of offsets for %s:%s (numLines:%d numOffsets:%d) is greater than number of lines: %s\n",
			multiLine.FileBeat.Hostname, multiLine.File, len(messages), len(multiLine.Offsets), source)
		offsets = offsets[0:len(messages)]
	} else if len(offsets) < len(messages) {
		warnings += fmt.Sprintf(
			"number of offsets for %s:%s (numLines:%d numOffsets:%d) is less than number of lines: %s\n",
			multiLine.FileBeat.Hostname, multiLine.File, len(messages), len(multiLine.Offsets), source)
		offsets = generateOffsets(messages, offsets)
		warnings += fmt.Sprintf("new offsets: %v\n", offsets)
	}

	// deal with offsets that are not sorted in increasing order
	if !uint64sAreSorted(offsets) {
		warnings += fmt.Sprintf("offsets are not sorted: %v\n", offsets)
		offsets = generateOffsets(messages, offsets)
		warnings += fmt.Sprintf("new offsets: %v\n", offsets)
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

	// Get the ccWorkerID. This can be a number in older systems, so we have to parse it out.
	ccWorkerID, err := convertWorkerID(multiLine.Fields.CCWorkerID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ccWorkerID from %v in \"%s\": %s", multiLine.Fields.CCWorkerID, source, err)
	}

	message := &parsedMessage{
		Date:        date,
		Type:        line.Type,
		HostID:      ccWorkerID,
		ContainerID: IDsToString(multiLine.FileBeat.Hostname),
		LogFileName: multiLine.File,
		Lines:       compactLines,
		Warnings:    warnings,
		ServiceID:   IDsToString(multiLine.Fields.Service),
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
